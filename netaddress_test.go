package bmwire_test

import (
	"bytes"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jimmysong/bmwire"
)

// TestNetAddress tests the NetAddress API.
func TestNetAddress(t *testing.T) {
	ip := net.ParseIP("127.0.0.1")
	port := 8333

	// Test NewNetAddress.
	tcpAddr := &net.TCPAddr{
		IP:   ip,
		Port: port,
	}
	na, err := bmwire.NewNetAddress(tcpAddr, 0)
	if err != nil {
		t.Errorf("NewNetAddress: %v", err)
	}

	// Ensure we get the same ip, port, and services back out.
	if !na.IP.Equal(ip) {
		t.Errorf("NetNetAddress: wrong ip - got %v, want %v", na.IP, ip)
	}
	if na.Port != uint16(port) {
		t.Errorf("NetNetAddress: wrong port - got %v, want %v", na.Port,
			port)
	}
	if na.Services != 0 {
		t.Errorf("NetNetAddress: wrong services - got %v, want %v",
			na.Services, 0)
	}
	if na.HasService(bmwire.SFNodeNetwork) {
		t.Errorf("HasService: SFNodeNetwork service is set")
	}

	// Ensure adding the full service node flag works.
	na.AddService(bmwire.SFNodeNetwork)
	if na.Services != bmwire.SFNodeNetwork {
		t.Errorf("AddService: wrong services - got %v, want %v",
			na.Services, bmwire.SFNodeNetwork)
	}
	if !na.HasService(bmwire.SFNodeNetwork) {
		t.Errorf("HasService: SFNodeNetwork service not set")
	}

	// Ensure max payload is expected value for latest protocol version.
	wantPayload := uint32(30)
	maxPayload := bmwire.TstMaxNetAddressPayload()
	if maxPayload != wantPayload {
		t.Errorf("maxNetAddressPayload: wrong max payload length for "+
			"- got %v, want %v", maxPayload, wantPayload)
	}

	// Check for expected failure on wrong address type.
	udpAddr := &net.UDPAddr{}
	_, err = bmwire.NewNetAddress(udpAddr, 0)
	if err != bmwire.ErrInvalidNetAddr {
		t.Errorf("NewNetAddress: expected error not received - "+
			"got %v, want %v", err, bmwire.ErrInvalidNetAddr)
	}
}

// TestNetAddressWire tests the NetAddress bmwire.encode and decode for various
// protocol versions and timestamp flag combinations.
func TestNetAddressWire(t *testing.T) {
	// baseNetAddr is used in the various tests as a baseline NetAddress.
	baseNetAddr := bmwire.NetAddress{
		Timestamp: time.Unix(0x495fab29, 0), // 2009-01-03 12:15:05 -0600 CST
		Services:  bmwire.SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	}

	// baseNetAddrNoTS is baseNetAddr with a zero value for the timestamp.
	baseNetAddrNoTS := baseNetAddr
	baseNetAddrNoTS.Timestamp = time.Time{}

	// baseNetAddrEncoded is the bmwire.encoded bytes of baseNetAddr.
	baseNetAddrEncoded := []byte{
		0x49, 0x5f, 0xab, 0x29, // Timestamp
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // SFNodeNetwork
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, // IP 127.0.0.1
		0x20, 0x8d, // Port 8333 in big-endian
	}

	// baseNetAddrNoTSEncoded is the bmwire.encoded bytes of baseNetAddrNoTS.
	baseNetAddrNoTSEncoded := []byte{
		// No timestamp
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // SFNodeNetwork
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x01, // IP 127.0.0.1
		0x20, 0x8d, // Port 8333 in big-endian
	}

	tests := []struct {
		in  bmwire.NetAddress // NetAddress to encode
		out bmwire.NetAddress // Expected decoded NetAddress
		ts  bool              // Include timestamp?
		buf []byte            // Wire encoding
	}{
		// Latest protocol version without ts flag.
		{
			baseNetAddr,
			baseNetAddrNoTS,
			false,
			baseNetAddrNoTSEncoded,
		},

		// Latest protocol version with ts flag.
		{
			baseNetAddr,
			baseNetAddr,
			true,
			baseNetAddrEncoded,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to bmwire.format.
		var buf bytes.Buffer
		err := bmwire.TstWriteNetAddress(&buf, &test.in, test.ts)
		if err != nil {
			t.Errorf("writeNetAddress #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("writeNetAddress #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode the message from bmwire.format.
		var na bmwire.NetAddress
		rbuf := bytes.NewReader(test.buf)
		err = bmwire.TstReadNetAddress(rbuf, &na, test.ts)
		if err != nil {
			t.Errorf("readNetAddress #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(na, test.out) {
			t.Errorf("readNetAddress #%d\n got: %s want: %s", i,
				spew.Sdump(na), spew.Sdump(test.out))
			continue
		}
	}
}

// TestNetAddressWireErrors performs negative tests against bmwire.encode and
// decode NetAddress to confirm error paths work correctly.
func TestNetAddressWireErrors(t *testing.T) {

	// baseNetAddr is used in the various tests as a baseline NetAddress.
	baseNetAddr := bmwire.NetAddress{
		Timestamp: time.Unix(0x495fab29, 0), // 2009-01-03 12:15:05 -0600 CST
		Services:  bmwire.SFNodeNetwork,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      8333,
	}

	tests := []struct {
		in       *bmwire.NetAddress // Value to encode
		buf      []byte             // Wire encoding
		ts       bool               // Include timestamp flag
		max      int                // Max size of fixed buffer to induce errors
		writeErr error              // Expected write error
		readErr  error              // Expected read error
	}{
		// Latest protocol version with timestamp and intentional
		// read/write errors.
		// Force errors on timestamp.
		{&baseNetAddr, []byte{}, true, 0, io.ErrShortWrite, io.EOF},
		// Force errors on services.
		{&baseNetAddr, []byte{}, true, 4, io.ErrShortWrite, io.EOF},
		// Force errors on ip.
		{&baseNetAddr, []byte{}, true, 12, io.ErrShortWrite, io.EOF},
		// Force errors on port.
		{&baseNetAddr, []byte{}, true, 28, io.ErrShortWrite, io.EOF},

		// Latest protocol version with no timestamp and intentional
		// read/write errors.
		// Force errors on services.
		{&baseNetAddr, []byte{}, false, 0, io.ErrShortWrite, io.EOF},
		// Force errors on ip.
		{&baseNetAddr, []byte{}, false, 8, io.ErrShortWrite, io.EOF},
		// Force errors on port.
		{&baseNetAddr, []byte{}, false, 24, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to bmwire.format.
		w := newFixedWriter(test.max)
		err := bmwire.TstWriteNetAddress(w, test.in, test.ts)
		if err != test.writeErr {
			t.Errorf("writeNetAddress #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		// Decode from bmwire.format.
		var na bmwire.NetAddress
		r := newFixedReader(test.max, test.buf)
		err = bmwire.TstReadNetAddress(r, &na, test.ts)
		if err != test.readErr {
			t.Errorf("readNetAddress #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}
