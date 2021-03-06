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

func TestSubackMessageFields(t *testing.T) {
	msg := NewSubackMessage()

	msg.SetPacketId(100)
	assert.Equal(t, true, 100, msg.PacketId(), "Error setting packet ID.")

	msg.AddReturnCode(1)
	assert.Equal(t, true, 1, len(msg.ReturnCodes()), "Error adding return code.")

	err := msg.AddReturnCode(0x90)
	assert.Error(t, true, err)
}

func TestSubackMessageDecode(t *testing.T) {
	msgBytes := []byte{
		byte(SUBACK << 4),
		6,
		0,    // packet ID MSB (0)
		7,    // packet ID LSB (7)
		0,    // return code 1
		1,    // return code 2
		2,    // return code 3
		0x80, // return code 4
	}

	src := bytes.NewBuffer(msgBytes)
	msg := NewSubackMessage()

	n, err := msg.Decode(src)
	assert.NoError(t, true, err, "Error decoding message.")

	assert.Equal(t, true, len(msgBytes), n, "Error decoding message.")

	assert.Equal(t, true, SUBACK, msg.Type(), "Error decoding message.")

	assert.Equal(t, true, 4, len(msg.ReturnCodes()), "Error adding return code.")
}

// test with wrong return code
func TestSubackMessageDecode2(t *testing.T) {
	msgBytes := []byte{
		byte(SUBACK << 4),
		6,
		0,    // packet ID MSB (0)
		7,    // packet ID LSB (7)
		0,    // return code 1
		1,    // return code 2
		2,    // return code 3
		0x81, // return code 4
	}

	src := bytes.NewBuffer(msgBytes)
	msg := NewSubackMessage()

	_, err := msg.Decode(src)
	assert.Error(t, true, err)
}

func TestSubackMessageEncode(t *testing.T) {
	msgBytes := []byte{
		byte(SUBACK << 4),
		6,
		0,    // packet ID MSB (0)
		7,    // packet ID LSB (7)
		0,    // return code 1
		1,    // return code 2
		2,    // return code 3
		0x80, // return code 4
	}

	msg := NewSubackMessage()
	msg.SetPacketId(7)
	msg.AddReturnCode(0)
	msg.AddReturnCode(1)
	msg.AddReturnCode(2)
	msg.AddReturnCode(0x80)

	dst, n, err := msg.Encode()
	assert.NoError(t, true, err, "Error decoding message.")

	assert.Equal(t, true, len(msgBytes), n, "Error decoding message.")

	assert.Equal(t, true, msgBytes, dst.(*bytes.Buffer).Bytes(), "Error decoding message.")
}
