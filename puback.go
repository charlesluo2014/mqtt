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

import "io"

type PubackMessage struct {
	fixedHeader

	packetId uint16
}

var _ Message = (*PubackMessage)(nil)

func NewPubackMessage() *PubackMessage {
	msg := &PubackMessage{}
	msg.SetType(PUBACK)

	return msg
}

func (this *PubackMessage) PacketId() uint16 {
	return this.packetId
}

func (this *PubackMessage) SetPacketId(v uint16) {
	this.packetId = v
}

func (this *PubackMessage) Decode(src io.Reader) (int, error) {
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

	return total, nil
}

func (this *PubackMessage) Encode() (io.Reader, int, error) {
	this.SetRemainingLength(2)

	_, total, err := this.fixedHeader.Encode()
	if err != nil {
		return nil, 0, err
	}

	if err = writeUint16(this.buf, this.packetId); err != nil {
		return nil, 0, err
	}
	total += 2

	return this.buf, total, nil
}
