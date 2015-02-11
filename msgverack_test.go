// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bmwire_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/jimmysong/bmwire"
	"github.com/davecgh/go-spew/spew"
)

// TestVerAck tests the MsgVerAck API.
func TestVerAck(t *testing.T) {
	pver := bmwire.ProtocolVersion

	// Ensure the command is expected value.
	wantCmd := "verack"
	msg := bmwire.NewMsgVerAck()
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgVerAck: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value.
	wantPayload := uint32(0)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	return
}

// TestVerAckWire tests the MsgVerAck bmwire.encode and decode for various
// protocol versions.
func TestVerAckWire(t *testing.T) {
	msgVerAck := bmwire.NewMsgVerAck()
	msgVerAckEncoded := []byte{}

	tests := []struct {
		in   *bmwire.MsgVerAck // Message to encode
		out  *bmwire.MsgVerAck // Expected decoded message
		buf  []byte          // Wire encoding
		pver uint32          // Protocol version for bmwire.encoding
	}{
		// Latest protocol version.
		{
			msgVerAck,
			msgVerAck,
			msgVerAckEncoded,
			bmwire.ProtocolVersion,
		},

		// Protocol version BIP0035Version.
		{
			msgVerAck,
			msgVerAck,
			msgVerAckEncoded,
			bmwire.BIP0035Version,
		},

		// Protocol version BIP0031Version.
		{
			msgVerAck,
			msgVerAck,
			msgVerAckEncoded,
			bmwire.BIP0031Version,
		},

		// Protocol version NetAddressTimeVersion.
		{
			msgVerAck,
			msgVerAck,
			msgVerAckEncoded,
			bmwire.NetAddressTimeVersion,
		},

		// Protocol version MultipleAddressVersion.
		{
			msgVerAck,
			msgVerAck,
			msgVerAckEncoded,
			bmwire.MultipleAddressVersion,
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
		var msg bmwire.MsgVerAck
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
