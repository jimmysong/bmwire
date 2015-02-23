// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bmwire_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/jimmysong/bmwire"
)

// TestGetAddr tests the MsgGetAddr API.
func TestGetAddr(t *testing.T) {
	pver := bmwire.ProtocolVersion

	// Ensure the command is expected value.
	wantCmd := "getaddr"
	msg := bmwire.NewMsgGetAddr()
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgGetAddr: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value for latest protocol version.
	// Num addresses (varInt) + max allowed addresses.
	wantPayload := uint32(0)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	return
}

// TestGetAddrWire tests the MsgGetAddr bmwire.encode and decode for various
// protocol versions.
func TestGetAddrWire(t *testing.T) {
	msgGetAddr := bmwire.NewMsgGetAddr()
	msgGetAddrEncoded := []byte{}

	tests := []struct {
		in   *bmwire.MsgGetAddr // Message to encode
		out  *bmwire.MsgGetAddr // Expected decoded message
		buf  []byte             // Wire encoding
		pver uint32             // Protocol version for bmwire.encoding
	}{
		// Latest protocol version.
		{
			msgGetAddr,
			msgGetAddr,
			msgGetAddrEncoded,
			bmwire.ProtocolVersion,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode the message to bmwire.format.
		var buf bytes.Buffer
		err := test.in.Encode(&buf, test.pver)
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
		var msg bmwire.MsgGetAddr
		rbuf := bytes.NewReader(test.buf)
		err = msg.Decode(rbuf, test.pver)
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
