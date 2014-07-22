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
	"encoding/binary"
	"fmt"
	"io"
	"regexp"

	"github.com/dataence/glog"
)

var clientIdRegexp *regexp.Regexp

func init() {
	clientIdRegexp, _ = regexp.Compile("^[0-9a-zA-Z]*$")
}

type ConnackReturnCode struct {
	code     byte
	response string
	desc     string
}

const (
	maxLPString          uint16 = 65535
	maxFixedHeaderLength int    = 5
	maxRemainingLength   int32  = 268435455 // bytes, or 256 MB
)

const (
	QosAtMostOnce byte = iota
	QosAtLeastOnce
	QosExactlyOnce
	QosFailure = 0x80
)

var (
	ErrConnackConnectionAccepted error = ConnackReturnCode{
		0,
		"Connection Accepted",
		"Connection accepted",
	}

	ErrConnackUnacceptableProtocolVersion error = ConnackReturnCode{
		1,
		"Connection Refused, unacceptable protocol version",
		"The Server does not support the level of the MQTT protocol requested by the Client",
	}

	ErrConnackIdentifierRejected error = ConnackReturnCode{
		2,
		"Connection Refused, identifier rejected",
		"The Client identifier is correct UTF-8 but not allowed by the server",
	}

	ErrConnackServerUnavailable error = ConnackReturnCode{
		3,
		"Connection Refused, Server unavailable",
		"The Network Connection has been made but the MQTT service is unavailable",
	}

	ErrConnackBadUsernameOrPassword error = ConnackReturnCode{
		4,
		"Connection Refused, bad user name or password",
		"The data in the user name or password is malformed",
	}

	ErrConnackNotAuthorized error = ConnackReturnCode{
		5,
		"Connection Refused, not authorized",
		"The Client is not authorized to connect",
	}

	ConnackReturnCodes []error = []error{
		ErrConnackConnectionAccepted,
		ErrConnackUnacceptableProtocolVersion,
		ErrConnackIdentifierRejected,
		ErrConnackServerUnavailable,
		ErrConnackBadUsernameOrPassword,
		ErrConnackNotAuthorized,
	}
)

var SupportedVersions map[byte]string = map[byte]string{
	0x3: "MQIsdp",
	0x4: "MQTT",
}

func (this ConnackReturnCode) Error() string {
	return fmt.Sprintf("CONNACK Return Code (%d): %s", this.code, this.response)
}

func CopyMessage(dst io.Writer, src io.Reader) (int64, error) {
	n, err := io.CopyN(dst, src, 1)
	if err != nil {
		return 0, err
	}
	total := n

	remlen, m, err := readVarint32(dst, src)
	if err != nil {
		return total, err
	}
	total += int64(m)

	if n, err = io.CopyN(dst, src, int64(remlen)); err != nil {
		return total, err
	}
	total += n

	return total, nil
}

func ValidTopic(topic []byte) bool {
	return len(topic) > 0 && bytes.IndexByte(topic, '#') == -1 && bytes.IndexByte(topic, '*') == -1
}

func ValidQos(qos byte) bool {
	return qos == QosAtMostOnce || qos == QosAtLeastOnce || qos == QosExactlyOnce
}

func ValidClientId(cid []byte) bool {
	return clientIdRegexp.Match(cid)
}

func ValidVersion(v byte) bool {
	_, ok := SupportedVersions[v]
	return ok
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
