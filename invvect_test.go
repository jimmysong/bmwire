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

// TestInvVect tests the InvVect API.
func TestInvVect(t *testing.T) {
	hash := bmwire.ShaHash{}

	// Ensure we get the same payload and signature back out.
	iv := bmwire.NewInvVect(&hash)
	if !iv.Hash.IsEqual(&hash) {
		t.Errorf("NewInvVect: wrong hash - got %v, want %v",
			spew.Sdump(iv.Hash), spew.Sdump(hash))
	}

}

// TestInvVectWire tests the InvVect bmwire.encode and decode for various
// protocol versions and supported inventory vector types.
func TestInvVectWire(t *testing.T) {
	// Block 203707 hash.
	hashStr := "3264bc2ac36a60840790ba1d475d01367e7c723da941069e9dc"
	baseHash, err := bmwire.NewShaHashFromStr(hashStr)
	if err != nil {
		t.Errorf("NewShaHashFromStr: %v", err)
	}

	// errInvVect is an inventory vector with an error.
	errInvVect := bmwire.InvVect{
		Hash: bmwire.ShaHash{},
	}

	// errInvVectEncoded is the bmwire.encoded bytes of errInvVect.
	errInvVectEncoded := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // No hash
	}

	// txInvVect is an inventory vector representing a transaction.
	txInvVect := bmwire.InvVect{
		Hash: *baseHash,
	}

	// txInvVectEncoded is the bmwire.encoded bytes of txInvVect.
	txInvVectEncoded := []byte{
		0xdc, 0xe9, 0x69, 0x10, 0x94, 0xda, 0x23, 0xc7,
		0xe7, 0x67, 0x13, 0xd0, 0x75, 0xd4, 0xa1, 0x0b,
		0x79, 0x40, 0x08, 0xa6, 0x36, 0xac, 0xc2, 0x4b,
		0x26, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Block 203707 hash
	}

	// blockInvVect is an inventory vector representing a block.
	blockInvVect := bmwire.InvVect{
		Hash: *baseHash,
	}

	// blockInvVectEncoded is the bmwire.encoded bytes of blockInvVect.
	blockInvVectEncoded := []byte{
		0xdc, 0xe9, 0x69, 0x10, 0x94, 0xda, 0x23, 0xc7,
		0xe7, 0x67, 0x13, 0xd0, 0x75, 0xd4, 0xa1, 0x0b,
		0x79, 0x40, 0x08, 0xa6, 0x36, 0xac, 0xc2, 0x4b,
		0x26, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Block 203707 hash
	}

	tests := []struct {
		in  bmwire.InvVect // NetAddress to encode
		out bmwire.InvVect // Expected decoded NetAddress
		buf []byte         // Wire encoding
	}{
		// Latest protocol version error inventory vector.
		{
			errInvVect,
			errInvVect,
			errInvVectEncoded,
		},

		// Latest protocol version tx inventory vector.
		{
			txInvVect,
			txInvVect,
			txInvVectEncoded,
		},

		// Latest protocol version block inventory vector.
		{
			blockInvVect,
			blockInvVect,
			blockInvVectEncoded,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to bmwire.format.
		var buf bytes.Buffer
		err := bmwire.TstWriteInvVect(&buf, &test.in)
		if err != nil {
			t.Errorf("writeInvVect #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("writeInvVect #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode the message from bmwire.format.
		var iv bmwire.InvVect
		rbuf := bytes.NewReader(test.buf)
		err = bmwire.TstReadInvVect(rbuf, &iv)
		if err != nil {
			t.Errorf("readInvVect #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(iv, test.out) {
			t.Errorf("readInvVect #%d\n got: %s want: %s", i,
				spew.Sdump(iv), spew.Sdump(test.out))
			continue
		}
	}
}
