[![build](https://github.com/linkdata/wgnet/actions/workflows/build.yml/badge.svg)](https://github.com/linkdata/wgnet/actions/workflows/build.yml)
[![coverage](https://github.com/linkdata/wgnet/blob/coverage/main/badge.svg)](https://htmlpreview.github.io/?https://github.com/linkdata/wgnet/blob/coverage/main/report.html)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/wgnet)](https://goreportcard.com/report/github.com/linkdata/wgnet)
[![Docs](https://godoc.org/github.com/linkdata/wgnet?status.svg)](https://godoc.org/github.com/linkdata/wgnet)

# wgnet

WireGuard ContextDialer for Go.

`wgnet.New` requires a non-nil `*wgnet.Config`. Calling `Open` on an instance
created with `nil` config will panic.

`(*WgNet).Close` is intentionally asynchronous. It detaches the netstack
immediately and returns before the underlying WireGuard device is guaranteed
to release OS resources (for example the UDP listen port). This means an
immediate `Open` on another instance using the same listen port can fail
transiently with `address already in use`.

```go
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/linkdata/wgnet"
)

var ServerConfig = `[Interface]
PrivateKey = GInruesHOogIjjFsKCorYEAENAfYfPL/yH8ObFgyFUs=
ListenPort = 51820
Address = 10.131.132.1/24

[Peer]
PublicKey = kTUQWHx4Y3ZYMZQPnRarzlx0qnen3plDoI0z7s45in4=
AllowedIPs = 10.131.132.2/32
`

var ClientConfig = `[Interface]
PrivateKey = AEnvL9tVr+7JF0sMVjjzPjIxrrc/hoVJ5B82WWpVamI=
Address = 10.131.132.2/24
DNS = 1.1.1.1

[Peer]
PublicKey = Wh3yY7/fE3fyHJ8TOwLJ//CIRbgrlVl4bLQ+npNBSRU=
Endpoint = 127.0.0.1:51820
AllowedIPs = 0.0.0.0/0, ::/0
`

func main() {
	var err error
	var srv, cli *wgnet.WgNet
	var srvCfg, cliCfg *wgnet.Config
	if srvCfg, err = wgnet.Parse(strings.NewReader(ServerConfig), nil); err == nil {
		srv = wgnet.New(srvCfg)
		if cliCfg, err = wgnet.Parse(strings.NewReader(ClientConfig), nil); err == nil {
			cli = wgnet.New(cliCfg)
			if err = srv.Open(); err == nil {
				defer srv.Close()
				if err = cli.Open(); err == nil {
					defer cli.Close()
					ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
					defer cancel()
					var latency time.Duration
					if latency, err = cli.Ping4(ctx, "10.131.132.1"); err == nil {
						fmt.Println(latency > 0, latency < time.Minute)
					}
				}
			}
		}
	}
	if err != nil {
		panic(err)
	}
	// Output:
	// true true
}
```

## Setting up a Wireguard exit node on Debian/Ubuntu

### Install software and generate keys for exit node

```sh
apt install firewalld wireguard
wg genkey | tee /etc/wireguard/wg_private.key | wg pubkey | tee /etc/wireguard/wg_public.key
```

### Create configuration file

`nano /etc/wireguard/wg0.conf`

```conf
[Interface]
Address = 10.99.0.1/24
SaveConfig = true
ListenPort = 51820
PrivateKey = <wg_private.key>
```

### Configure firewall

```sh
firewall-cmd --permanent --zone public --add-interface=<external_if>
firewall-cmd --permanent --zone public --add-masquerade
firewall-cmd --permanent --zone public --add-port=51820/udp
firewall-cmd --reload
```

### Start and set wireguard to start on boot

```sh
wg-quick up wg0
systemctl enable wg-quick@wg0
```

### Add the clients that will use the exit node

```
wg set wg0 peer <peer1_public_key> allowed-ips 10.99.0.2
wg set wg0 peer <peer2_public_key> allowed-ips 10.99.0.3

wg-quick save wg0
```
