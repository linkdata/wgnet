package wgnet_test

import (
	"encoding/base64"
	"errors"
	"net/netip"
	"reflect"
	"strings"
	"testing"

	"github.com/linkdata/wgnet"
)

func decodeKey(key string) (decoded []byte) {
	decoded, _ = base64.StdEncoding.DecodeString(key)
	return
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		opts    *wgnet.Options
		wantCfg *wgnet.Config
		wantErr error
	}{
		{
			name:    "empty",
			text:    "",
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrInvalidInterfacePrivateKey,
		},
		{
			name:    "ErrKeyLengthNot32Bytes",
			text:    "[interface]\nprivatekey = Zm9vYmFy",
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrKeyLengthNot32Bytes,
		},
		{
			name:    "ErrInvalidPeerPublicKey",
			text:    "[interface]\nprivatekey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=\n[peer]\npublickey = a",
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrInvalidPeerPublicKey,
		},
		{
			name: "ErrMissingInterfaceAddress",
			text: `
				[interface]
				privatekey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				address = ::1
				[peer]
				publickey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				endpoint = 10.0.0.1:1
				`,
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrMissingInterfaceAddress,
		},
		{
			name: "ErrInvalidInterfaceAddress",
			text: `
				[interface]
				privatekey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				address = 123
				[peer]
				publickey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				endpoint = 10.0.0.1:1
				`,
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrInvalidInterfaceAddress,
		},
		{
			name: "ErrInvalidPeerEndpoint",
			text: `
				[interface]
				privatekey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				address = 192.168.1.0/24
				[peer]
				publickey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				endpoint = 123
				`,
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrInvalidPeerEndpoint,
		},
		{
			name: "ErrInvalidInterfaceDNS",
			text: `
				[interface]
				privatekey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				address = 192.168.1.0/24
				dns = 123
				[peer]
				publickey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				endpoint = 10.0.0.1:1
				`,
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrInvalidInterfaceDNS,
		},
		{
			name: "ErrInvalidPeerAllowedIPs",
			text: `
				[interface]
				privatekey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				address = 192.168.1.0/24
				dns = 1.1.1.1
				[peer]
				publickey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				endpoint = 10.0.0.1:1
				allowedips = 123
				`,
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrInvalidPeerAllowedIPs,
		},
		{
			name: "ErrInvalidPeerPresharedKey",
			text: `
				[interface]
				privatekey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				address = 192.168.1.0/24
				dns = 1.1.1.1
				[peer]
				publickey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				endpoint = 10.0.0.1:1
				presharedkey = meh
				`,
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrInvalidPeerPresharedKey,
		},
		{
			name: "ErrInvalidPeerPersistentKeepalive",
			text: `
				[interface]
				privatekey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				address = 192.168.1.0/24
				dns = 1.1.1.1
				[peer]
				publickey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				endpoint = 10.0.0.1:1
				persistentkeepalive = meh
				`,
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrInvalidPeerPersistentKeepalive,
		},
		{
			name: "ErrInvalidInterfaceListenPort",
			text: `
				[interface]
				privatekey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				address = 192.168.1.0/24
				dns = 1.1.1.1
				listenport = 123456
				[peer]
				publickey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				endpoint = 10.0.0.1:1
				persistentkeepalive = 10
				`,
			opts:    nil,
			wantCfg: nil,
			wantErr: wgnet.ErrInvalidInterfaceListenPort,
		},
		{
			name: "everything",
			text: `
				[interface]
				privatekey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				address = 192.168.1.0/24, fe80::/10
				listenport = 51820
				[peer]
				publickey = WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E=
				endpoint = 10.0.0.1:1
				presharedkey = 1234abcd
				persistentkeepalive = 10
				`,
			opts: &wgnet.Options{
				AllowedIPs: "192.168.2.0/24",
				DNS:        "1.1.1.1, 8.8.8.8",
				LogLevel:   1,
				AllowIpv6:  true,
			},
			wantCfg: &wgnet.Config{
				Addresses: []netip.Prefix{
					netip.MustParsePrefix("192.168.1.0/24"),
					netip.MustParsePrefix("fe80::/10"),
				},
				PrivateKey:   decodeKey("WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E="),
				PublicKey:    decodeKey("WDE5QVQyVWxQRWZBUEdldkxMWHRURng5MlVPTlk4M1E="),
				PresharedKey: []byte{0x12, 0x34, 0xAB, 0xCD},
				Endpoint:     netip.MustParseAddrPort("10.0.0.1:1"),
				AllowedIPs: []netip.Prefix{
					netip.MustParsePrefix("192.168.2.0/24"),
				},
				DNS: []netip.Addr{
					netip.MustParseAddr("1.1.1.1"),
					netip.MustParseAddr("8.8.8.8"),
				},
				ListenPort:          51820,
				LogLevel:            1,
				PersistentKeepalive: 10,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCfg, err := wgnet.Parse(strings.NewReader(tt.text), tt.opts)
			if err == nil {
				if tt.wantErr != nil {
					t.Errorf("Parse() wanted error = %v", tt.wantErr)
				}
				if !reflect.DeepEqual(gotCfg, tt.wantCfg) {
					t.Errorf("Parse() = %v, want %v", gotCfg, tt.wantCfg)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Parse() error = %v, wanted %v", err, tt.wantErr)
				}
			}
		})
	}
}
