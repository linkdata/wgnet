package wgnet_test

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

func Example() {
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
