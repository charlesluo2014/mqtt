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

// A PUBLISH Control Packet is sent from a Client to a Server or from Server to a Client
// to transport an Application Message.
type PublishMessage struct {
	fixedHeader

	packetId uint16
	topic    []byte
	payload  []byte
}

var _ Message = (*PublishMessage)(nil)

// NewPublishMessage creates a new PUBLISH message.
func NewPublishMessage() *PublishMessage {
	msg := &PublishMessage{}
	msg.SetType(PUBLISH)

	return msg
}

func (this PublishMessage) String() string {
	return fmt.Sprintf("%v\nTopic: %s\nPacket ID: %d\nPayload: %s\n",
		this.fixedHeader, this.topic, this.packetId, string(this.payload))
}

// Dup returns the value specifying the duplicate delivery of a PUBLISH Control Packet.
// If the DUP flag is set to 0, it indicates that this is the first occasion that the
// Client or Server has attempted to send this MQTT PUBLISH Packet. If the DUP flag is
// set to 1, it indicates that this might be re-delivery of an earlier attempt to send
// the Packet.
func (this *PublishMessage) Dup() bool {
	return ((this.flags >> 3) & 0x1) == 1
}

// SetDup sets the value specifying the duplicate delivery of a PUBLISH Control Packet.
func (this *PublishMessage) SetDup(v bool) {
	if v {
		this.flags |= 0x8 // 00001000
	} else {
		this.flags &= 247 // 11110111
	}
}

// Retain returns the value of the RETAIN flag. This flag is only used on the PUBLISH
// Packet. If the RETAIN flag is set to 1, in a PUBLISH Packet sent by a Client to a
// Server, the Server MUST store the Application Message and its QoS, so that it can be
// delivered to future subscribers whose subscriptions match its topic name.
func (this *PublishMessage) Retain() bool {
	return (this.flags & 0x1) == 1
}

// SetRetain sets the value of the RETAIN flag.
func (this *PublishMessage) SetRetain(v bool) {
	if v {
		this.flags |= 0x1 // 00000001
	} else {
		this.flags &= 254 // 11111110
	}
}

// QoS returns the field that indicates the level of assurance for delivery of an
// Application Message. The values are QosAtMostOnce, QosAtLeastOnce and QosExactlyOnce.
func (this *PublishMessage) QoS() byte {
	return (this.flags >> 1) & 0x3
}

// SetQoS sets the field that indicates the level of assurance for delivery of an
// Application Message. The values are QosAtMostOnce, QosAtLeastOnce and QosExactlyOnce.
// An error is returned if the value is not one of these.
func (this *PublishMessage) SetQoS(v byte) error {
	if v != 0x0 && v != 0x1 && v != 0x2 {
		return fmt.Errorf("publish/SetQoS: Invalid QoS %d.", v)
	}

	this.flags = (this.flags & 249) | (v << 1) // 243 = 11111001
	return nil
}

// Topic returns the the topic name that identifies the information channel to which
// payload data is published.
func (this *PublishMessage) Topic() []byte {
	return this.topic
}

// SetTopic sets the the topic name that identifies the information channel to which
// payload data is published. An error is returned if ValidTopic() is falbase.
func (this *PublishMessage) SetTopic(v []byte) error {
	if !ValidTopic(v) {
		return fmt.Errorf("publish/SetTopic: Invalid topic name (%s). Must not be empty or contain wildcard characters", string(v))
	}

	this.topic = v
	return nil
}

// PacketId returns the ID of the packet. It is only present in PUBLISH Packets where
// the QoS level is 1 or 2.
func (this *PublishMessage) PacketId() uint16 {
	return this.packetId
}

// SetPacketId sets the ID of the packet.
func (this *PublishMessage) SetPacketId(v uint16) {
	this.packetId = v
}

// Payload returns the application message that's part of the PUBLISH message.
func (this *PublishMessage) Payload() []byte {
	return this.payload
}

// SetPayload sets the application message that's part of the PUBLISH message.
func (this *PublishMessage) SetPayload(v []byte) {
	this.payload = v
}

// Decode reads from the io.Reader parameter until a full message is decoded, or
// when io.Reader returns EOF or error. The first return value is the number of
// bytes read from io.Reader. The second is error if Decode encounters any problems.
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

// Encode returns an io.Reader in which the encoded bytes can be read. The second
// return value is the number of bytes encoded, so the caller knows how many bytes
// there will be. If Encode returns an error, then the first two return values
// should be considered invalid.
// Any changes to the message after Encode() is called will invalidate the io.Reader.
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
