// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bmwire_test

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jimmysong/bmwire"
)

// TestObject tests the MsgObject API.
func TestObject(t *testing.T) {
	pver := bmwire.ProtocolVersion

	// Ensure the command is expected value.
	wantCmd := "object"
	now := time.Now()
	twentyBytes := make([]byte, 20)
	msg := bmwire.NewMsgObject(83928, now, bmwire.ObjectTypeGetPubKey, 1, 1, twentyBytes)
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgObject: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	// Num objectentory vectors (varInt) + max allowed objectentory vectors.
	wantPayload := uint32(100000000)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	return
}

// TestObjectWire tests the MsgObject bmwire.encode and decode for various numbers
// of objectentory vectors and protocol versions.
func TestObjectWire(t *testing.T) {

	twentyBytes := make([]byte, 20)
	expires := time.Unix(0x495fab29, 0) // 2009-01-03 12:15:05 -0600 CST)
	msg := bmwire.NewMsgObject(83928, expires, bmwire.ObjectTypeGetPubKey, 1, 1, twentyBytes)

	ObjectEncoded := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x47, 0xd8, // 83928 nonce
		0x00, 0x00, 0x00, 0x00, 0x49, 0x5f, 0xab, 0x29, // 64-bit timestamp
		0x00, 0x00, 0x00, 0x00, // object type (GETPUBKEY)
		0x01, // object version
		0x01, // stream number
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // 20-byte ripemd
	}

	tests := []struct {
		in   *bmwire.MsgObject // Message to encode
		out  *bmwire.MsgObject // Expected decoded message
		buf  []byte            // Wire encoding
		pver uint32            // Protocol version for bmwire.encoding
	}{
		// Latest protocol version with multiple object vectors.
		{
			msg,
			msg,
			ObjectEncoded,
			bmwire.ProtocolVersion,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode the message to bmwire.format.
		var buf bytes.Buffer
		err := test.in.BtcEncode(&buf, test.pver)
		if err != nil {
			t.Errorf("BtcEncode #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("BtcEncode #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode the message from bmwire.format.
		var msg bmwire.MsgObject
		rbuf := bytes.NewReader(test.buf)
		err = msg.BtcDecode(rbuf, test.pver)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}
