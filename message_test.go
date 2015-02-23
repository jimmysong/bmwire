// Copyright (c) 2013-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bmwire_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jimmysong/bmwire"
)

// makeHeader is a convenience function to make a message header in the form of
// a byte slice.  It is used to force errors when reading messages.
func makeHeader(bmnet bmwire.BitmessageNet, command string,
	payloadLen uint32, checksum uint32) []byte {

	// The length of a bitmessage message header is 24 bytes.
	// 4 byte magic number of the bitmessage network + 12 byte command + 4 byte
	// payload length + 4 byte checksum.
	buf := make([]byte, 24)
	binary.BigEndian.PutUint32(buf, uint32(bmnet))
	copy(buf[4:], []byte(command))
	binary.BigEndian.PutUint32(buf[16:], payloadLen)
	binary.BigEndian.PutUint32(buf[20:], checksum)
	return buf
}

// TestMessage tests the Read/WriteMessage and Read/WriteMessageN API.
func TestMessage(t *testing.T) {
	pver := bmwire.ProtocolVersion

	// Create the various types of messages to test.

	// MsgVersion.
	addrYou := &net.TCPAddr{IP: net.ParseIP("192.168.0.1"), Port: 8333}
	you, err := bmwire.NewNetAddress(addrYou, bmwire.SFNodeNetwork)
	if err != nil {
		t.Errorf("NewNetAddress: %v", err)
	}
	you.Timestamp = time.Time{} // Version message has zero value timestamp.
	addrMe := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	me, err := bmwire.NewNetAddress(addrMe, bmwire.SFNodeNetwork)
	if err != nil {
		t.Errorf("NewNetAddress: %v", err)
	}
	me.Timestamp = time.Time{} // Version message has zero value timestamp.
	msgVersion := bmwire.NewMsgVersion(me, you, 123123, []uint64{1})
	msgVerack := bmwire.NewMsgVerAck()
	msgGetAddr := bmwire.NewMsgGetAddr()
	msgAddr := bmwire.NewMsgAddr()
	msgInv := bmwire.NewMsgInv()
	msgGetData := bmwire.NewMsgGetData()

	// ripe-based getpubkey message
	var ripe [20]byte
	ripe[0] = 1
	var tag [32]byte
	expires := time.Unix(0x495fab29, 0) // 2009-01-03 12:15:05 -0600 CST)
	msgGetPubKey := bmwire.NewMsgGetPubKey(123123, expires, 2, 1, ripe, tag)

	tests := []struct {
		in     bmwire.Message       // Value to encode
		out    bmwire.Message       // Expected decoded value
		pver   uint32               // Protocol version for bmwire.encoding
		bmnet bmwire.BitmessageNet // Network to use for bmwire.encoding
		bytes  int                  // Expected num bytes read/written
	}{
		{msgVersion, msgVersion, pver, bmwire.MainNet, 121},
		{msgVerack, msgVerack, pver, bmwire.MainNet, 24},
		{msgGetAddr, msgGetAddr, pver, bmwire.MainNet, 24},
		{msgAddr, msgAddr, pver, bmwire.MainNet, 25},
		{msgInv, msgInv, pver, bmwire.MainNet, 25},
		{msgGetData, msgGetData, pver, bmwire.MainNet, 25},
		{msgGetPubKey, msgGetPubKey, pver, bmwire.MainNet, 66},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to bmwire.format.
		var buf bytes.Buffer
		nw, err := bmwire.WriteMessageN(&buf, test.in, test.pver, test.bmnet)
		if err != nil {
			t.Errorf("WriteMessage #%d error %v", i, err)
			continue
		}

		// Ensure the number of bytes written match the expected value.
		if nw != test.bytes {
			t.Errorf("WriteMessage #%d unexpected num bytes "+
				"written - got %d, want %d", i, nw, test.bytes)
		}

		// Decode from bmwire.format.
		rbuf := bytes.NewReader(buf.Bytes())
		nr, msg, _, err := bmwire.ReadMessageN(rbuf, test.pver, test.bmnet)
		if err != nil {
			t.Errorf("ReadMessage #%d error %v, msg %v", i, err,
				spew.Sdump(msg))
			continue
		}
		if !reflect.DeepEqual(msg, test.out) {
			t.Errorf("ReadMessage #%d\n got: %v want: %v", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}

		// Ensure the number of bytes read match the expected value.
		if nr != test.bytes {
			t.Errorf("ReadMessage #%d unexpected num bytes read - "+
				"got %d, want %d", i, nr, test.bytes)
		}
	}

	// Do the same thing for Read/WriteMessage, but ignore the bytes since
	// they don't return them.
	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to bmwire.format.
		var buf bytes.Buffer
		err := bmwire.WriteMessage(&buf, test.in, test.pver, test.bmnet)
		if err != nil {
			t.Errorf("WriteMessage #%d error %v", i, err)
			continue
		}

		// Decode from bmwire.format.
		rbuf := bytes.NewReader(buf.Bytes())
		msg, _, err := bmwire.ReadMessage(rbuf, test.pver, test.bmnet)
		if err != nil {
			t.Errorf("ReadMessage #%d error %v, msg %v", i, err,
				spew.Sdump(msg))
			continue
		}
		if !reflect.DeepEqual(msg, test.out) {
			t.Errorf("ReadMessage #%d\n got: %v want: %v", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}

// TestReadMessageWireErrors performs negative tests against bmwire.decoding into
// concrete messages to confirm error paths work correctly.
func TestReadMessageWireErrors(t *testing.T) {
	pver := bmwire.ProtocolVersion
	bmnet := bmwire.MainNet

	// Ensure message errors are as expected with no function specified.
	wantErr := "something bad happened"
	testErr := bmwire.MessageError{Description: wantErr}
	if testErr.Error() != wantErr {
		t.Errorf("MessageError: wrong error - got %v, want %v",
			testErr.Error(), wantErr)
	}

	// Ensure message errors are as expected with a function specified.
	wantFunc := "foo"
	testErr = bmwire.MessageError{Func: wantFunc, Description: wantErr}
	if testErr.Error() != wantFunc+": "+wantErr {
		t.Errorf("MessageError: wrong error - got %v, want %v",
			testErr.Error(), wantErr)
	}

	// Wire encoded bytes for a message that exceeds max overall message
	// length.
	mpl := uint32(bmwire.MaxMessagePayload)
	exceedMaxPayloadBytes := makeHeader(bmnet, "getaddr", mpl+1, 0)

	// Wire encoded bytes for a command which is invalid utf-8.
	badCommandBytes := makeHeader(bmnet, "bogus", 0, 0)
	badCommandBytes[4] = 0x81

	// Wire encoded bytes for a command which is valid, but not supported.
	unsupportedCommandBytes := makeHeader(bmnet, "bogus", 0, 0)

	// Wire encoded bytes for a message which exceeds the max payload for
	// a specific message type.
	exceedTypePayloadBytes := makeHeader(bmnet, "getaddr", 1, 0)

	// Wire encoded bytes for a message which does not deliver the full
	// payload according to the header length.
	shortPayloadBytes := makeHeader(bmnet, "version", 115, 0)

	// Wire encoded bytes for a message with a bad checksum.
	badChecksumBytes := makeHeader(bmnet, "version", 2, 0xbeef)
	badChecksumBytes = append(badChecksumBytes, []byte{0x0, 0x0}...)

	// Wire encoded bytes for a message which has a valid header, but is
	// the wrong format.  An addr starts with a varint of the number of
	// contained in the message.  Claim there is two, but don't provide
	// them.  At the same time, forge the header fields so the message is
	// otherwise accurate.
	badMessageBytes := makeHeader(bmnet, "addr", 1, 0xfab848c9)
	badMessageBytes = append(badMessageBytes, 0x2)

	// Wire encoded bytes for a message which the header claims has 15k
	// bytes of data to discard.
	discardBytes := makeHeader(bmnet, "bogus", 15*1024, 0)

	// wrong network bytes
	wrongNetBytes := makeHeader(0x09090909, "", 0, 0)

	tests := []struct {
		buf     []byte               // Wire encoding
		pver    uint32               // Protocol version for bmwire.encoding
		bmnet  bmwire.BitmessageNet // Bitmessage network for bmwire.encoding
		max     int                  // Max size of fixed buffer to induce errors
		readErr error                // Expected read error
		bytes   int                  // Expected num bytes read
	}{
		// Latest protocol version with intentional read errors.

		// Short header.
		{
			[]byte{},
			pver,
			bmnet,
			0,
			io.EOF,
			0,
		},

		// Wrong network.  Want MainNet, but giving wrong network.
		{
			wrongNetBytes,
			pver,
			bmnet,
			len(wrongNetBytes),
			&bmwire.MessageError{},
			24,
		},

		// Exceed max overall message payload length.
		{
			exceedMaxPayloadBytes,
			pver,
			bmnet,
			len(exceedMaxPayloadBytes),
			&bmwire.MessageError{},
			24,
		},

		// Invalid UTF-8 command.
		{
			badCommandBytes,
			pver,
			bmnet,
			len(badCommandBytes),
			&bmwire.MessageError{},
			24,
		},

		// Valid, but unsupported command.
		{
			unsupportedCommandBytes,
			pver,
			bmnet,
			len(unsupportedCommandBytes),
			&bmwire.MessageError{},
			24,
		},

		// Exceed max allowed payload for a message of a specific type.
		{
			exceedTypePayloadBytes,
			pver,
			bmnet,
			len(exceedTypePayloadBytes),
			&bmwire.MessageError{},
			24,
		},

		// Message with a payload shorter than the header indicates.
		{
			shortPayloadBytes,
			pver,
			bmnet,
			len(shortPayloadBytes),
			io.EOF,
			24,
		},

		// Message with a bad checksum.
		{
			badChecksumBytes,
			pver,
			bmnet,
			len(badChecksumBytes),
			&bmwire.MessageError{},
			26,
		},

		// Message with a valid header, but wrong format.
		{
			badMessageBytes,
			pver,
			bmnet,
			len(badMessageBytes),
			io.EOF,
			25,
		},

		// 15k bytes of data to discard.
		{
			discardBytes,
			pver,
			bmnet,
			len(discardBytes),
			&bmwire.MessageError{},
			24,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Decode from bmwire.format.
		r := newFixedReader(test.max, test.buf)
		nr, _, _, err := bmwire.ReadMessageN(r, test.pver, test.bmnet)
		if reflect.TypeOf(err) != reflect.TypeOf(test.readErr) {
			t.Errorf("ReadMessage #%d wrong error got: %v <%T>, "+
				"want: %T", i, err, err, test.readErr)
			continue
		}

		// Ensure the number of bytes written match the expected value.
		if nr != test.bytes {
			t.Errorf("ReadMessage #%d unexpected num bytes read - "+
				"got %d, want %d", i, nr, test.bytes)
		}

		// For errors which are not of type bmwire.MessageError, check
		// them for equality.
		if _, ok := err.(*bmwire.MessageError); !ok {
			if err != test.readErr {
				t.Errorf("ReadMessage #%d wrong error got: %v <%T>, "+
					"want: %v <%T>", i, err, err,
					test.readErr, test.readErr)
				continue
			}
		}
	}
}

// TestWriteMessageWireErrors performs negative tests against bmwire.encoding from
// concrete messages to confirm error paths work correctly.
func TestWriteMessageWireErrors(t *testing.T) {
	pver := bmwire.ProtocolVersion
	bmnet := bmwire.MainNet
	wireErr := &bmwire.MessageError{}

	// Fake message with a command that is too long.
	badCommandMsg := &fakeMessage{command: "somethingtoolong"}

	// Fake message with a problem during encoding
	encodeErrMsg := &fakeMessage{forceEncodeErr: true}

	// Fake message that has payload which exceeds max overall message size.
	exceedOverallPayload := make([]byte, bmwire.MaxMessagePayload+1)
	exceedOverallPayloadErrMsg := &fakeMessage{payload: exceedOverallPayload}

	// Fake message that has payload which exceeds max allowed per message.
	exceedPayload := make([]byte, 1)
	exceedPayloadErrMsg := &fakeMessage{payload: exceedPayload, forceLenErr: true}

	// Fake message that is used to force errors in the header and payload
	// writes.
	bogusPayload := []byte{0x01, 0x02, 0x03, 0x04}
	bogusMsg := &fakeMessage{command: "bogus", payload: bogusPayload}

	tests := []struct {
		msg    bmwire.Message       // Message to encode
		pver   uint32               // Protocol version for bmwire.encoding
		bmnet bmwire.BitmessageNet // Bitmessage network for bmwire.encoding
		max    int                  // Max size of fixed buffer to induce errors
		err    error                // Expected error
		bytes  int                  // Expected num bytes written
	}{
		// Command too long.
		{badCommandMsg, pver, bmnet, 0, wireErr, 0},
		// Force error in payload encode.
		{encodeErrMsg, pver, bmnet, 0, wireErr, 0},
		// Force error due to exceeding max overall message payload size.
		{exceedOverallPayloadErrMsg, pver, bmnet, 0, wireErr, 0},
		// Force error due to exceeding max payload for message type.
		{exceedPayloadErrMsg, pver, bmnet, 0, wireErr, 0},
		// Force error in header write.
		{bogusMsg, pver, bmnet, 0, io.ErrShortWrite, 0},
		// Force error in payload write.
		{bogusMsg, pver, bmnet, 24, io.ErrShortWrite, 24},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode bmwire.format.
		w := newFixedWriter(test.max)
		nw, err := bmwire.WriteMessageN(w, test.msg, test.pver, test.bmnet)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("WriteMessage #%d wrong error got: %v <%T>, "+
				"want: %T", i, err, err, test.err)
			continue
		}

		// Ensure the number of bytes written match the expected value.
		if nw != test.bytes {
			t.Errorf("WriteMessage #%d unexpected num bytes "+
				"written - got %d, want %d", i, nw, test.bytes)
		}

		// For errors which are not of type bmwire.MessageError, check
		// them for equality.
		if _, ok := err.(*bmwire.MessageError); !ok {
			if err != test.err {
				t.Errorf("ReadMessage #%d wrong error got: %v <%T>, "+
					"want: %v <%T>", i, err, err,
					test.err, test.err)
				continue
			}
		}
	}
}
