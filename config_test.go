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
PresharedKey = AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=
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

func TestConfig_String_Base64PresharedKey(t *testing.T) {
	const base64PSK = "WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E="
	const input = `[Interface]
PrivateKey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
Address = 192.168.1.0/24
DNS = 1.1.1.1

[Peer]
PublicKey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
Endpoint = 10.0.0.1:1
PresharedKey = ` + base64PSK + `
AllowedIPs = 0.0.0.0/0
`
	cfg, err := wgnet.Parse(strings.NewReader(input), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.String()
	if !strings.Contains(got, "PresharedKey = "+base64PSK) {
		t.Errorf("String() should encode PresharedKey as base64\ngot: %s", got)
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
preshared_key=000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f
allowed_ip=192.168.1.0/24
allowed_ip=10.0.0.0/8
persistent_keepalive_interval=10
`
	if got != want {
		t.Errorf("mismatch\n got: %s\nwant: %s\n", got, want)
	}
}
