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

type PublishMessage struct {
	fixedHeader

	packetId uint16
	topic    []byte
	payload  []byte
}

var _ Message = (*PublishMessage)(nil)

func NewPublishMessage() *PublishMessage {
	msg := &PublishMessage{}
	msg.SetType(PUBLISH)

	return msg
}

func (this PublishMessage) String() string {
	return fmt.Sprintf("%v\nTopic: %s\nPacket ID: %d\nPayload: %s\n",
		this.fixedHeader, this.topic, this.packetId, string(this.payload))
}

func (this *PublishMessage) Dup() bool {
	return ((this.flags >> 3) & 0x1) == 1
}

func (this *PublishMessage) SetDup(v bool) {
	if v {
		this.flags |= 0x8 // 00001000
	} else {
		this.flags &= 247 // 11110111
	}
}

func (this *PublishMessage) Retain() bool {
	return (this.flags & 0x1) == 1
}

func (this *PublishMessage) SetRetain(v bool) {
	if v {
		this.flags |= 0x1 // 00000001
	} else {
		this.flags &= 254 // 11111110
	}
}

func (this *PublishMessage) QoS() byte {
	return (this.flags >> 1) & 0x3
}

func (this *PublishMessage) SetQoS(v byte) error {
	if v != 0x0 && v != 0x1 && v != 0x2 {
		return fmt.Errorf("publish/SetQoS: Invalid QoS %d.", v)
	}

	this.flags = (this.flags & 249) | (v << 1) // 243 = 11111001
	return nil
}

func (this *PublishMessage) Topic() []byte {
	return this.topic
}

func (this *PublishMessage) SetTopic(v []byte) error {
	if !ValidTopic(v) {
		return fmt.Errorf("publish/SetTopic: Invalid topic name (%s). Must not be empty or contain wildcard characters", string(v))
	}

	this.topic = v
	return nil
}

func (this *PublishMessage) PacketId() uint16 {
	return this.packetId
}

func (this *PublishMessage) SetPacketId(v uint16) {
	this.packetId = v
}

func (this *PublishMessage) Payload() []byte {
	return this.payload
}

func (this *PublishMessage) SetPayload(v []byte) {
	this.payload = v
}

func (this *PublishMessage) Decode(src io.Reader) (int, error) {
	total := 0

	n, err := this.fixedHeader.Decode(src)
	if err != nil {
		return total + n, err
	}
	total += n

	if this.topic, n, err = readLPBytes(this.buf); err != nil {
		return total + n, err
	}
	total += n

	if !ValidTopic(this.topic) {
		return total, fmt.Errorf("publish/Decode: Invalid topic name (%s). Must not be empty or contain wildcard characters", string(this.topic))
	}

	// The packet identifier field is only present in the PUBLISH packets where the
	// QoS level is 1 or 2
	if this.QoS() != 0 {
		if this.packetId, err = readUint16(this.buf); err != nil {
			return 0, err
		}
		total += 2
	}

	this.payload = this.buf.Next(this.buf.Len())
	total += len(this.payload)

	return total, nil
}

func (this *PublishMessage) Encode() (io.Reader, int, error) {
	if len(this.topic) == 0 {
		return nil, 0, fmt.Errorf("publish/Encode: Topic name is empty.")
	}

	if len(this.payload) == 0 {
		return nil, 0, fmt.Errorf("publish/Encode: Payload is empty.")
	}

	total := 2 + len(this.topic) + len(this.payload)
	if this.QoS() != 0 {
		total += 2
	}
	this.SetRemainingLength(int32(total))

	total = 0

	_, n, err := this.fixedHeader.Encode()
	if err != nil {
		return nil, total, err
	}
	total += n

	if n, err = writeLPBytes(this.buf, this.topic); err != nil {
		return nil, total, err
	}
	total += n

	// The packet identifier field is only present in the PUBLISH packets where the QoS level is 1 or 2
	if this.QoS() != 0 {
		if err = writeUint16(this.buf, this.packetId); err != nil {
			return nil, total, err
		}
		total += 2
	}

	if n, err = this.buf.Write(this.payload); err != nil {
		return nil, total, err
	}
	total += n

	return this.buf, total, nil
}
