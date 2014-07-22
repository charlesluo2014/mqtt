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

type SubackMessage struct {
	fixedHeader

	packetId    uint16
	returnCodes []byte
}

var _ Message = (*SubackMessage)(nil)

func NewSubackMessage() *SubackMessage {
	msg := &SubackMessage{}
	msg.SetType(SUBACK)

	return msg
}

func (this SubackMessage) String() string {
	return fmt.Sprintf("%s\nPacket ID: %d\nReturn Codes: %v\n", this.fixedHeader, this.packetId, this.returnCodes)
}

func (this *SubackMessage) PacketId() uint16 {
	return this.packetId
}

func (this *SubackMessage) SetPacketId(v uint16) {
	this.packetId = v
}

func (this *SubackMessage) ReturnCodes() []byte {
	return this.returnCodes
}

func (this *SubackMessage) AddReturnCodes(ret []byte) error {
	for _, c := range ret {
		if c != QosAtMostOnce && c != QosAtLeastOnce && c != QosExactlyOnce && c != QosFailure {
			return fmt.Errorf("suback/AddReturnCode: Invalid return code %d. Must be 0, 1, 2, 0x80.", c)
		}

		this.returnCodes = append(this.returnCodes, c)
	}

	return nil
}

func (this *SubackMessage) AddReturnCode(ret byte) error {
	return this.AddReturnCodes([]byte{ret})
}

func (this *SubackMessage) Decode(src io.Reader) (int, error) {
	total := 0

	n, err := this.fixedHeader.Decode(src)
	if err != nil {
		return total + n, err
	}
	total += n

	if this.packetId, err = readUint16(this.buf); err != nil {
		return 0, err
	}
	total += 2

	this.returnCodes = this.buf.Next(this.buf.Len())
	total += len(this.returnCodes)

	for i, code := range this.returnCodes {
		if code != 0x00 && code != 0x01 && code != 0x02 && code != 0x80 {
			return total, fmt.Errorf("suback/Decode: Invalid return code %d for topic %d", code, i)
		}
	}

	return total, nil
}

func (this *SubackMessage) Encode() (io.Reader, int, error) {
	for i, code := range this.returnCodes {
		if code != 0x00 && code != 0x01 && code != 0x02 && code != 0x80 {
			return nil, 0, fmt.Errorf("suback/Encode: Invalid return code %d for topic %d", code, i)
		}
	}

	this.SetRemainingLength(2 + int32(len(this.returnCodes)))

	_, total, err := this.fixedHeader.Encode()
	if err != nil {
		return nil, 0, err
	}

	if err = writeUint16(this.buf, this.packetId); err != nil {
		return nil, 0, err
	}
	total += 2

	var n int
	if n, err = this.buf.Write(this.returnCodes); err != nil {
		return nil, 0, err
	}
	total += n

	return this.buf, total, nil
}
