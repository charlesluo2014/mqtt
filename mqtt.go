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

/*
Package mqtt is a encoder/decoder library for MQTT 3.1 and 3.1.1 messages. You can find
the MQTT specs at the following locations:

* 3.1.1 - http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/
* 3.1 - http://public.dhe.ibm.com/software/dw/webservices/ws-mqtt/mqtt-v3r1.html



*/

package mqtt

import (
	"bytes"
	"encoding/binary"
	"io"
	"regexp"

	"github.com/dataence/glog"
)

var clientIdRegexp *regexp.Regexp

func init() {
	clientIdRegexp, _ = regexp.Compile("^[0-9a-zA-Z]*$")
}

const (
	maxLPString          uint16 = 65535
	maxFixedHeaderLength int    = 5
	maxRemainingLength   int32  = 268435455 // bytes, or 256 MB
)

const (
	// QoS 0: At most once delivery
	// The message is delivered according to the capabilities of the underlying network.
	// No response is sent by the receiver and no retry is performed by the sender. The
	// message arrives at the receiver either once or not at all.
	QosAtMostOnce byte = iota

	// QoS 1: At least once delivery
	// This quality of service ensures that the message arrives at the receiver at least once.
	// A QoS 1 PUBLISH Packet has a Packet Identifier in its variable header and is acknowledged
	// by a PUBACK Packet. Section 2.3.1 provides more information about Packet Identifiers.
	QosAtLeastOnce

	// QoS 2: Exactly once delivery
	// This is the highest quality of service, for use when neither loss nor duplication of
	// messages are acceptable. There is an increased overhead associated with this quality of
	// service.
	QosExactlyOnce

	// QosFailure is a return value for a subscription if there's a problem while subscribing
	// to a specific topic.
	QosFailure = 0x80
)

// SupportedVersions is a map of the version number (0x3 or 0x4) to the version string,
// "MQIsdp" for 0x3, and "MQTT" for 0x4.
var SupportedVersions map[byte]string = map[byte]string{
	0x3: "MQIsdp",
	0x4: "MQTT",
}

// CopyMessage copies a single MQTT message from the io.Reader to the io.Writer. It returns
// the number of bytes copied and an error indicator. If an error is returned, then the
// bytes copied should be considered invalid.
func CopyMessage(dst io.Writer, src io.Reader) (int64, error) {
	// Copy the first byte. This is first byte of the control packet fixed header.
	total, err := io.CopyN(dst, src, 1)
	if err != nil {
		return 0, err
	}

	// Read the remaining length value from io.Reader, and copy the bytes into io.Writer
	// at the same time.
	remlen, m, err := readVarint32(dst, src)
	if err != nil {
		return total, err
	}
	total += int64(m)

	// Copy N bytes from io.Reader to io.Writer now that we know the remaining length.
	n, err := io.CopyN(dst, src, int64(remlen))
	if err != nil {
		return total, err
	}
	total += n

	return total, nil
}

// ValidTopic checks the topic, which is a slice of bytes, to see if it's valid. Topic is
// considered valid if it's longer than 0 bytes, and doesn't contain any wildcard characters
// such as * and #.
func ValidTopic(topic []byte) bool {
	return len(topic) > 0 && bytes.IndexByte(topic, '#') == -1 && bytes.IndexByte(topic, '*') == -1
}

// ValidQos checks the QoS value to see if it's valid. Valid QoS are QosAtMostOnce,
// QosAtLeastonce, and QosExactlyOnce.
func ValidQos(qos byte) bool {
	return qos == QosAtMostOnce || qos == QosAtLeastOnce || qos == QosExactlyOnce
}

// ValidClientId checks the client ID, which is a slice of bytes, to see if it's valid.
// Client ID is valid if it meets the requirement from the MQTT spec:
// 		The Server MUST allow ClientIds which are between 1 and 23 UTF-8 encoded bytes in length,
//		and that contain only the characters
//
//		"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
func ValidClientId(cid []byte) bool {
	return clientIdRegexp.Match(cid)
}

// ValidVersion checks to see if the version is valid. Current supported versions include 0x3 and 0x4.
func ValidVersion(v byte) bool {
	_, ok := SupportedVersions[v]
	return ok
}

// ValidConnackError checks to see if the error is a Connack Error or not
func ValidConnackError(err error) bool {
	return err == ErrUnacceptableProtocolVersion || err == ErrIdentifierRejected ||
		err == ErrServerUnavailable || err == ErrBadUsernameOrPassword || err == ErrNotAuthorized
}

func writeByte(dst io.Writer, b byte) error {
	var buf [1]byte
	buf[0] = b

	if _, err := dst.Write(buf[:]); err != nil {
		return err
	}

	return nil
}

func readUint16(buf *bytes.Buffer) (uint16, error) {
	if buf.Len() < 2 {
		return 0, glog.NewError("Insufficient buffer size. Expecting %d, got %d.", 2, buf.Len())
	}

	return binary.BigEndian.Uint16(buf.Next(2)), nil
}

func writeUint16(buf *bytes.Buffer, n uint16) error {
	var b [2]byte

	binary.BigEndian.PutUint16(b[:], n)
	buf.Write(b[:])

	return nil
}

func readLPBytes(buf *bytes.Buffer) ([]byte, int, error) {
	total := 0

	n, err := readUint16(buf)
	total += 2
	if err != nil {
		return nil, total, err
	}

	if buf.Len() < int(n) {
		return nil, total, glog.NewError("Insufficient buffer size. Expecting %d, got %d.", n, buf.Len())
	}

	total += int(n)

	return buf.Next(int(n)), total, nil
}

func writeLPBytes(buf *bytes.Buffer, b []byte) (int, error) {
	if len(b) > int(maxLPString) {
		return 0, glog.NewError("Length greater than %d bytes.", maxLPString)
	}

	total := 0

	err := writeUint16(buf, uint16(len(b)))
	if err != nil {
		return 0, err
	}
	total += 2

	n, err := buf.Write(b)
	if err != nil {
		return total, err
	}
	total += n

	return total, nil
}

// Modified from http://golang.org/src/pkg/encoding/binary/varint.go#106
func readVarint32(dst io.Writer, src io.Reader) (int32, int, error) {
	var x int32
	var s uint
	var i int
	var buf [4]byte

	for i = 0; i < 4; i++ {
		_, err := src.Read(buf[i : i+1])
		if err != nil {
			return 0, i + 1, err
		}

		if buf[i] < 0x80 {
			x |= int32(buf[i]) << s
			break
		}

		x |= int32(buf[i]&0x7f) << s
		s += 7
	}

	if i > 3 || (i == 3 && buf[3]&0x80 > 0) {
		return x, i + 1, glog.NewError("Malformed remaining length. 4th byte has continuation bit set.")
	}

	if dst != nil {
		if n, err := dst.Write(buf[:i+1]); err != nil {
			return x, n, glog.NewError("Error writing data: %v", err)
		}
	}

	return x, i + 1, nil
}

func writeVarint32(dst io.Writer, x int32) (int, error) {
	if x > maxRemainingLength {
		return 0, glog.NewError("Exceeded maximum of %d", maxRemainingLength)
	}

	var buf [4]byte
	i := 0

	for x >= 0x80 {
		buf[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	buf[i] = byte(x)

	n, err := dst.Write(buf[:i+1])
	if err != nil {
		return n, glog.NewError("Error writing data: %v", err)
	}

	return n, nil
}
