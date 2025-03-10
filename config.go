package wgnet

import (
	"encoding/base64"
	"fmt"
	"net/netip"
	"strings"
)

type Config struct {
	Addresses           []netip.Prefix
	PrivateKey          []byte
	PublicKey           []byte
	PresharedKey        []byte
	Endpoint            netip.AddrPort
	AllowedIPs          []netip.Prefix
	DNS                 []netip.Addr
	ListenPort          int
	LogLevel            int
	PersistentKeepalive int
}

func (cfg *Config) UapiConf() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "private_key=%x\n", cfg.PrivateKey)
	if cfg.ListenPort > 0 {
		fmt.Fprintf(&buf, "listen_port=%d\n", cfg.ListenPort)
	}
	fmt.Fprintf(&buf, "public_key=%x\n", cfg.PublicKey)
	if cfg.Endpoint.IsValid() {
		fmt.Fprintf(&buf, "endpoint=%s\n", cfg.Endpoint.String())
	}
	if len(cfg.PresharedKey) > 0 {
		fmt.Fprintf(&buf, "preshared_key=%x\n", cfg.PresharedKey)
	}
	for _, pf := range cfg.AllowedIPs {
		fmt.Fprintf(&buf, "allowed_ip=%s\n", pf.String())
	}
	if cfg.PersistentKeepalive > 0 {
		fmt.Fprintf(&buf, "persistent_keepalive_interval=%d\n", cfg.PersistentKeepalive)
	}
	return buf.String()
}

func (cfg *Config) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "[Interface]\nPrivateKey = %s", base64.StdEncoding.EncodeToString(cfg.PrivateKey))
	if cfg.ListenPort > 0 {
		fmt.Fprintf(&buf, "\nListenPort = %d", cfg.ListenPort)
	}
	if len(cfg.Addresses) > 0 {
		buf.WriteString("\nAddress = ")
		for n, pf := range cfg.Addresses {
			if n > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(pf.String())
		}
	}
	if len(cfg.DNS) > 0 {
		buf.WriteString("\nDNS = ")
		for n, addr := range cfg.DNS {
			if n > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(addr.String())
		}
	}
	fmt.Fprintf(&buf, "\n\n[Peer]\nPublicKey = %s",
		base64.StdEncoding.EncodeToString(cfg.PublicKey),
	)
	if cfg.Endpoint.IsValid() {
		fmt.Fprintf(&buf, "\nEndpoint = %s", cfg.Endpoint.String())
	}
	if len(cfg.PresharedKey) > 0 {
		fmt.Fprintf(&buf, "\nPresharedKey = %x", cfg.PresharedKey)
	}
	if cfg.PersistentKeepalive > 0 {
		fmt.Fprintf(&buf, "\nPersistentKeepalive = %v", cfg.PersistentKeepalive)
	}
	if len(cfg.AllowedIPs) > 0 {
		buf.WriteString("\nAllowedIPs = ")
		for n, pf := range cfg.AllowedIPs {
			if n > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(pf.String())
		}
	}
	buf.WriteByte('\n')
	return buf.String()
}
