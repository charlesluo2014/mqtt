// Copyright (c) 2014 Dataence, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mqtt

import (
	"fmt"
	"io"
)

type ConnackMessage struct {
	fixedHeader

	sessionPresent bool
	returnCode     error
}

var _ Message = (*ConnackMessage)(nil)

func NewConnackMessage() *ConnackMessage {
	msg := &ConnackMessage{}
	msg.SetType(CONNACK)

	return msg
}

func (this ConnackMessage) String() string {
	return fmt.Sprintf("%v\nSession Present: %t\nReturn code: %v\n",
		this.fixedHeader, this.sessionPresent, this.returnCode)
}

func (this *ConnackMessage) SessionPresent() bool {
	return this.sessionPresent
}

func (this *ConnackMessage) SetSessionPresent(v bool) {
	if v {
		this.sessionPresent = true
	} else {
		this.sessionPresent = false
	}
}

func (this *ConnackMessage) ReturnCode() error {
	return this.returnCode
}

func (this *ConnackMessage) SetReturnCode(ret error) {
	this.returnCode = ret
}

func (this *ConnackMessage) Decode(src io.Reader) (int, error) {
	total := 0

	n, err := this.fixedHeader.Decode(src)
	if err != nil {
		return total + n, err
	}
	total += n

	var b byte

	// Read session present flag
	if b, err = this.buf.ReadByte(); err != nil {
		return total, err
	}
	total += 1

	if b&254 != 0 {
		return 0, fmt.Errorf("connack/Decode: Bits 7-1 in Connack Acknowledge Flags byte (1) are not 0")
	}

	this.sessionPresent = b&0x1 == 1

	// Read return code
	if b, err = this.buf.ReadByte(); err != nil {
		return total, err
	}
	total += 1

	if b > 5 {
		return 0, fmt.Errorf("connack/Decode: Invalid CONNACK return code (%d)", b)
	}

	this.returnCode = ConnackReturnCodes[b]

	return total, nil
}

func (this *ConnackMessage) Encode() (io.Reader, int, error) {
	// CONNACK remaining length fixed at 2 bytes
	this.SetRemainingLength(2)

	_, total, err := this.fixedHeader.Encode()
	if err != nil {
		return nil, 0, err
	}

	var b [2]byte
	if this.sessionPresent {
		b[0] = 1
	}

	retcode, ok := this.returnCode.(ConnackReturnCode)
	if !ok {
		return nil, 0, fmt.Errorf("connack/Encode: Invalid return code")
	}

	if retcode.code > 5 {
		return nil, 0, fmt.Errorf("connack/Encode: Invalid CONNACK return code (%d)", retcode.code)
	}

	b[1] = retcode.code
	var n int

	if n, err = this.buf.Write(b[:]); err != nil {
		return nil, 0, err
	}

	total += n

	return this.buf, total, nil
}
