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

type SubscribeMessage struct {
	fixedHeader

	packetId uint16
	topics   [][]byte
	qos      []byte
}

var _ Message = (*SubscribeMessage)(nil)

func NewSubscribeMessage() *SubscribeMessage {
	msg := &SubscribeMessage{}
	msg.SetType(SUBSCRIBE)

	return msg
}

func (this *SubscribeMessage) PacketId() uint16 {
	return this.packetId
}

func (this *SubscribeMessage) SetPacketId(v uint16) {
	this.packetId = v
}

func (this *SubscribeMessage) Topics() [][]byte {
	return this.topics
}

func (this *SubscribeMessage) AddTopic(topic []byte, qos byte) {
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
		return
	}

	this.topics = append(this.topics, topic)
	this.qos = append(this.qos, qos)
}

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

func (this *SubscribeMessage) TopicExists(topic []byte) bool {
	for _, t := range this.topics {
		if bytes.Equal(t, topic) {
			return true
		}
	}

	return false
}

func (this *SubscribeMessage) TopicQos(topic []byte) byte {
	for i, t := range this.topics {
		if bytes.Equal(t, topic) {
			return this.qos[i]
		}
	}

	return QosFailure
}

func (this *SubscribeMessage) Qos() []byte {
	return this.qos
}

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
