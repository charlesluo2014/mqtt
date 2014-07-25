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

var (
	lpstrings []string = []string{
		"this is a test",
		"hope it succeeds",
		"but just in case",
		"send me your millions",
		"",
	}

	lpstringBytes []byte = []byte{
		0x0, 0xe, 't', 'h', 'i', 's', ' ', 'i', 's', ' ', 'a', ' ', 't', 'e', 's', 't',
		0x0, 0x10, 'h', 'o', 'p', 'e', ' ', 'i', 't', ' ', 's', 'u', 'c', 'c', 'e', 'e', 'd', 's',
		0x0, 0x10, 'b', 'u', 't', ' ', 'j', 'u', 's', 't', ' ', 'i', 'n', ' ', 'c', 'a', 's', 'e',
		0x0, 0x15, 's', 'e', 'n', 'd', ' ', 'm', 'e', ' ', 'y', 'o', 'u', 'r', ' ', 'm', 'i', 'l', 'l', 'i', 'o', 'n', 's',
		0x0, 0x0,
	}

	msgBytes []byte = []byte{
		byte(CONNECT << 4),
		60,
		0, // Length MSB (0)
		4, // Length LSB (4)
		'M', 'Q', 'T', 'T',
		4,   // Protocol level 4
		206, // connect flags 11001110, will QoS = 01
		0,   // Keep Alive MSB (0)
		10,  // Keep Alive LSB (10)
		0,   // Client ID MSB (0)
		7,   // Client ID LSB (7)
		's', 'u', 'r', 'g', 'e', 'm', 'q',
		0, // Will Topic MSB (0)
		4, // Will Topic LSB (4)
		'w', 'i', 'l', 'l',
		0,  // Will Message MSB (0)
		12, // Will Message LSB (12)
		's', 'e', 'n', 'd', ' ', 'm', 'e', ' ', 'h', 'o', 'm', 'e',
		0, // Username ID MSB (0)
		7, // Username ID LSB (7)
		's', 'u', 'r', 'g', 'e', 'm', 'q',
		0,  // Password ID MSB (0)
		10, // Password ID LSB (10)
		'v', 'e', 'r', 'y', 's', 'e', 'c', 'r', 'e', 't',
	}
)

func TestReadUint16(t *testing.T) {
	var nums []uint16 = []uint16{257, 10, 65535}
	data := []byte{0x1, 0x1, 0x0, 0x0a, 0xff, 0xff}
	buf := bytes.NewBuffer(data)

	for _, n := range nums {
		if b, err := readUint16(buf); err != nil {
			t.Errorf("Incorrect result. Error reading %d: %v", n, err)
		} else if b != n {
			t.Errorf("Incorrect result. Expecting %d, got %d.", n, b)
		}
	}

	if _, err := readUint16(buf); err == nil {
		t.Errorf("Incorrect result. Expecting error, got none: %v", err)
	}
}

func TestWriteUint16(t *testing.T) {
	var nums []uint16 = []uint16{257, 10, 65535}
	data := []byte{0x1, 0x1, 0x0, 0x0a, 0xff, 0xff}
	var buf bytes.Buffer

	for _, n := range nums {
		if err := writeUint16(&buf, n); err != nil {
			t.Errorf("Incorrect result. Error writing %d: %v", n, err)
		}
	}

	if !bytes.Equal(data, buf.Bytes()) {
		t.Errorf("Incorrect result. Input and output are not equal")
	}
}

func TestReadLPBytes(t *testing.T) {
	buf := bytes.NewBuffer(lpstringBytes)

	for _, str := range lpstrings {
		if b, n, err := readLPBytes(buf); err != nil {
			t.Errorf("Incorrect result. Error reading line '%s': %v", str, err)
		} else if string(b) != str {
			t.Errorf("Incorrect result. Expecting '%s', got '%s'.", str, string(b))
		} else if n != len(str)+2 {
			t.Errorf("Incorrect result. Expecting length %d, got %d.", len(str)+2, n)
		}
	}
}

func TestWriteLPBytes(t *testing.T) {
	var buf bytes.Buffer

	for _, str := range lpstrings {
		if n, err := writeLPBytes(&buf, []byte(str)); err != nil {
			t.Errorf("Incorrect result. Error writing line '%s': %v", str, err)
		} else if n != len(str)+2 {
			t.Errorf("Incorrect result. Expecting length %d, got %d.", len(str)+2, n)
		}
	}

	if !bytes.Equal(lpstringBytes, buf.Bytes()) {
		t.Errorf("Incorrect result. Input and output are not equal.")
	}
}

func TestCopyMessageSuccess(t *testing.T) {
	src := bytes.NewBuffer(msgBytes)
	var dst bytes.Buffer

	_, err := CopyMessage(&dst, src)
	if err != nil {
		t.Errorf("Error copying message: %v", err)
	}

	if !bytes.Equal(msgBytes, dst.Bytes()) {
		t.Errorf("Incorrect result. Input and output are not equal.\n%v\n%v", msgBytes, dst.Bytes())
	}
}

func TestCopyMessageFailure(t *testing.T) {
	src := bytes.NewBuffer(msgBytes[:len(msgBytes)-2])
	var dst bytes.Buffer

	_, err := CopyMessage(&dst, src)
	if err == nil {
		t.Errorf("Incorrect result. Expecting error, got none.")
	}
}

func TestMessageTypes(t *testing.T) {
	if CONNECT != 1 ||
		CONNACK != 2 ||
		PUBLISH != 3 ||
		PUBACK != 4 ||
		PUBREC != 5 ||
		PUBREL != 6 ||
		PUBCOMP != 7 ||
		SUBSCRIBE != 8 ||
		SUBACK != 9 ||
		UNSUBSCRIBE != 10 ||
		UNSUBACK != 11 ||
		PINGREQ != 12 ||
		PINGRESP != 13 ||
		DISCONNECT != 14 {

		t.Errorf("Message types have invalid code")
	}
}

func TestQosCodes(t *testing.T) {
	if QosAtMostOnce != 0 || QosAtLeastOnce != 1 || QosExactlyOnce != 2 {
		t.Errorf("QOS codes invalid")
	}
}

func TestConnackReturnCodes(t *testing.T) {
	assert.Equal(t, false, ErrUnacceptableProtocolVersion, ConnackCode(1).Error(), "Incorrect ConnackCode error value.")

	assert.Equal(t, false, ErrIdentifierRejected, ConnackCode(2).Error(), "Incorrect ConnackCode error value.")

	assert.Equal(t, false, ErrServerUnavailable, ConnackCode(3).Error(), "Incorrect ConnackCode error value.")

	assert.Equal(t, false, ErrBadUsernameOrPassword, ConnackCode(4).Error(), "Incorrect ConnackCode error value.")

	assert.Equal(t, false, ErrNotAuthorized, ConnackCode(5).Error(), "Incorrect ConnackCode error value.")
}

func TestFixedHeaderFlags(t *testing.T) {
	type detail struct {
		name  string
		flags byte
	}

	details := map[MessageType]detail{
		RESERVED:    detail{"RESERVED", 0},
		CONNECT:     detail{"CONNECT", 0},
		CONNACK:     detail{"CONNACK", 0},
		PUBLISH:     detail{"PUBLISH", 0},
		PUBACK:      detail{"PUBACK", 0},
		PUBREC:      detail{"PUBREC", 0},
		PUBREL:      detail{"PUBREL", 2},
		PUBCOMP:     detail{"PUBCOMP", 0},
		SUBSCRIBE:   detail{"SUBSCRIBE", 2},
		SUBACK:      detail{"SUBACK", 0},
		UNSUBSCRIBE: detail{"UNSUBSCRIBE", 2},
		UNSUBACK:    detail{"UNSUBACK", 0},
		PINGREQ:     detail{"PINGREQ", 0},
		PINGRESP:    detail{"PINGRESP", 0},
		DISCONNECT:  detail{"DISCONNECT", 0},
		RESERVED2:   detail{"RESERVED2", 0},
	}

	for m, d := range details {
		if m.Name() != d.name {
			t.Errorf("Name mismatch. Expecting %s, got %s.", d.name, m.Name())
		}

		if m.DefaultFlags() != d.flags {
			t.Errorf("Flag mismatch for %s. Expecting %d, got %d.", m.Name(), d.flags, m.DefaultFlags())
		}
	}
}

func TestSupportedVersions(t *testing.T) {
	for k, v := range SupportedVersions {
		if k == 0x03 && v != "MQIsdp" {
			t.Errorf("Protocol version and name mismatch. Expect %s, got %s.", "MQIsdp", v)
		}
	}
}
