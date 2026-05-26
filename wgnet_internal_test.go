package wgnet

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync"
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

type loopbackPingConn struct {
	fakeConn
	payload []byte
}

func (c *loopbackPingConn) Read(b []byte) (n int, err error) {
	err = c.readErr
	if err == nil {
		n = copy(b, c.payload)
	}
	return
}

func (c *loopbackPingConn) Write(b []byte) (n int, err error) {
	err = c.writeErr
	if err == nil {
		c.payload = append(c.payload[:0], b...)
		n = len(b)
	}
	return
}

type cancelPingConn struct {
	fakeConn
	readStarted     chan struct{}
	deadlineChanged chan time.Time
	closeRead       chan struct{}
	startOnce       sync.Once
	closeOnce       sync.Once
}

func newCancelPingConn() *cancelPingConn {
	return &cancelPingConn{
		readStarted:     make(chan struct{}),
		deadlineChanged: make(chan time.Time, 4),
		closeRead:       make(chan struct{}),
	}
}

func (c *cancelPingConn) Read(_ []byte) (n int, err error) {
	c.startOnce.Do(func() { close(c.readStarted) })
	for {
		select {
		case deadline := <-c.deadlineChanged:
			if !deadline.IsZero() && !deadline.After(time.Now()) {
				err = os.ErrDeadlineExceeded
				return
			}
		case <-c.closeRead:
			err = net.ErrClosed
			return
		}
	}
}

func (c *cancelPingConn) SetDeadline(t time.Time) (err error) {
	err = c.fakeConn.SetDeadline(t)
	if err == nil {
		select {
		case c.deadlineChanged <- t:
		default:
		}
	}
	return
}

func (c *cancelPingConn) Close() (err error) {
	c.closeOnce.Do(func() { close(c.closeRead) })
	err = c.fakeConn.Close()
	return
}

func TestPing4WithDialer_ClosesSocketOnError(t *testing.T) {
	conn := &fakeConn{writeErr: io.ErrClosedPipe}
	dialer := &fakeDialer{conn: conn}
	_, err := ping4WithDialer(t.Context(), dialer, "127.0.0.1")
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected %v, got %v", io.ErrClosedPipe, err)
	}
	if conn.closeCalls != 1 {
		t.Fatalf("expected 1 close call, got %d", conn.closeCalls)
	}
}

func TestPing4WithDialer_RejectsEchoRequestPacket(t *testing.T) {
	conn := &loopbackPingConn{}
	dialer := &fakeDialer{conn: conn}
	_, err := ping4WithDialer(t.Context(), dialer, "127.0.0.1")
	if !errors.Is(err, ErrInvalidPingReply) {
		t.Fatalf("expected %v, got %v", ErrInvalidPingReply, err)
	}
	if conn.closeCalls != 1 {
		t.Fatalf("expected 1 close call, got %d", conn.closeCalls)
	}
}

func TestPing4WithDialer_CanceledContextUnblocksRead(t *testing.T) {
	conn := newCancelPingConn()
	dialer := &fakeDialer{conn: conn}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		_, err := ping4WithDialer(ctx, dialer, "127.0.0.1")
		done <- err
	}()

	select {
	case <-conn.readStarted:
	case <-time.After(time.Second):
		_ = conn.Close()
		t.Fatal("timed out waiting for ping read")
	}

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected %v, got %v", context.Canceled, err)
		}
	case <-time.After(time.Second):
		_ = conn.Close()
		t.Fatal("timed out waiting for canceled ping")
	}

	if conn.closeCalls != 1 {
		t.Fatalf("expected 1 close call, got %d", conn.closeCalls)
	}
}

func TestPing4WithDialer_CapsDeadlineToTenSeconds(t *testing.T) {
	conn := &fakeConn{writeErr: io.ErrClosedPipe}
	dialer := &fakeDialer{conn: conn}

	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(time.Hour))
	defer cancel()

	_, err := ping4WithDialer(ctx, dialer, "127.0.0.1")
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected %v, got %v", io.ErrClosedPipe, err)
	}

	remaining := time.Until(conn.deadline)
	if remaining < time.Second*7 || remaining > time.Second*13 {
		t.Fatalf("expected deadline near 10s from now, got %s", remaining)
	}
}

func TestPing4WithDialer_UsesEarlierContextDeadline(t *testing.T) {
	conn := &fakeConn{writeErr: io.ErrClosedPipe}
	dialer := &fakeDialer{conn: conn}

	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(time.Second*2))
	defer cancel()

	_, err := ping4WithDialer(ctx, dialer, "127.0.0.1")
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected %v, got %v", io.ErrClosedPipe, err)
	}

	remaining := time.Until(conn.deadline)
	if remaining < time.Second || remaining > time.Second*3 {
		t.Fatalf("expected deadline near 2s from now, got %s", remaining)
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
