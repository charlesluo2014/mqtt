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

// The CONNACK Packet is the packet sent by the Server in response to a CONNECT Packet
// received from a Client. The first packet sent from the Server to the Client MUST
// be a CONNACK Packet [MQTT-3.2.0-1].
//
// If the Client does not receive a CONNACK Packet from the Server within a reasonable
// amount of time, the Client SHOULD close the Network Connection. A "reasonable" amount
// of time depends on the type of application and the communications infrastructure.
type ConnackMessage struct {
	fixedHeader

	sessionPresent bool
	returnCode     ConnackCode
}

var _ Message = (*ConnackMessage)(nil)

// NewConnackMessage creates a new CONNACK message
func NewConnackMessage() *ConnackMessage {
	msg := &ConnackMessage{}
	msg.SetType(CONNACK)

	return msg
}

// String returns a string representation of the CONNACK message
func (this ConnackMessage) String() string {
	return fmt.Sprintf("%v\nSession Present: %t\nReturn code: %v\n",
		this.fixedHeader, this.sessionPresent, this.returnCode)
}

// SessionPresent returns the session present flag value
func (this *ConnackMessage) SessionPresent() bool {
	return this.sessionPresent
}

// SetSessionPresent sets the value of the session present flag
func (this *ConnackMessage) SetSessionPresent(v bool) {
	if v {
		this.sessionPresent = true
	} else {
		this.sessionPresent = false
	}
}

// ReturnCode returns the return code received for the CONNECT message. The return
// type is an error
func (this *ConnackMessage) ReturnCode() ConnackCode {
	return this.returnCode
}

func (this *ConnackMessage) SetReturnCode(ret ConnackCode) {
	this.returnCode = ret
}

// Decode reads from the io.Reader parameter until a full message is decoded, or
// when io.Reader returns EOF or error. The first return value is the number of
// bytes read from io.Reader. The second is error if Decode encounters any problems.
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

	this.returnCode = ConnackCode(b)

	return total, nil
}

// Encode returns an io.Reader in which the encoded bytes can be read. The second
// return value is the number of bytes encoded, so the caller knows how many bytes
// there will be. If Encode returns an error, then the first two return values
// should be considered invalid.
// Any changes to the message after Encode() is called will invalidate the io.Reader.
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

	if this.returnCode > 5 {
		return nil, 0, fmt.Errorf("connack/Encode: Invalid CONNACK return code (%d)", this.returnCode)
	}

	b[1] = this.returnCode.Value()

	n, err := this.buf.Write(b[:])
	if err != nil {
		return nil, 0, err
	}
	total += n

	return this.buf, total, nil
}
