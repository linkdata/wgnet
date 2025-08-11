package wgnet

import (
	"bytes"
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"net/netip"
	"strconv"
	"time"

	"github.com/linkdata/deadlock"
	"github.com/tailscale/wireguard-go/conn"
	"github.com/tailscale/wireguard-go/device"
	"github.com/tailscale/wireguard-go/tun"
	"github.com/tailscale/wireguard-go/tun/netstack"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type WgNet struct {
	cfg *Config // read-only
	tun tun.Device
	mu  deadlock.Mutex // protects following
	dev *device.Device
	ns  *netstack.Net
}

var (
	ErrUnsupportedNetwork = errors.New("unsupported network")
	ErrInvalidPingReply   = errors.New("invalid ping reply")
)

func New(cfg *Config) *WgNet {
	return &WgNet{cfg: cfg}
}

func (wgnet *WgNet) getnet() (ns *netstack.Net, err error) {
	err = net.ErrClosed
	if wgnet != nil {
		wgnet.mu.Lock()
		if ns = wgnet.ns; ns != nil {
			err = nil
		}
		wgnet.mu.Unlock()
	}
	return
}

func (wgnet *WgNet) DialContext(ctx context.Context, network, address string) (conn net.Conn, err error) {
	var ns *netstack.Net
	if ns, err = wgnet.getnet(); err == nil {
		conn, err = ns.DialContext(ctx, network, address)
	}
	return
}

func (wgnet *WgNet) Dial(network string, address string) (net.Conn, error) {
	return wgnet.DialContext(context.Background(), network, address)
}

// LookupHost implements net.DefaultResolver.LookupHost
func (wgnet *WgNet) LookupHost(ctx context.Context, host string) (addrs []string, err error) {
	var ns *netstack.Net
	if ns, err = wgnet.getnet(); err == nil {
		addrs, err = ns.LookupContextHost(ctx, host)
	}
	return
}

func (wgnet *WgNet) Ping4(ctx context.Context, address string) (latency time.Duration, err error) {
	var ns *netstack.Net
	if ns, err = wgnet.getnet(); err == nil {
		var socket net.Conn
		if socket, err = ns.DialContext(ctx, "ping4", address); err == nil {
			requestPing := icmp.Echo{
				Seq:  rand.IntN(1 << 16), // #nosec G404
				Data: strconv.AppendInt([]byte("wgnet"), int64(rand.IntN(1<<32) /*#nosec G404*/), 16),
			}
			icmpBytes, _ := (&icmp.Message{Type: ipv4.ICMPTypeEcho, Code: 0, Body: &requestPing}).Marshal(nil)
			start := time.Now()
			dl := start.Add(time.Second * 10)
			if ctxdl, ok := ctx.Deadline(); ok {
				dl = ctxdl
			}
			if err = socket.SetDeadline(dl); err == nil {
				if _, err = socket.Write(icmpBytes); err == nil {
					var n int
					if n, err = socket.Read(icmpBytes[:]); err == nil {
						var replyPacket *icmp.Message
						if replyPacket, err = icmp.ParseMessage(1, icmpBytes[:n]); err == nil {
							err = ErrInvalidPingReply
							if replyPing, ok := replyPacket.Body.(*icmp.Echo); ok {
								if replyPing.Seq == requestPing.Seq && bytes.Equal(replyPing.Data, requestPing.Data) {
									latency = time.Since(start)
									err = nil
								}
							}
						}
					}
				}
			}
		}
	}
	return
}

func (wgnet *WgNet) Open() (err error) {
	_ = wgnet.Close()
	wgnet.mu.Lock()
	defer wgnet.mu.Unlock()
	var addrs []netip.Addr
	for _, pf := range wgnet.cfg.Addresses {
		addrs = append(addrs, pf.Addr())
	}
	if wgnet.tun, wgnet.ns, err = netstack.CreateNetTUN(addrs, wgnet.cfg.DNS, 1420); err == nil {
		wgnet.dev = device.NewDevice(wgnet.tun, conn.NewDefaultBind(), device.NewLogger(wgnet.cfg.LogLevel, "wgnet"))
		if err = wgnet.dev.IpcSet(wgnet.cfg.UapiConf()); err == nil {
			if err = wgnet.dev.Up(); err == nil {
				return
			}
		}
	}
	wgnet.ns = nil
	if dev := wgnet.dev; dev != nil {
		wgnet.dev = nil
		_ = wgnet.close(dev)
	}
	return
}

func (wgnet *WgNet) closing() (dev *device.Device) {
	wgnet.mu.Lock()
	if wgnet.ns != nil {
		dev = wgnet.dev
		wgnet.ns = nil
		wgnet.dev = nil
	}
	wgnet.mu.Unlock()
	return
}

type deviceLoad interface {
	IsUnderLoad() bool
	Close()
}

func WaitForNoLoad(dev deviceLoad, sleeptime, closetime, maxtime time.Duration) {
	var waited time.Duration
	var noload time.Duration
	for waited < maxtime {
		time.Sleep(sleeptime)
		waited += sleeptime
		noload += sleeptime
		if dev.IsUnderLoad() {
			noload = 0
		}
		if noload >= closetime {
			dev.Close()
			return
		}
	}
}

func (wgnet *WgNet) close(dev *device.Device) (err error) {
	wgnet.tun = nil // dev.Close() will close the TUN as well
	dev.RemoveAllPeers()
	go WaitForNoLoad(dev, time.Millisecond*100, time.Second*10, time.Second*60)
	return
}

func (wgnet *WgNet) Close() (err error) {
	if wgnet != nil {
		if dev := wgnet.closing(); dev != nil {
			err = wgnet.close(dev)
		}
	}
	return
}

func (wgnet *WgNet) Listen(network string, address string) (l net.Listener, err error) {
	var addrport netip.AddrPort
	if addrport, err = netip.ParseAddrPort(address); err == nil {
		var ns *netstack.Net
		if ns, err = wgnet.getnet(); err == nil {
			err = ErrUnsupportedNetwork
			switch network {
			case "tcp", "tcp4", "tcp6":
				l, err = ns.ListenTCPAddrPort(addrport)
			}
		}
	}
	return
}
