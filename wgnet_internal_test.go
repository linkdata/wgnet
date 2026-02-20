package wgnet

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
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

type loadwaiter struct {
	underLoad atomic.Bool
	closed    atomic.Bool
}

func (lw *loadwaiter) IsUnderLoad() bool { return lw.underLoad.Load() }
func (lw *loadwaiter) Close()            { lw.closed.Store(true) }

func TestWaitForNoLoad(t *testing.T) {
	var lw loadwaiter
	lw.underLoad.Store(true)
	go func() {
		time.Sleep(time.Millisecond * 50)
		lw.underLoad.Store(false)
	}()
	waitForNoLoad(&lw, time.Millisecond, time.Millisecond*10, time.Millisecond*100)
	if !lw.closed.Load() {
		t.Fatal("expected device close after no-load period")
	}
}

func TestWaitForNoLoad_ClosesAtMaxTime(t *testing.T) {
	var lw loadwaiter
	lw.underLoad.Store(true)
	waitForNoLoad(&lw, time.Millisecond, time.Millisecond*10, time.Millisecond*20)
	if !lw.closed.Load() {
		t.Fatal("expected device close at max wait time")
	}
}
