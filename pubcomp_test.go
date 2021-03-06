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
	"testing"

	"github.com/dataence/assert"
)

func TestPubcompMessageFields(t *testing.T) {
	msg := NewPubcompMessage()

	msg.SetPacketId(100)
	if msg.PacketId() != 100 {
		t.Errorf("Error setting packet ID. Expecting 100, got %d.", msg.PacketId())
	}
}

func TestPubcompMessageDecode(t *testing.T) {
	msgBytes := []byte{
		byte(PUBCOMP << 4),
		2,
		0, // packet ID MSB (0)
		7, // packet ID LSB (7)
	}

	src := bytes.NewBuffer(msgBytes)
	msg := NewPubcompMessage()

	n, err := msg.Decode(src)
	assert.NoError(t, true, err, "Error decoding message.")

	assert.Equal(t, true, len(msgBytes), n, "Error decoding message.")

	assert.Equal(t, true, PUBCOMP, msg.Type(), "Error decoding message.")

	assert.Equal(t, true, 7, msg.PacketId(), "Error decoding message.")
}

// test insufficient bytes
func TestPubcompMessageDecode2(t *testing.T) {
	msgBytes := []byte{
		byte(PUBCOMP << 4),
		2,
		7, // packet ID LSB (7)
	}

	src := bytes.NewBuffer(msgBytes)
	msg := NewPubcompMessage()

	_, err := msg.Decode(src)
	assert.Error(t, true, err)
}

func TestPubcompMessageEncode(t *testing.T) {
	msgBytes := []byte{
		byte(PUBCOMP << 4),
		2,
		0, // packet ID MSB (0)
		7, // packet ID LSB (7)
	}

	msg := NewPubcompMessage()
	msg.SetPacketId(7)

	dst, n, err := msg.Encode()
	assert.NoError(t, true, err, "Error decoding message.")

	assert.Equal(t, true, len(msgBytes), n, "Error decoding message.")

	assert.Equal(t, true, msgBytes, dst.(*bytes.Buffer).Bytes(), "Error decoding message.")
}
