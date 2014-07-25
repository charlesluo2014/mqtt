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

	"github.com/dataence/glog"
)

// Fixed header
// - 1 byte for control packet type (bits 7-4) and flags (bits 3-0)
// - up to 4 byte for remaining length
type fixedHeader struct {
	buf    *bytes.Buffer
	remlen int32
	mtype  MessageType
	flags  byte
}

// String returns a string representation of the message.
func (this fixedHeader) String() string {
	return fmt.Sprintf("Packet type: %s\nFlags: %08b\nRemaining Length: %d bytes\n", this.mtype.Name(), this.flags, this.remlen)
}

// Encode returns an io.Reader in which the encoded bytes can be read. The second
// return value is the number of bytes encoded, so the caller knows how many bytes
// there will be. If Encode returns an error, then the first two return values
// should be considered invalid.
// Any changes to the message after Encode() is called will invalidate the io.Reader.
func (this *fixedHeader) Encode() (io.Reader, int, error) {
	total := 0

	if this.remlen > maxRemainingLength {
		return nil, 0, fmt.Errorf("header/Encode: remaining length (%d) too big", this.remlen)
	}

	if !this.mtype.Valid() {
		return nil, 0, fmt.Errorf("header/Encode: Invalid message type %d", this.mtype)
	}

	this.resetBuf()

	if err := this.buf.WriteByte(byte(this.mtype)<<4 | this.flags); err != nil {
		return nil, 0, err
	}
	total += 1

	n, err := writeVarint32(this.buf, this.remlen)
	if err != nil {
		return nil, total + n, err
	}
	total += n

	return this.buf, total, nil
}

// Decode reads from the io.Reader parameter until a full message is decoded, or
// when io.Reader returns EOF or error. The first return value is the number of
// bytes read from io.Reader. The second is error if Decode encounters any problems.
func (this *fixedHeader) Decode(src io.Reader) (int, error) {
	this.resetBuf()

	total, err := this.copy(src)
	if err != nil {
		return int(total), err
	}

	if int(this.remlen) != this.buf.Len() {
		return int(total), fmt.Errorf("header/Decode: Insufficient buffer size. Expecting %d bytes, got %d bytes.", this.remlen, this.buf.Len())
	}

	return int(total), nil
}

// Name returns a string representation of the message type. Examples include
// "PUBLISH", "SUBSCRIBE", and others. This is statically defined for each of
// the message types and cannot be changed.
func (this *fixedHeader) Name() string {
	return this.Type().Name()
}

// Desc returns a string description of the message type. For example, a
// CONNECT message would return "Client request to connect to Server." These
// descriptions are statically defined (copied from the MQTT spec) and cannot
// be changed.
func (this *fixedHeader) Desc() string {
	return this.Type().Desc()
}

// Type returns the MessageType of the Message. The retured value should be one
// of the constants defined for MessageType.
func (this *fixedHeader) Type() MessageType {
	return this.mtype
}

// SetType sets the message type of this message. It also correctly sets the
// default flags for the message type. It returns an error if the type is invalid.
func (this *fixedHeader) SetType(mtype MessageType) error {
	if !mtype.Valid() {
		return fmt.Errorf("header/SetType: Invalid control packet type %d", mtype)
	}

	this.mtype = mtype

	this.flags = mtype.DefaultFlags()

	return nil
}

// Flags returns the fixed header flags for this message.
func (this *fixedHeader) Flags() byte {
	return this.flags
}

// RemainingLength returns the length of the non-fixed-header part of the message.
func (this *fixedHeader) RemainingLength() int32 {
	return this.remlen
}

// SetRemainingLength sets the length of the non-fixed-header part of the message.
// It returns error if the length is greater than 268435455, which is the max
// message length as defined by the MQTT spec.
func (this *fixedHeader) SetRemainingLength(remlen int32) error {
	if remlen > maxRemainingLength || remlen < 0 {
		return fmt.Errorf("header/SetLength: Value (%d) out of bound (max %d, min 0)", remlen, maxRemainingLength)
	}

	this.remlen = remlen
	return nil
}

func (this *fixedHeader) copy(src io.Reader) (int64, error) {
	total, err := io.CopyN(this.buf, src, 1)
	if err != nil {
		return 0, err
	}

	b, err := this.buf.ReadByte()
	if err != nil {
		return 0, err
	}

	mtype := MessageType(b >> 4)
	if !mtype.Valid() {
		return total, glog.NewError("Invalid message type %d.", mtype)
	}

	if mtype != this.mtype {
		return total, glog.NewError("Invalid message type %d. Expecting %d.", mtype, this.mtype)
	}

	this.flags = b & 0x0f
	if this.mtype != PUBLISH && this.flags != this.mtype.DefaultFlags() {
		return total, glog.NewError("Invalid message (%d) flags. Expecting %d, got %d", this.mtype, this.mtype.DefaultFlags, this.flags)
	}

	if this.mtype == PUBLISH && !ValidQos((this.flags>>1)&0x3) {
		return total, glog.NewError("Invalid QoS (%d) for PUBLISH message.", (this.flags>>1)&0x3)
	}

	var m int
	this.remlen, m, err = readVarint32(this.buf, src)
	if err != nil {
		return total + int64(m), err
	}
	total += int64(m)
	this.buf.Next(m)

	n, err := io.CopyN(this.buf, src, int64(this.remlen))
	if err != nil {
		return total + n, err
	}

	return total, nil
}

func (this *fixedHeader) resetBuf() {
	if this.buf == nil {
		this.buf = new(bytes.Buffer)
	} else {
		this.buf.Reset()
	}
}
