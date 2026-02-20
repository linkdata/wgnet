package wgnet

import (
	"context"
	"errors"
	"io"
	"net"
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
