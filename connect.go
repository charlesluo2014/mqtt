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
	"fmt"
	"io"
)

type ConnectMessage struct {
	fixedHeader

	// 7: username flag
	// 6: password flag
	// 5: will retain
	// 4-3: will QoS
	// 2: will flag
	// 1: clean session
	// 0: reserved
	connectFlags byte

	version byte

	keepAlive uint16

	protoName,
	clientId,
	willTopic,
	willMessage,
	username,
	password []byte
}

var _ Message = (*ConnectMessage)(nil)

func NewConnectMessage() *ConnectMessage {
	msg := &ConnectMessage{}
	msg.SetType(CONNECT)

	return msg
}

func (this ConnectMessage) String() string {
	return fmt.Sprintf("%v\nConnect Flags: %08b\nVersion: %d\nKeepAlive: %d\nClient ID: %s\nWill Topic: %s\nWill Message: %s\nUsername: %s\nPassword: %s\n",
		this.fixedHeader,
		this.connectFlags,
		this.Version(),
		this.KeepAlive(),
		this.ClientId(),
		this.WillTopic(),
		this.WillMessage(),
		this.Username(),
		this.Password(),
	)
}

func (this *ConnectMessage) Version() byte {
	return this.version
}

func (this *ConnectMessage) SetVersion(v byte) error {
	if _, ok := SupportedVersions[v]; !ok {
		return fmt.Errorf("connect/SetVersion: Invalid version number %d", v)
	}

	this.version = v
	return nil
}

func (this *ConnectMessage) CleanSession() bool {
	return ((this.connectFlags >> 1) & 0x1) == 1
}

func (this *ConnectMessage) SetCleanSession(v bool) {
	if v {
		this.connectFlags |= 0x2 // 00000010
	} else {
		this.connectFlags &= 253 // 11111101
	}
}

func (this *ConnectMessage) WillFlag() bool {
	return ((this.connectFlags >> 2) & 0x1) == 1
}

func (this *ConnectMessage) SetWillFlag(v bool) {
	if v {
		this.connectFlags |= 0x4 // 00000100
	} else {
		this.connectFlags &= 251 // 11111011
	}
}

func (this *ConnectMessage) WillQos() byte {
	return (this.connectFlags >> 3) & 0x3
}

func (this *ConnectMessage) SetWillQos(qos byte) error {
	if qos != QosAtMostOnce && qos != QosAtLeastOnce && qos != QosExactlyOnce {
		return fmt.Errorf("connect/SetWillQos: Invalid QoS level %d", qos)
	}

	this.connectFlags = (this.connectFlags & 231) | (qos << 3) // 231 = 11100111
	return nil
}

func (this *ConnectMessage) WillRetain() bool {
	return ((this.connectFlags >> 5) & 0x1) == 1
}

func (this *ConnectMessage) SetWillRetain(v bool) {
	if v {
		this.connectFlags |= 32 // 00100000
	} else {
		this.connectFlags &= 223 // 11011111
	}
}

func (this *ConnectMessage) PasswordFlag() bool {
	return ((this.connectFlags >> 6) & 0x1) == 1
}

func (this *ConnectMessage) SetPasswordFlag(v bool) {
	if v {
		this.connectFlags |= 64 // 01000000
	} else {
		this.connectFlags &= 191 // 10111111
	}
}

func (this *ConnectMessage) UsernameFlag() bool {
	return ((this.connectFlags >> 7) & 0x1) == 1
}

func (this *ConnectMessage) SetUsernameFlag(v bool) {
	if v {
		this.connectFlags |= 128 // 10000000
	} else {
		this.connectFlags &= 127 // 01111111
	}
}

func (this *ConnectMessage) KeepAlive() uint16 {
	return this.keepAlive
}

func (this *ConnectMessage) SetKeepAlive(v uint16) {
	this.keepAlive = v
}

func (this *ConnectMessage) ClientId() []byte {
	return this.clientId
}

func (this *ConnectMessage) SetClientId(v []byte) error {
	if len(v) > 0 && !ValidClientId(v) {
		return ErrConnackIdentifierRejected
	}

	this.clientId = v
	return nil
}

func (this *ConnectMessage) WillTopic() []byte {
	return this.willTopic
}

func (this *ConnectMessage) SetWillTopic(v []byte) {
	this.willTopic = v

	if len(v) > 0 {
		this.SetWillFlag(true)
	} else if len(this.willMessage) == 0 {
		this.SetWillFlag(false)
	}
}

func (this *ConnectMessage) WillMessage() []byte {
	return this.willMessage
}

func (this *ConnectMessage) SetWillMessage(v []byte) {
	this.willMessage = v

	if len(v) > 0 {
		this.SetWillFlag(true)
	} else if len(this.willTopic) == 0 {
		this.SetWillFlag(false)
	}
}

func (this *ConnectMessage) Username() []byte {
	return this.username
}

func (this *ConnectMessage) SetUsername(v []byte) {
	this.username = v

	if len(v) > 0 {
		this.SetUsernameFlag(true)
	} else {
		this.SetUsernameFlag(false)
	}
}

func (this *ConnectMessage) Password() []byte {
	return this.password
}

func (this *ConnectMessage) SetPassword(v []byte) {
	this.password = v

	if len(v) > 0 {
		this.SetPasswordFlag(true)
	} else {
		this.SetPasswordFlag(false)
	}
}

func (this *ConnectMessage) Decode(src io.Reader) (int, error) {
	total := 0

	n, err := this.fixedHeader.Decode(src)
	if err != nil {
		return total + n, err
	}
	total += n

	if n, err = this.decodeMessage(); err != nil {
		return total + n, err
	}
	total += n

	return total, nil
}

func (this *ConnectMessage) Encode() (io.Reader, int, error) {
	if this.Type() != CONNECT {
		return nil, 0, fmt.Errorf("connect/Encode: Invalid message type. Expecting %d, got %d", CONNECT, this.Type())
	}

	total := 0
	var n int
	verstr, ok := SupportedVersions[this.version]
	if !ok {
		return nil, 0, fmt.Errorf("connect/Encode: Unsupported protocol version %d", this.version)
	}

	// 2 bytes protocol name length
	// n bytes protocol name
	// 1 byte protocol version
	// 1 byte connect flags
	// 2 bytes keep alive timer
	total += 2 + len(verstr) + 1 + 1 + 2

	// Add the clientID length, 2 is the length prefix
	total += 2 + len(this.clientId)

	// Add the will topic and will message length, and the length prefixes
	if this.WillFlag() {
		total += 2 + len(this.willTopic) + 2 + len(this.willMessage)
	}

	// Add the username length
	// According to the 3.1 spec, it's possible that the usernameFlag is set,
	// but the user name string is missing.
	if this.UsernameFlag() && len(this.username) > 0 {
		total += 2 + len(this.username)
	}

	// Add the password length
	// According to the 3.1 spec, it's possible that the passwordFlag is set,
	// but the password string is missing.
	if this.PasswordFlag() && len(this.password) > 0 {
		total += 2 + len(this.password)
	}

	if err := this.SetRemainingLength(int32(total)); err != nil {
		return nil, 0, err
	}

	total = 0

	_, n, err := this.fixedHeader.Encode()
	if err != nil {
		return nil, total + n, err
	}
	total += n

	if n, err = this.encodeMessage(); err != nil {
		return nil, total + n, err
	}
	total += n

	return this.buf, total, nil
}

func (this *ConnectMessage) encodeMessage() (int, error) {
	total := 0

	verstr, ok := SupportedVersions[this.version]
	if !ok {
		return 0, fmt.Errorf("connect/encodeVariableHeader: Unsupported protocol version %d", this.version)
	}

	n, err := writeLPBytes(this.buf, []byte(verstr))
	if err != nil {
		return 0, err
	}
	total += int(n)

	this.buf.WriteByte(this.version)
	total += 1

	this.buf.WriteByte(this.connectFlags)
	total += 1

	if err = writeUint16(this.buf, this.keepAlive); err != nil {
		return total, err
	}
	total += 2

	if n, err = writeLPBytes(this.buf, this.clientId); err != nil {
		return total + n, err
	}
	total += n

	if this.WillFlag() {
		if n, err = writeLPBytes(this.buf, this.willTopic); err != nil {
			return total + n, err
		}
		total += n

		if n, err = writeLPBytes(this.buf, this.willMessage); err != nil {
			return total + n, err
		}
		total += n
	}

	// According to the 3.1 spec, it's possible that the usernameFlag is set,
	// but the username string is missing.
	if this.UsernameFlag() && len(this.username) > 0 {
		if n, err = writeLPBytes(this.buf, this.username); err != nil {
			return total + n, err
		}
		total += n
	}

	// According to the 3.1 spec, it's possible that the passwordFlag is set,
	// but the password string is missing.
	if this.PasswordFlag() && len(this.password) > 0 {
		if n, err = writeLPBytes(this.buf, this.password); err != nil {
			return total + n, err
		}
		total += n
	}

	return total, nil
}

func (this *ConnectMessage) decodeMessage() (int, error) {
	var n, total int
	var err error

	if this.protoName, n, err = readLPBytes(this.buf); err != nil {
		return total + n, err
	}
	total += n

	if this.version, err = this.buf.ReadByte(); err != nil {
		return total, err
	}
	total += 1

	if verstr, ok := SupportedVersions[this.version]; !ok {
		return total, ErrConnackUnacceptableProtocolVersion
	} else if verstr != string(this.protoName) {
		return total, ErrConnackUnacceptableProtocolVersion
	}

	if this.connectFlags, err = this.buf.ReadByte(); err != nil {
		return total, err
	}
	total += 1

	if this.connectFlags&0x1 != 0 {
		return total, fmt.Errorf("connect/decodeMessage: Connect Flags reserved bit 0 is not 0")
	}

	if this.WillQos() > QosExactlyOnce {
		return total, fmt.Errorf("connect/decodeMessage: Invalid QoS level (%d) for %s message", this.WillQos(), this.Name())
	}

	if !this.WillFlag() && (this.WillRetain() || this.WillQos() != QosAtMostOnce) {
		return total, fmt.Errorf("connect/decodeMessage: Protocol violation: If the Will Flag (%t) is set to 0 the Will QoS (%d) and Will Retain (%t) fields MUST be set to zero", this.WillFlag(), this.WillQos(), this.WillRetain())
	}

	if this.UsernameFlag() && !this.PasswordFlag() {
		return total, fmt.Errorf("connect/decodeMessage: Username flag is set but Password flag is not set")
	}

	if this.keepAlive, err = readUint16(this.buf); err != nil {
		return total, err
	}
	total += 2

	if this.clientId, n, err = readLPBytes(this.buf); err != nil {
		return total + n, err
	}
	total += n

	// If the Client supplies a zero-byte ClientId, the Client MUST also set CleanSession to 1
	if len(this.clientId) == 0 && !this.CleanSession() {
		return total, ErrConnackIdentifierRejected
	}

	// The ClientId must contain only characters 0-9, a-z, and A-Z
	// We also support ClientId longer than 23 encoded bytes
	// We do not support ClientId outside of the above characters
	if len(this.clientId) > 0 && !ValidClientId(this.clientId) {
		return total, ErrConnackIdentifierRejected
	}

	if this.WillFlag() {
		if this.willTopic, n, err = readLPBytes(this.buf); err != nil {
			return total + n, err
		}
		total += n

		if this.willMessage, n, err = readLPBytes(this.buf); err != nil {
			return total + n, err
		}
		total += n
	}

	// According to the 3.1 spec, it's possible that the passwordFlag is set,
	// but the password string is missing.
	if this.UsernameFlag() && this.buf.Len() > 0 {
		if this.username, n, err = readLPBytes(this.buf); err != nil {
			return total + n, err
		}
		total += n
	}

	// According to the 3.1 spec, it's possible that the passwordFlag is set,
	// but the password string is missing.
	if this.PasswordFlag() && this.buf.Len() > 0 {
		if this.password, n, err = readLPBytes(this.buf); err != nil {
			return total + n, err
		}
		total += n
	}

	if this.buf.Len() > 0 {
		return total, fmt.Errorf("connect/decodeMessage: Invalid buffer size. Still has %d bytes at the end.", this.buf.Len())
	}

	return total, nil
}
