package wgnet_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	mrand "math/rand/v2"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/linkdata/wgnet"
)

var serverConfig = `[Interface]
PrivateKey = GInruesHOogIjjFsKCorYEAENAfYfPL/yH8ObFgyFUs=
ListenPort = %d
Address = 10.131.132.1/24

[Peer]
PublicKey = kTUQWHx4Y3ZYMZQPnRarzlx0qnen3plDoI0z7s45in4=
AllowedIPs = 10.131.132.2/32
`

var clientConfig = `[Interface]
PrivateKey = AEnvL9tVr+7JF0sMVjjzPjIxrrc/hoVJ5B82WWpVamI=
Address = 10.131.132.2/24
DNS = 1.1.1.1

[Peer]
PublicKey = Wh3yY7/fE3fyHJ8TOwLJ//CIRbgrlVl4bLQ+npNBSRU=
Endpoint = 127.0.0.1:%d
AllowedIPs = 0.0.0.0/0, ::/0
`

var nextListenPort = 10000 + mrand.IntN(1000)

func makeNets() (srv, cli *wgnet.WgNet) {
	listenPort := nextListenPort
	nextListenPort++
	if nextListenPort > 65000 {
		nextListenPort = 10000
	}
	var err error
	var srvCfg, cliCfg *wgnet.Config
	if srvCfg, err = wgnet.Parse(strings.NewReader(fmt.Sprintf(serverConfig, listenPort)), nil); err == nil {
		srv = wgnet.New(srvCfg)
		if cliCfg, err = wgnet.Parse(strings.NewReader(fmt.Sprintf(clientConfig, listenPort)), nil); err == nil {
			cli = wgnet.New(cliCfg)
			if err = srv.Open(); err == nil {
				if err = cli.Open(); err == nil {
					return
				}
				_ = srv.Close()
			}
		}
	}
	if err != nil {
		panic(err)
	}
	return
}

func maybeFatal(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestWgNet_Open_Fails(t *testing.T) {
	srv := wgnet.New(&wgnet.Config{})
	err := srv.Open()
	if err == nil {
		t.Error("expected error")
	}
	err = srv.Close()
	if err != nil {
		t.Error(err)
	}
}

func TestWgNet_PingServer(t *testing.T) {
	srv, cli := makeNets()
	defer cli.Close()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	latency, err := cli.Ping4(ctx, "10.131.132.1")
	maybeFatal(t, err)
	t.Log(latency)
}

func Benchmark_PingServer(b *testing.B) {
	srv, cli := makeNets()
	defer cli.Close()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(b.Context(), time.Minute)
	defer cancel()

	var totlatency time.Duration
	var numlatency time.Duration
	for b.Loop() {
		numlatency++
		latency, err := cli.Ping4(ctx, "10.131.132.1")
		totlatency += latency
		if err != nil {
			b.Error(err)
			break
		}
	}
	b.Log(totlatency / numlatency)
}

func TestWgNet_LookupHost(t *testing.T) {
	srv, cli := makeNets()
	defer cli.Close()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	ips, err := cli.LookupHost(ctx, "127.0.0.1") // will be a no-op
	maybeFatal(t, err)
	t.Log(ips)
}

func TestWgNet_Listen(t *testing.T) {
	srv, cli := makeNets()
	defer cli.Close()
	defer srv.Close()

	l, err := srv.Listen("tcp", "10.131.132.1:0")
	maybeFatal(t, err)
	defer l.Close()

	t.Log(l.Addr().String())

	want := make([]byte, 16)
	_, _ = rand.Read(want)

	conn, err := cli.Dial("tcp", l.Addr().String())
	maybeFatal(t, err)
	defer conn.Close()

	err = conn.SetDeadline(time.Now().Add(time.Second))
	maybeFatal(t, err)

	go func() {
		_, err := conn.Write(want)
		if err != nil {
			t.Error(err)
		}
	}()

	buf := make([]byte, len(want))
	accepted, err := l.Accept()
	if err == nil {
		defer accepted.Close()
		if _, err = accepted.Read(buf); err != nil {
			t.Error(err)
		}
	} else {
		t.Error(err)
	}

	if !bytes.Equal(want, buf) {
		t.Error(buf)
	}
}

func TestWgNet_LookupHost_ExternalServer(t *testing.T) {
	clientconfigfile := os.Getenv("WGNET_TEST_CLIENT_CONFIG")
	if clientconfigfile == "" {
		t.Skip("WGNET_TEST_CLIENT_CONFIG not set")
	}
	b, err := os.ReadFile(clientconfigfile)
	maybeFatal(t, err)
	cfg, err := wgnet.Parse(bytes.NewReader(b), nil)
	maybeFatal(t, err)
	cli := wgnet.New(cfg)
	err = cli.Open()
	maybeFatal(t, err)
	defer cli.Close()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*2)
	defer cancel()

	pingaddr := cfg.Addresses[0].Masked().Addr().Next()
	latency, err := cli.Ping4(ctx, pingaddr.String())
	if err != nil {
		t.Fatal("failed to ping", pingaddr.String(), err)
	}
	t.Log(pingaddr.String(), latency)

	now := time.Now()
	ips, err := cli.LookupHost(ctx, "cloudflare.com")
	maybeFatal(t, err)
	t.Log("cloudflare.com", ips, time.Since(now))
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
	wgnet.WaitForNoLoad(&lw, time.Millisecond, time.Millisecond*10, time.Millisecond*100)
	if !lw.closed.Load() {
		t.Fatal("expected device close after no-load period")
	}
}

func TestWaitForNoLoad_ClosesAtMaxTime(t *testing.T) {
	var lw loadwaiter
	lw.underLoad.Store(true)
	wgnet.WaitForNoLoad(&lw, time.Millisecond, time.Millisecond*10, time.Millisecond*20)
	if !lw.closed.Load() {
		t.Fatal("expected device close at max wait time")
	}
}
