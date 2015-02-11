// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bmwire_test

import (
	"bytes"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jimmysong/bmwire"
)

// TestVersion tests the MsgVersion API.
func TestVersion(t *testing.T) {
	pver := bmwire.ProtocolVersion

	// Create version message data.
	tcpAddrMe := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	me, err := bmwire.NewNetAddress(tcpAddrMe, bmwire.SFNodeNetwork)
	if err != nil {
		t.Errorf("NewNetAddress: %v", err)
	}
	tcpAddrYou := &net.TCPAddr{IP: net.ParseIP("192.168.0.1"), Port: 8333}
	you, err := bmwire.NewNetAddress(tcpAddrYou, bmwire.SFNodeNetwork)
	if err != nil {
		t.Errorf("NewNetAddress: %v", err)
	}
	nonce, err := bmwire.RandomUint64()
	if err != nil {
		t.Errorf("RandomUint64: error generating nonce: %v", err)
	}

	// Ensure we get the correct data back out.
	msg := bmwire.NewMsgVersion(me, you, nonce, []uint64{1})
	if msg.ProtocolVersion != int32(pver) {
		t.Errorf("NewMsgVersion: wrong protocol version - got %v, want %v",
			msg.ProtocolVersion, pver)
	}
	if !reflect.DeepEqual(&msg.AddrMe, me) {
		t.Errorf("NewMsgVersion: wrong me address - got %v, want %v",
			spew.Sdump(&msg.AddrMe), spew.Sdump(me))
	}
	if !reflect.DeepEqual(&msg.AddrYou, you) {
		t.Errorf("NewMsgVersion: wrong you address - got %v, want %v",
			spew.Sdump(&msg.AddrYou), spew.Sdump(you))
	}
	if msg.Nonce != nonce {
		t.Errorf("NewMsgVersion: wrong nonce - got %v, want %v",
			msg.Nonce, nonce)
	}
	if msg.UserAgent != bmwire.DefaultUserAgent {
		t.Errorf("NewMsgVersion: wrong user agent - got %v, want %v",
			msg.UserAgent, bmwire.DefaultUserAgent)
	}
	if len(msg.StreamNumbers) != 1 {
		t.Errorf("NewMsgVersion: wrong number of streams - got %v, want %v",
			len(msg.StreamNumbers), 1)
	}

	if msg.StreamNumbers[0] != 1 {
		t.Errorf("NewMsgVersion: wrong streams - got %v, want %v",
			msg.StreamNumbers[0], 1)
	}

	msg.AddUserAgent("myclient", "1.2.3", "optional", "comments")
	customUserAgent := bmwire.DefaultUserAgent + "myclient:1.2.3(optional; comments)/"
	if msg.UserAgent != customUserAgent {
		t.Errorf("AddUserAgent: wrong user agent - got %s, want %s",
			msg.UserAgent, customUserAgent)
	}

	msg.AddUserAgent("mygui", "3.4.5")
	customUserAgent += "mygui:3.4.5/"
	if msg.UserAgent != customUserAgent {
		t.Errorf("AddUserAgent: wrong user agent - got %s, want %s",
			msg.UserAgent, customUserAgent)
	}

	// accounting for ":", "/"
	err = msg.AddUserAgent(strings.Repeat("t",
		bmwire.MaxUserAgentLen-len(customUserAgent)-2+1), "")
	if _, ok := err.(*bmwire.MessageError); !ok {
		t.Errorf("AddUserAgent: expected error not received "+
			"- got %v, want %T", err, bmwire.MessageError{})

	}

	// Version message should not have any services set by default.
	if msg.Services != 0 {
		t.Errorf("NewMsgVersion: wrong default services - got %v, want %v",
			msg.Services, 0)

	}
	if msg.HasService(bmwire.SFNodeNetwork) {
		t.Errorf("HasService: SFNodeNetwork service is not set")
	}
	msg.AddService(bmwire.SFNodeNetwork)
	if !msg.HasService(bmwire.SFNodeNetwork) {
		t.Errorf("HasService: SFNodeNetwork service is not set")
	}

	// Ensure the command is expected value.
	wantCmd := "version"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgVersion: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value.
	// Protocol version 4 bytes + services 8 bytes + timestamp 8 bytes +
	// remote and local net addresses + nonce 8 bytes + length of user agent
	// (varInt) + max allowed user agent length + last block 4 bytes +
	// relay transactions flag 1 byte.
	wantPayload := uint32(2102)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	// Ensure adding the full service node flag works.
	msg.AddService(bmwire.SFNodeNetwork)
	if msg.Services != bmwire.SFNodeNetwork {
		t.Errorf("AddService: wrong services - got %v, want %v",
			msg.Services, bmwire.SFNodeNetwork)
	}
	if !msg.HasService(bmwire.SFNodeNetwork) {
		t.Errorf("HasService: SFNodeNetwork service not set")
	}

	// Use a fake connection.
	conn := &fakeConn{localAddr: tcpAddrMe, remoteAddr: tcpAddrYou}
	msg, err = bmwire.NewMsgVersionFromConn(conn, nonce, []uint64{1})
	if err != nil {
		t.Errorf("NewMsgVersionFromConn: %v", err)
	}

	// Ensure we get the correct connection data back out.
	if !msg.AddrMe.IP.Equal(tcpAddrMe.IP) {
		t.Errorf("NewMsgVersionFromConn: wrong me ip - got %v, want %v",
			msg.AddrMe.IP, tcpAddrMe.IP)
	}
	if !msg.AddrYou.IP.Equal(tcpAddrYou.IP) {
		t.Errorf("NewMsgVersionFromConn: wrong you ip - got %v, want %v",
			msg.AddrYou.IP, tcpAddrYou.IP)
	}

	// Use a fake connection with local UDP addresses to force a failure.
	conn = &fakeConn{
		localAddr:  &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333},
		remoteAddr: tcpAddrYou,
	}
	msg, err = bmwire.NewMsgVersionFromConn(conn, nonce, []uint64{1})
	if err != bmwire.ErrInvalidNetAddr {
		t.Errorf("NewMsgVersionFromConn: expected error not received "+
			"- got %v, want %v", err, bmwire.ErrInvalidNetAddr)
	}

	// Use a fake connection with remote UDP addresses to force a failure.
	conn = &fakeConn{
		localAddr:  tcpAddrMe,
		remoteAddr: &net.UDPAddr{IP: net.ParseIP("192.168.0.1"), Port: 8333},
	}
	msg, err = bmwire.NewMsgVersionFromConn(conn, nonce, []uint64{1})
	if err != bmwire.ErrInvalidNetAddr {
		t.Errorf("NewMsgVersionFromConn: expected error not received "+
			"- got %v, want %v", err, bmwire.ErrInvalidNetAddr)
	}

	return
}

// TestVersionWire tests the MsgVersion bmwire.encode and decode for various
// protocol versions.
func TestVersionWire(t *testing.T) {
	// verRelayTxFalse and verRelayTxFalseEncoded is a version message as of
	// BIP0037Version with the transaction relay disabled.
	verRelayTxFalseEncoded := make([]byte, len(baseVersionEncoded))
	copy(verRelayTxFalseEncoded, baseVersionEncoded)
	verRelayTxFalseEncoded[len(verRelayTxFalseEncoded)-1] = 0

	tests := []struct {
		in   *bmwire.MsgVersion // Message to encode
		out  *bmwire.MsgVersion // Expected decoded message
		buf  []byte             // Wire encoding
		pver uint32             // Protocol version for bmwire.encoding
	}{
		// Latest protocol version.
		{
			baseVersion,
			baseVersion,
			baseVersionEncoded,
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
		var msg bmwire.MsgVersion
		rbuf := bytes.NewBuffer(test.buf)
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

// TestVersionWireErrors performs negative tests against bmwire.encode and
// decode of MsgGetHeaders to confirm error paths work correctly.
func TestVersionWireErrors(t *testing.T) {
	// Use protocol version 60002 specifically here instead of the latest
	// because the test data is using bytes encoded with that protocol
	// version.
	pver := uint32(60002)
	//	wireErr := &bmwire.MessageError{}

	// Ensure calling MsgVersion.BtcDecode with a non *bytes.Buffer returns
	// error.
	fr := newFixedReader(0, []byte{})
	if err := baseVersion.BtcDecode(fr, pver); err == nil {
		t.Errorf("Did not received error when calling " +
			"MsgVersion.BtcDecode with non *bytes.Buffer")
	}

	// Copy the base version and change the user agent to exceed max limits.
	bvc := *baseVersion
	exceedUAVer := &bvc
	newUA := "/" + strings.Repeat("t", bmwire.MaxUserAgentLen-8+1) + ":0.0.1/"
	exceedUAVer.UserAgent = newUA

	// Encode the new UA length as a varint.
	var newUAVarIntBuf bytes.Buffer
	err := bmwire.TstWriteVarInt(&newUAVarIntBuf, pver, uint64(len(newUA)))
	if err != nil {
		t.Errorf("writeVarInt: error %v", err)
	}

	// // Make a new buffer big enough to hold the base version plus the new
	// // bytes for the bigger varint to hold the new size of the user agent
	// // and the new user agent string.  Then stich it all together.
	// newLen := len(baseVersionEncoded) - len(baseVersion.UserAgent)
	// newLen = newLen + len(newUAVarIntBuf.Bytes()) - 1 + len(newUA)
	// exceedUAVerEncoded := make([]byte, newLen)
	// copy(exceedUAVerEncoded, baseVersionEncoded[0:80])
	// copy(exceedUAVerEncoded[80:], newUAVarIntBuf.Bytes())
	// copy(exceedUAVerEncoded[83:], []byte(newUA))
	// copy(exceedUAVerEncoded[83+len(newUA):], baseVersionEncoded[97:100])

	tests := []struct {
		in       *bmwire.MsgVersion // Value to encode
		buf      []byte             // Wire encoding
		pver     uint32             // Protocol version for bmwire.encoding
		max      int                // Max size of fixed buffer to induce errors
		writeErr error              // Expected write error
		readErr  error              // Expected read error
	}{
		// Force error in protocol version.
		{baseVersion, baseVersionEncoded, pver, 0, io.ErrShortWrite, io.EOF},
		// Force error in services.
		{baseVersion, baseVersionEncoded, pver, 4, io.ErrShortWrite, io.EOF},
		// Force error in timestamp.
		{baseVersion, baseVersionEncoded, pver, 12, io.ErrShortWrite, io.EOF},
		// Force error in remote address.
		{baseVersion, baseVersionEncoded, pver, 20, io.ErrShortWrite, io.EOF},
		// Force error in local address.
		{baseVersion, baseVersionEncoded, pver, 47, io.ErrShortWrite, io.ErrUnexpectedEOF},
		// Force error in nonce.
		{baseVersion, baseVersionEncoded, pver, 73, io.ErrShortWrite, io.ErrUnexpectedEOF},
		// Force error in user agent length.
		{baseVersion, baseVersionEncoded, pver, 81, io.ErrShortWrite, io.EOF},
		// Force error in user agent.
		{baseVersion, baseVersionEncoded, pver, 82, io.ErrShortWrite, io.ErrUnexpectedEOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to bmwire.format.
		w := newFixedWriter(test.max)
		err := test.in.BtcEncode(w, test.pver)
		if reflect.TypeOf(err) != reflect.TypeOf(test.writeErr) {
			t.Errorf("BtcEncode #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		// For errors which are not of type bmwire.MessageError, check
		// them for equality.
		if _, ok := err.(*bmwire.MessageError); !ok {
			if err != test.writeErr {
				t.Errorf("BtcEncode #%d wrong error got: %v, "+
					"want: %v", i, err, test.writeErr)
				continue
			}
		}

		// Decode from bmwire.format.
		var msg bmwire.MsgVersion
		buf := bytes.NewBuffer(test.buf[0:test.max])
		err = msg.BtcDecode(buf, test.pver)
		if reflect.TypeOf(err) != reflect.TypeOf(test.readErr) {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}

		// For errors which are not of type bmwire.MessageError, check
		// them for equality.
		if _, ok := err.(*bmwire.MessageError); !ok {
			if err != test.readErr {
				t.Errorf("BtcDecode #%d wrong error got: %v, "+
					"want: %v", i, err, test.readErr)
				continue
			}
		}
	}
}

// baseVersion is used in the various tests as a baseline MsgVersion.
var baseVersion = &bmwire.MsgVersion{
	ProtocolVersion: 3,
	Services:        bmwire.SFNodeNetwork,
	Timestamp:       time.Unix(0x495fab29, 0), // 2009-01-03 12:15:05 -0600 CST)
	AddrYou: bmwire.NetAddress{
		Timestamp: time.Time{}, // Zero value -- no timestamp in version
		Services:  bmwire.SFNodeNetwork,
		IP:        net.ParseIP("192.168.0.1"),
		Port:      8333,
	},
	AddrMe: bmwire.NetAddress{
		Timestamp: time.Time{}, // Zero value -- no timestamp in version
		Services:  bmwire.SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	},
	Nonce:         123123, // 0x1e0f3
	UserAgent:     "/btcdtest:0.0.1/",
	StreamNumbers: []uint64{1},
}

// baseVersionEncoded is the bmwire.encoded bytes for baseVersion using protocol
// version 60002 and is used in the various tests.
var baseVersionEncoded = []byte{
	0x00, 0x00, 0x00, 0x03, // Protocol version 60002
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // SFNodeNetwork
	0x00, 0x00, 0x00, 0x00, 0x49, 0x5f, 0xab, 0x29, // 64-bit Timestamp
	// AddrYou -- No timestamp for NetAddress in version message
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // SFNodeNetwork
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0xff, 0xff, 0xc0, 0xa8, 0x00, 0x01, // IP 192.168.0.1
	0x20, 0x8d, // Port 8333 in big-endian
	// AddrMe -- No timestamp for NetAddress in version message
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // SFNodeNetwork
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, // IP 127.0.0.1
	0x20, 0x8d, // Port 8333 in big-endian
	0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xe0, 0xf3, // Nonce
	0x10, // Varint for user agent length
	0x2f, 0x62, 0x74, 0x63, 0x64, 0x74, 0x65, 0x73,
	0x74, 0x3a, 0x30, 0x2e, 0x30, 0x2e, 0x31, 0x2f, // User agent
	0x01, 0x01, // Stream Numbers
}
