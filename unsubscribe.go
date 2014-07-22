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

type UnsubscribeMessage struct {
	fixedHeader

	packetId uint16
	topics   [][]byte
}

var _ Message = (*UnsubscribeMessage)(nil)

func NewUnsubscribeMessage() *UnsubscribeMessage {
	msg := &UnsubscribeMessage{}
	msg.SetType(UNSUBSCRIBE)

	return msg
}

func (this *UnsubscribeMessage) PacketId() uint16 {
	return this.packetId
}

func (this *UnsubscribeMessage) SetPacketId(v uint16) {
	this.packetId = v
}

func (this *UnsubscribeMessage) Topics() [][]byte {
	return this.topics
}

func (this *UnsubscribeMessage) AddTopic(topic []byte) {
	if this.TopicExists(topic) {
		return
	}

	this.topics = append(this.topics, topic)
}

func (this *UnsubscribeMessage) RemoveTopic(topic []byte) {
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
	}
}

func (this *UnsubscribeMessage) TopicExists(topic []byte) bool {
	for _, t := range this.topics {
		if bytes.Equal(t, topic) {
			return true
		}
	}

	return false
}

func (this *UnsubscribeMessage) Decode(src io.Reader) (int, error) {
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
	}

	if len(this.topics) == 0 {
		return 0, fmt.Errorf("unsubscribe/Decode: Empty topic list")
	}

	return total, nil
}

func (this *UnsubscribeMessage) Encode() (io.Reader, int, error) {
	// packet ID
	total := 2

	for _, t := range this.topics {
		total += 2 + len(t)
	}

	this.SetRemainingLength(int32(total))

	total = 0

	_, total, err := this.fixedHeader.Encode()
	if err != nil {
		return nil, 0, err
	}

	if err = writeUint16(this.buf, this.packetId); err != nil {
		return nil, 0, err
	}
	total += 2

	var n int

	for _, t := range this.topics {
		if n, err = writeLPBytes(this.buf, t); err != nil {
			return nil, 0, err
		}
		total += n
	}

	return this.buf, total, nil
}
