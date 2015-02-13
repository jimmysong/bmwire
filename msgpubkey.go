// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bmwire

import (
	"fmt"
	"io"
	"time"
)

const (
	SimplePubKeyVersion    = 2
	ExtendedPubKeyVersion  = 3
	EncryptedPubKeyVersion = 4
)

type MsgPubKey struct {
	Nonce        uint64
	ExpiresTime  time.Time
	ObjectType   ObjectType
	Version      uint64
	StreamNumber uint64
	Behavior     uint32
	SigningKey   PubKey
	EncryptKey   PubKey
	NonceTrials  uint64
	ExtraBytes   uint64
	Signature    []byte
	Tag          ShaHash
	Encrypted    []byte
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgPubKey) BtcDecode(r io.Reader, pver uint32) error {
	var sec int64
	err := readElements(r, &msg.Nonce, &sec, &msg.ObjectType)
	if err != nil {
		return err
	}

	if msg.ObjectType != ObjectTypePubKey {
		str := fmt.Sprintf("Object Type should be %d, but is %d",
			ObjectTypePubKey, msg.ObjectType)
		return messageError("BtcDecode", str)
	}

	msg.ExpiresTime = time.Unix(sec, 0)
	msg.Version, err = readVarInt(r, pver)
	if err != nil {
		return err
	}

	msg.StreamNumber, err = readVarInt(r, pver)
	if err != nil {
		return err
	}

	if msg.Version >= EncryptedPubKeyVersion {
		err = readElement(r, &msg.Tag)
		if err != nil {
			return err
		}

	} else if msg.Version == ExtendedPubKeyVersion {

	} else {

	}

	return err
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgPubKey) BtcEncode(w io.Writer, pver uint32) error {
	err := writeElements(w, msg.Nonce, msg.ExpiresTime.Unix(), msg.ObjectType)
	if err != nil {
		return err
	}

	err = writeVarInt(w, pver, msg.Version)
	if err != nil {
		return err
	}

	err = writeVarInt(w, pver, msg.StreamNumber)
	if err != nil {
		return err
	}

	return err
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgPubKey) Command() string {
	return CmdObject
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgPubKey) MaxPayloadLength(pver uint32) uint32 {
	return uint32(8 + 8 + 4 + 8 + 8 + 32)
}

// NewMsgPubKey returns a new object message that conforms to the
// Message interface using the passed parameters and defaults for the remaining
// fields.
func NewMsgPubKey(nonce uint64, expires time.Time,
	version, streamNumber uint64, behavior uint32,
	signingKey, encryptKey PubKey, nonceTrials, extraBytes uint64,
	signature []byte, tag ShaHash) *MsgPubKey {

	// Limit the timestamp to one second precision since the protocol
	// doesn't support better.
	return &MsgPubKey{
		Nonce:        nonce,
		ExpiresTime:  expires,
		ObjectType:   ObjectTypeGetPubKey,
		Version:      version,
		StreamNumber: streamNumber,
		Behavior:     behavior,
		SigningKey:   signingKey,
		EncryptKey:   encryptKey,
		NonceTrials:  nonceTrials,
		ExtraBytes:   extraBytes,
		Signature:    signature,
		Tag:          tag,
	}
}
