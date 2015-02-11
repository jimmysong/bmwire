// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bmwire

import (
	"fmt"
	"io"
	"time"
)

type ObjectType uint32

const (
	ObjectTypeGetPubKey ObjectType = 0
	ObjectTypePubKey    ObjectType = 1
	ObjectTypeMsg       ObjectType = 2
	ObjectTypeBroadcast ObjectType = 3
)

// Map of service flags back to their constant names for pretty printing.
var obStrings = map[ObjectType]string{
	ObjectTypeGetPubKey: "GETPUBKEY",
	ObjectTypePubKey:    "PUBKEY",
	ObjectTypeMsg:       "MSG",
	ObjectTypeBroadcast: "BROADCAST",
}

type MsgObject struct {
	Nonce        uint64
	ExpiresTime  time.Time
	ObjectType   ObjectType
	Version      uint64
	StreamNumber uint64
	Payload      []byte
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgObject) BtcDecode(r io.Reader, pver uint32) error {
	var sec int64
	err := readElements(r, &msg.Nonce, &sec, &msg.ObjectType)
	if err != nil {
		return err
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

	s := 256
	for {
		tmp := make([]byte, s)
		n, err := io.ReadAtLeast(r, tmp, s)
		msg.Payload = append(msg.Payload, tmp...)
		if err == io.ErrUnexpectedEOF {
			msg.Payload = msg.Payload[:len(msg.Payload)-(s-n)]
			break
		} else if err != nil {
			return err
		}
	}

	return err
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgObject) BtcEncode(w io.Writer, pver uint32) error {
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

	_, err = w.Write(msg.Payload)

	return err
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgObject) Command() string {
	return CmdObject
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgObject) MaxPayloadLength(pver uint32) uint32 {
	return uint32(100000000)
}

func (msg *MsgObject) String() string {
	return fmt.Sprintf("msgobject: %d %s %s %d %d %x", msg.Nonce, msg.ExpiresTime, obStrings[msg.ObjectType], msg.Version, msg.StreamNumber, msg.Payload)
}
