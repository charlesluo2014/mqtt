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

// A SUBACK Packet is sent by the Server to the Client to confirm receipt and processing
// of a SUBSCRIBE Packet.
//
// A SUBACK Packet contains a list of return codes, that specify the maximum QoS level
// that was granted in each Subscription that was requested by the SUBSCRIBE.
type SubackMessage struct {
	fixedHeader

	packetId    uint16
	returnCodes []byte
}

var _ Message = (*SubackMessage)(nil)

// NewSubackMessage creates a new SUBACK message.
func NewSubackMessage() *SubackMessage {
	msg := &SubackMessage{}
	msg.SetType(SUBACK)

	return msg
}

// String returns a string representation of the message.
func (this SubackMessage) String() string {
	return fmt.Sprintf("%s\nPacket ID: %d\nReturn Codes: %v\n", this.fixedHeader, this.packetId, this.returnCodes)
}

// PacketId returns the ID of the packet.
func (this *SubackMessage) PacketId() uint16 {
	return this.packetId
}

// SetPacketId sets the ID of the packet.
func (this *SubackMessage) SetPacketId(v uint16) {
	this.packetId = v
}

// ReturnCodes returns the list of QoS returns from the subscriptions sent in the SUBSCRIBE message.
func (this *SubackMessage) ReturnCodes() []byte {
	return this.returnCodes
}

// AddReturnCodes sets the list of QoS returns from the subscriptions sent in the SUBSCRIBE message.
// An error is returned if any of the QoS values are not valid.
func (this *SubackMessage) AddReturnCodes(ret []byte) error {
	for _, c := range ret {
		if c != QosAtMostOnce && c != QosAtLeastOnce && c != QosExactlyOnce && c != QosFailure {
			return fmt.Errorf("suback/AddReturnCode: Invalid return code %d. Must be 0, 1, 2, 0x80.", c)
		}

		this.returnCodes = append(this.returnCodes, c)
	}

	return nil
}

// AddReturnCode adds a single QoS return value.
func (this *SubackMessage) AddReturnCode(ret byte) error {
	return this.AddReturnCodes([]byte{ret})
}

// Decode reads from the io.Reader parameter until a full message is decoded, or
// when io.Reader returns EOF or error. The first return value is the number of
// bytes read from io.Reader. The second is error if Decode encounters any problems.
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

// Encode returns an io.Reader in which the encoded bytes can be read. The second
// return value is the number of bytes encoded, so the caller knows how many bytes
// there will be. If Encode returns an error, then the first two return values
// should be considered invalid.
// Any changes to the message after Encode() is called will invalidate the io.Reader.
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
