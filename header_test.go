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

func TestMessageHeaderFields(t *testing.T) {
	header := &fixedHeader{}

	header.SetRemainingLength(33)
	if header.RemainingLength() != 33 {
		t.Errorf("Incorrect remaining length. Expecting 33, got %d.", header.RemainingLength())
	}

	if err := header.SetRemainingLength(268435456); err == nil {
		t.Errorf("Incorrect result while setting remaining length to 268435456. Expecting error, got none.")
	}

	if err := header.SetRemainingLength(-1); err == nil {
		t.Errorf("Incorrect result while setting remaining length to -1. Expecting error, got none.")
	}

	if err := header.SetType(RESERVED); err == nil {
		t.Errorf("Incorrect result while setting type to RESERVED. Expecting error, got none.")
	}

	if err := header.SetType(PUBREL); err != nil {
		t.Errorf("Incorrect result while setting type to PUBREL: %v", err)
	}

	if header.Type() != PUBREL {
		t.Errorf("Incorret message type. Expecting %d, got %d.", PUBREL, header.Type())
	}

	if header.Name() != "PUBREL" {
		t.Errorf("Incorrect message name. Expecting PUBREL, got %s.", header.Name())
	}

	if header.Flags() != 2 {
		t.Errorf("Incorrect message flag. Expecting 2, got %d.", header.Flags())
	}
}

func TestMessageHeaderDecode(t *testing.T) {
	headerBytes := []byte{0x6f, 193, 2}
	buf := bytes.NewBuffer(headerBytes)
	header := &fixedHeader{}

	_, err := header.Decode(buf)
	if err == nil {
		t.Errorf("Incorrect result. Expecting error, got none.")
	}
}

// Remaining length too big
func TestMessageHeaderDecode2(t *testing.T) {
	headerBytes := []byte{0x62, 0xff, 0xff, 0xff, 0xff}
	buf := bytes.NewBuffer(headerBytes)
	header := &fixedHeader{}

	_, err := header.Decode(buf)
	if err == nil {
		t.Fatalf("Incorrect result. Expecting error, got none.")
	}
}

func TestMessageHeaderDecode3(t *testing.T) {
	headerBytes := []byte{0x62, 0xff}
	buf := bytes.NewBuffer(headerBytes)
	header := &fixedHeader{}

	_, err := header.Decode(buf)
	if err == nil {
		t.Fatalf("Incorrect result. Expecting error, got none.")
	}
}

func TestMessageHeaderDecode4(t *testing.T) {
	headerBytes := []byte{0x62, 0xff, 0xff, 0xff, 0x7f}
	buf := bytes.NewBuffer(headerBytes)
	header := &fixedHeader{
		mtype: 6,
		flags: 2,
	}

	n, err := header.Decode(buf)
	assert.Equal(t, true, 5, n, "Incorrect bytes decoded")

	assert.Equal(t, true, maxRemainingLength, header.RemainingLength(), "Incorrect remaining length")

	assert.Error(t, true, err)
}

func TestMessageHeaderDecode5(t *testing.T) {
	headerBytes := []byte{0x62, 0xff, 0x7f}
	buf := bytes.NewBuffer(headerBytes)
	header := &fixedHeader{
		mtype: 6,
		flags: 2,
	}

	n, err := header.Decode(buf)
	if err == nil {
		t.Fatalf("Unable to decode header: %v", err)
	} else if n != 3 {
		t.Errorf("Incorrect number of bytes decoded. Expecting 3, got %d.", n)
	}
}

func TestMessageHeaderEncode(t *testing.T) {
	header := &fixedHeader{}
	headerBytes := []byte{0x62, 193, 2}

	if err := header.SetType(PUBREL); err != nil {
		t.Errorf("Error setting message header type: %v", err)
	}

	if err := header.SetRemainingLength(321); err != nil {
		t.Errorf("Error setting message header length: %v", err)
	}

	dst, n, err := header.Encode()
	if err != nil {
		t.Errorf("Error encoding message header: %v", err)
	} else if n != 3 {
		t.Errorf("Error in bytes encoded. Expecting 3, got %d.", n)
	}

	if !bytes.Equal(dst.(*bytes.Buffer).Bytes(), headerBytes) {
		t.Errorf("Error encoding header, desitination buffer not the same as original.\n%v\n%v", dst.(*bytes.Buffer).Bytes(), headerBytes)
	}
}

func TestMessageHeaderEncode2(t *testing.T) {
	header := &fixedHeader{}

	if err := header.SetType(PUBREL); err != nil {
		t.Errorf("Error setting message header type: %v", err)
	}

	header.remlen = 268435456

	_, _, err := header.Encode()
	if err == nil {
		t.Errorf("Incorrect result. Expecting error, got none.")
	}
}

func TestMessageHeaderEncode3(t *testing.T) {
	header := &fixedHeader{}
	headerBytes := []byte{0x62, 0xff, 0xff, 0xff, 0x7f}

	if err := header.SetType(PUBREL); err != nil {
		t.Errorf("Error setting message header type: %v", err)
	}

	if err := header.SetRemainingLength(maxRemainingLength); err != nil {
		t.Errorf("Error setting message header length: %v", err)
	}

	dst, n, err := header.Encode()
	if err != nil {
		t.Errorf("Error encoding message header: %v", err)
	} else if n != 5 {
		t.Errorf("Error in bytes encoded. Expecting 5, got %d.", n)
	}

	if !bytes.Equal(dst.(*bytes.Buffer).Bytes(), headerBytes) {
		t.Errorf("Error encoding header, desitination buffer (%v) not the same as original (%v)", dst.(*bytes.Buffer).Bytes(), headerBytes)
	}
}

func TestMessageHeaderEncode4(t *testing.T) {
	header := &fixedHeader{}

	header.mtype = RESERVED2

	_, _, err := header.Encode()
	if err == nil {
		t.Errorf("Incorrect result. Expecting error, got none.")
	}
}

// This test is to ensure that an empty message is at least 2 bytes long
func TestMessageHeaderEncode5(t *testing.T) {
	msg := NewPingreqMessage()

	dst, n, err := msg.Encode()
	if err != nil {
		t.Errorf("Error encoding PINGREQ message: %v", err)
	} else if n != 2 {
		t.Errorf("Incorrect result. Expecting length of 2 bytes, got %d.", dst.(*bytes.Buffer).Len())
	}
}
