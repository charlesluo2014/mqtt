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
	"bytes"
	"fmt"
	"io"
)

// The SUBSCRIBE Packet is sent from the Client to the Server to create one or more
// Subscriptions. Each Subscription registers a Client’s interest in one or more
// Topics. The Server sends PUBLISH Packets to the Client in order to forward
// Application Messages that were published to Topics that match these Subscriptions.
// The SUBSCRIBE Packet also specifies (for each Subscription) the maximum QoS with
// which the Server can send Application Messages to the Client.
type SubscribeMessage struct {
	fixedHeader

	packetId uint16
	topics   [][]byte
	qos      []byte
}

var _ Message = (*SubscribeMessage)(nil)

// NewSubscribeMessage creates a new SUBSCRIBE message.
func NewSubscribeMessage() *SubscribeMessage {
	msg := &SubscribeMessage{}
	msg.SetType(SUBSCRIBE)

	return msg
}

// PacketId returns the ID of the packet.
func (this *SubscribeMessage) PacketId() uint16 {
	return this.packetId
}

// SetPacketId sets the ID of the packet.
func (this *SubscribeMessage) SetPacketId(v uint16) {
	this.packetId = v
}

// Topics returns a list of topics sent by the Client.
func (this *SubscribeMessage) Topics() [][]byte {
	return this.topics
}

// AddTopic adds a single topic to the message, along with the corresponding QoS.
// An error is returned if QoS is invalid.
func (this *SubscribeMessage) AddTopic(topic []byte, qos byte) error {
	if !ValidQos(qos) {
		return fmt.Errorf("Invalid QoS %d", qos)
	}

	var i int
	var t []byte
	var found bool

	for i, t = range this.topics {
		if bytes.Equal(t, topic) {
			found = true
			break
		}
	}

	if found {
		this.qos[i] = qos
		return nil
	}

	this.topics = append(this.topics, topic)
	this.qos = append(this.qos, qos)

	return nil
}

// RemoveTopic removes a single topic from the list of existing ones in the message.
// If topic does not exist it just does nothing.
func (this *SubscribeMessage) RemoveTopic(topic []byte) {
	var i int
	var t []byte
	var found bool

	for i, t = range this.topics {
		if bytes.Equal(t, topic) {
			found = true
			break
		}
	}

	if found {
		this.topics = append(this.topics[:i], this.topics[i+1:]...)
		this.qos = append(this.qos[:i], this.qos[i+1:]...)
	}
}

// TopicExists checks to see if a topic exists in the list.
func (this *SubscribeMessage) TopicExists(topic []byte) bool {
	for _, t := range this.topics {
		if bytes.Equal(t, topic) {
			return true
		}
	}

	return false
}

// TopicQos returns the QoS level of a topic. If topic does not exist, QosFailure
// is returned.
func (this *SubscribeMessage) TopicQos(topic []byte) byte {
	for i, t := range this.topics {
		if bytes.Equal(t, topic) {
			return this.qos[i]
		}
	}

	return QosFailure
}

// Qos returns the list of QoS current in the message.
func (this *SubscribeMessage) Qos() []byte {
	return this.qos
}

// Decode reads from the io.Reader parameter until a full message is decoded, or
// when io.Reader returns EOF or error. The first return value is the number of
// bytes read from io.Reader. The second is error if Decode encounters any problems.
func (this *SubscribeMessage) Decode(src io.Reader) (int, error) {
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

	for this.buf.Len() > 0 {
		t, n, err := readLPBytes(this.buf)
		if err != nil {
			return total + n, err
		}
		total += n

		this.topics = append(this.topics, t)

		b, err := this.buf.ReadByte()
		if err != nil {
			return total, err
		}
		total += 1

		this.qos = append(this.qos, b)
	}

	if len(this.topics) == 0 {
		return 0, fmt.Errorf("subscribe/Decode: Empty topic list")
	}

	return total, nil
}

// Encode returns an io.Reader in which the encoded bytes can be read. The second
// return value is the number of bytes encoded, so the caller knows how many bytes
// there will be. If Encode returns an error, then the first two return values
// should be considered invalid.
// Any changes to the message after Encode() is called will invalidate the io.Reader.
func (this *SubscribeMessage) Encode() (io.Reader, int, error) {
	// packet ID
	total := 2

	for _, t := range this.topics {
		total += 2 + len(t) + 1
	}

	this.SetRemainingLength(int32(total))

	total = 0

	_, total, err := this.fixedHeader.Encode()
	if err != nil {
		return nil, total, err
	}

	if err = writeUint16(this.buf, this.packetId); err != nil {
		return nil, total, err
	}
	total += 2

	var n int

	for i, t := range this.topics {
		if n, err = writeLPBytes(this.buf, t); err != nil {
			return nil, total, err
		}
		total += n

		this.buf.WriteByte(this.qos[i])
		total += 1
	}

	return this.buf, total, nil
}
