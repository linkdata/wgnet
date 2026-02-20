package wgnet

import (
	"context"
	"errors"
	"io"
	"net"
	"net/netip"
	"strconv"
	"testing"
	"time"
)

type fakeDialer struct {
	conn net.Conn
	err  error
}

func (d *fakeDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return d.conn, d.err
}

type fakeConn struct {
	writeErr   error
	readErr    error
	deadline   time.Time
	closeCalls int
}

func (c *fakeConn) Read(_ []byte) (n int, err error) { return 0, c.readErr }

func (c *fakeConn) Write(b []byte) (n int, err error) {
	err = c.writeErr
	if err == nil {
		n = len(b)
	}
	return
}

func (c *fakeConn) Close() error {
	c.closeCalls++
	return nil
}

func (c *fakeConn) LocalAddr() net.Addr                 { return nil }
func (c *fakeConn) RemoteAddr() net.Addr                { return nil }
func (c *fakeConn) SetDeadline(t time.Time) (err error) { c.deadline = t; return }
func (c *fakeConn) SetReadDeadline(time.Time) error     { return nil }
func (c *fakeConn) SetWriteDeadline(time.Time) error    { return nil }

func TestPing4WithDialer_ClosesSocketOnError(t *testing.T) {
	conn := &fakeConn{writeErr: io.ErrClosedPipe}
	dialer := &fakeDialer{conn: conn}
	_, err := ping4WithDialer(context.Background(), dialer, "127.0.0.1")
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected %v, got %v", io.ErrClosedPipe, err)
	}
	if conn.closeCalls != 1 {
		t.Fatalf("expected 1 close call, got %d", conn.closeCalls)
	}
}

func TestMustEndpoint_PrefersIPv4FromLookup(t *testing.T) {
	origLookupNetIP := endpointLookupNetIP
	endpointLookupNetIP = func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{
			netip.MustParseAddr("2001:db8::1"),
			netip.MustParseAddr("192.0.2.10"),
		}, nil
	}
	t.Cleanup(func() { endpointLookupNetIP = origLookupNetIP })

	endpoint, err := mustEndpoint("example.test:51820", ErrInvalidPeerEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	if !endpoint.IsValid() {
		t.Fatal("expected valid endpoint")
	}
	if !endpoint.Addr().Is4() {
		t.Fatalf("expected IPv4 endpoint, got %s", endpoint.Addr())
	}
	if endpoint.Addr() != netip.MustParseAddr("192.0.2.10") {
		t.Fatalf("endpoint addr = %s, want 192.0.2.10", endpoint.Addr())
	}
}

func TestMustEndpoint_PortOutOfRange(t *testing.T) {
	_, err := mustEndpoint("example.test:70000", ErrInvalidPeerEndpoint)
	if !errors.Is(err, ErrInvalidPeerEndpoint) {
		t.Fatalf("expected %v in error, got %v", ErrInvalidPeerEndpoint, err)
	}
	if !errors.Is(err, ErrInvalidPeerEndpointPort) {
		t.Fatalf("expected %v in error, got %v", ErrInvalidPeerEndpointPort, err)
	}
}

func TestMustEndpoint_PortNotNumeric(t *testing.T) {
	_, err := mustEndpoint("example.test:not-a-number", ErrInvalidPeerEndpoint)
	if !errors.Is(err, ErrInvalidPeerEndpoint) {
		t.Fatalf("expected %v in error, got %v", ErrInvalidPeerEndpoint, err)
	}
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("expected strconv.NumError in error chain, got %v", err)
	}
}
