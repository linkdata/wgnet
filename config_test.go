package wgnet_test

import (
	"strings"
	"testing"

	"github.com/linkdata/wgnet"
)

const text = `[Interface]
PrivateKey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
ListenPort = 6789
Address = 192.168.1.0/24,10.0.0.0/8
DNS = 1.1.1.1,8.8.8.8,9.9.9.9

[Peer]
PublicKey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
Endpoint = 10.0.0.1:1
PresharedKey = 1234abcd
PersistentKeepalive = 10
AllowedIPs = 192.168.1.0/24,10.0.0.0/8
`

func TestConfig_String(t *testing.T) {
	cfg, err := wgnet.Parse(strings.NewReader(text), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.String()
	if got != text {
		t.Errorf("mismatch\n got: %s\nwant: %s\n", got, text)
	}
}

func TestConfig_UapiConf(t *testing.T) {
	cfg, err := wgnet.Parse(strings.NewReader(text), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.UapiConf()
	want := `private_key=583139415432556c50456641504765764c4c58745446783932554f4e59383351
listen_port=6789
public_key=583139415432556c50456641504765764c4c58745446783932554f4e59383351
endpoint=10.0.0.1:1
preshared_key=1234abcd
allowed_ip=192.168.1.0/24
allowed_ip=10.0.0.0/8
persistent_keepalive_interval=10
`
	if got != want {
		t.Errorf("mismatch\n got: %s\nwant: %s\n", got, want)
	}
}
