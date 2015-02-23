// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bmwire_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jimmysong/bmwire"
)

// TestGetPubKey tests the MsgGetPubKey API.
func TestGetPubKey(t *testing.T) {
	// Ensure the command is expected value.
	wantCmd := "object"
	now := time.Now()
	// ripe-based getpubkey message
	ripeBytes := make([]byte, 20)
	ripeBytes[0] = 1
	ripe, err := bmwire.NewRipeHash(ripeBytes)
	if err != nil {
		t.Fatalf("could not make a ripe hash %s", err)
	}
	msg := bmwire.NewMsgGetPubKey(83928, now, 2, 1, ripe, nil)
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgGetPubKey: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	// Num objectentory vectors (varInt) + max allowed objectentory vectors.
	wantPayload := uint32(1 << 18)
	maxPayload := msg.MaxPayloadLength()
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"got %v, want %v", maxPayload, wantPayload)
	}

	return
}

// TestGetPubKeyWire tests the MsgGetPubKey bmwire.encode and decode for various numbers
// of objectentory vectors and protocol versions.
func TestGetPubKeyWire(t *testing.T) {

	ripeBytes := make([]byte, 20)
	ripeBytes[0] = 1
	ripe, err := bmwire.NewRipeHash(ripeBytes)
	if err != nil {
		t.Fatalf("could not make a ripe hash %s", err)
	}

	expires := time.Unix(0x495fab29, 0) // 2009-01-03 12:15:05 -0600 CST)

	// empty tag, something in ripe
	msgRipe := bmwire.NewMsgGetPubKey(83928, expires, 2, 1, ripe, nil)

	// empty ripe, something in tag
	tagBytes := make([]byte, 32)
	tagBytes[0] = 1
	tag, err := bmwire.NewShaHash(tagBytes)
	if err != nil {
		t.Fatalf("could not make a tag hash %s", err)
	}
	msgTag := bmwire.NewMsgGetPubKey(83928, expires, 4, 1, nil, tag)

	RipeEncoded := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x47, 0xd8, // 83928 nonce
		0x00, 0x00, 0x00, 0x00, 0x49, 0x5f, 0xab, 0x29, // 64-bit timestamp
		0x00, 0x00, 0x00, 0x00, // object type (GETPUBKEY)
		0x02, // object version
		0x01, // stream number
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // 20-byte ripemd
	}

	TagEncoded := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x47, 0xd8, // 83928 nonce
		0x00, 0x00, 0x00, 0x00, 0x49, 0x5f, 0xab, 0x29, // 64-bit timestamp
		0x00, 0x00, 0x00, 0x00, // object type (GETPUBKEY)
		0x04, // object version
		0x01, // stream number
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 32-byte ripemd
	}

	tests := []struct {
		in  *bmwire.MsgGetPubKey // Message to encode
		out *bmwire.MsgGetPubKey // Expected decoded message
		buf []byte               // Wire encoding
	}{
		// Latest protocol version with multiple object vectors.
		{
			msgRipe,
			msgRipe,
			RipeEncoded,
		},
		{
			msgTag,
			msgTag,
			TagEncoded,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode the message to bmwire.format.
		var buf bytes.Buffer
		err := test.in.Encode(&buf)
		if err != nil {
			t.Errorf("Encode #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("Encode #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode the message from bmwire.format.
		var msg bmwire.MsgGetPubKey
		rbuf := bytes.NewReader(test.buf)
		err = msg.Decode(rbuf)
		if err != nil {
			t.Errorf("Decode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("Decode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}

// TestGetPubKeyWireError tests the MsgGetPubKey error paths
func TestGetPubKeyWireError(t *testing.T) {
	wireErr := &bmwire.MessageError{}

	// Ensure calling MsgVersion.Decode with a non *bytes.Buffer returns
	// error.
	fr := newFixedReader(0, []byte{})
	if err := baseVersion.Decode(fr); err == nil {
		t.Errorf("Did not received error when calling " +
			"MsgVersion.Decode with non *bytes.Buffer")
	}

	tests := []struct {
		in       *bmwire.MsgGetPubKey // Value to encode
		buf      []byte               // Wire encoding
		max      int                  // Max size of fixed buffer to induce errors
		writeErr error                // Expected write error
		readErr  error                // Expected read error
	}{
		// Force error in nonce
		{baseGetPubKey, baseGetPubKeyEncoded, 0, io.ErrShortWrite, io.EOF},
		// Force error in expirestime.
		{baseGetPubKey, baseGetPubKeyEncoded, 8, io.ErrShortWrite, io.EOF},
		// Force error in object type.
		{baseGetPubKey, baseGetPubKeyEncoded, 16, io.ErrShortWrite, io.EOF},
		// Force error in version.
		{baseGetPubKey, baseGetPubKeyEncoded, 20, io.ErrShortWrite, io.EOF},
		// Force error in stream number.
		{baseGetPubKey, baseGetPubKeyEncoded, 21, io.ErrShortWrite, io.EOF},
		// Force error in ripe.
		{baseGetPubKey, baseGetPubKeyEncoded, 22, io.ErrShortWrite, io.EOF},
		// Force error in tag.
		{tagGetPubKey, tagGetPubKeyEncoded, 22, io.ErrShortWrite, io.EOF},
		// Force error object type validation.
		{baseGetPubKey, basePubKeyEncoded, 20, io.ErrShortWrite, wireErr},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to bmwire.format.
		w := newFixedWriter(test.max)
		err := test.in.Encode(w)
		if reflect.TypeOf(err) != reflect.TypeOf(test.writeErr) {
			t.Errorf("Encode #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		// For errors which are not of type bmwire.MessageError, check
		// them for equality.
		if _, ok := err.(*bmwire.MessageError); !ok {
			if err != test.writeErr {
				t.Errorf("Encode #%d wrong error got: %v, "+
					"want: %v", i, err, test.writeErr)
				continue
			}
		}

		// Decode from bmwire.format.
		var msg bmwire.MsgGetPubKey
		buf := bytes.NewBuffer(test.buf[0:test.max])
		err = msg.Decode(buf)
		if reflect.TypeOf(err) != reflect.TypeOf(test.readErr) {
			t.Errorf("Decode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}

		// For errors which are not of type bmwire.MessageError, check
		// them for equality.
		if _, ok := err.(*bmwire.MessageError); !ok {
			if err != test.readErr {
				t.Errorf("Decode #%d wrong error got: %v, "+
					"want: %v", i, err, test.readErr)
				continue
			}
		}
	}
}

// baseGetPubKey is used in the various tests as a baseline MsgGetPubKey.
var baseGetPubKey = &bmwire.MsgGetPubKey{
	Nonce:        123123,                   // 0x1e0f3
	ExpiresTime:  time.Unix(0x495fab29, 0), // 2009-01-03 12:15:05 -0600 CST)
	ObjectType:   bmwire.ObjectTypeGetPubKey,
	Version:      3,
	StreamNumber: 1,
	Ripe:         &bmwire.RipeHash{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	Tag:          nil,
}

// baseGetPubKeyEncoded is the bmwire.encoded bytes for baseGetPubKey
// using version 2 (pre-tag
var baseGetPubKeyEncoded = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xe0, 0xf3, // Nonce
	0x00, 0x00, 0x00, 0x00, 0x49, 0x5f, 0xab, 0x29, // 64-bit Timestamp
	0x00, 0x00, 0x00, 0x00, // object type
	0x03, // Version
	0x01, // Stream Number
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, // Ripe
}

// baseGetPubKey is used in the various tests as a baseline MsgGetPubKey.
var tagGetPubKey = &bmwire.MsgGetPubKey{
	Nonce:        123123,                   // 0x1e0f3
	ExpiresTime:  time.Unix(0x495fab29, 0), // 2009-01-03 12:15:05 -0600 CST)
	ObjectType:   bmwire.ObjectTypeGetPubKey,
	Version:      4,
	StreamNumber: 1,
	Ripe:         nil,
	Tag: &bmwire.ShaHash{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
}

// baseGetPubKeyEncoded is the bmwire.encoded bytes for baseGetPubKey
// using version 2 (pre-tag
var tagGetPubKeyEncoded = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xe0, 0xf3, // Nonce
	0x00, 0x00, 0x00, 0x00, 0x49, 0x5f, 0xab, 0x29, // 64-bit Timestamp
	0x00, 0x00, 0x00, 0x00, // object type
	0x04, // Version
	0x01, // Stream Number
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Ripe
}
