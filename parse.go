package wgnet

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/linkdata/inifile"
)

var ErrInvalidInterfacePrivateKey = errors.New("invalid [Interface] PrivateKey")
var ErrInvalidPeerPublicKey = errors.New("invalid [Peer] PublicKey")
var ErrInvalidPeerEndpoint = errors.New("invalid [Peer] Endpoint")
var ErrMissingInterfaceAddress = errors.New("missing [Interface] Address")
var ErrKeyLengthNot32Bytes = errors.New("key length not 32 bytes")
var ErrInvalidInterfaceAddress = errors.New("invalid [Interface] Address")
var ErrInvalidInterfaceDNS = errors.New("invalid [Interface] DNS")
var ErrInvalidPeerAllowedIPs = errors.New("invalid [Peer] AllowedIPs")
var ErrInvalidPeerPresharedKey = errors.New("invalid [Peer] PresharedKey")
var ErrInvalidPeerPersistentKeepalive = errors.New("invalid [Peer] PersistentKeepalive")
var ErrInvalidInterfaceListenPort = errors.New("invalid [Interface] ListenPort")
var ErrInvalidPeerEndpointPort = errors.New("invalid [Peer] Endpoint port")
var ErrMissingPeerEndpointAddress = errors.New("missing [Peer] Endpoint address")

var endpointLookupNetIP = net.DefaultResolver.LookupNetIP

// Parse reads a WireGuard configuration file, validates it and returns a Config.
func Parse(r io.Reader, opts *Options) (cfg *Config, err error) {
	if opts == nil {
		opts = DefaultOptions
	}

	var inif inifile.File
	if inif, err = inifile.Parse(r, ','); err == nil {
		var cf Config
		if cf.PrivateKey, err = mustDecode(inif, "interface", "privatekey", ErrInvalidInterfacePrivateKey); err == nil {
			if cf.PublicKey, err = mustDecode(inif, "peer", "publickey", ErrInvalidPeerPublicKey); err == nil {
				for addr := range strings.SplitSeq(inif.GetDefault("interface", "address", ""), ",") {
					if addr != "" {
						var pf netip.Prefix
						if pf, err = mustPrefix(addr, ErrInvalidInterfaceAddress); err != nil {
							return
						}
						if opts.AllowIpv6 || pf.Addr().Is4() {
							cf.Addresses = append(cf.Addresses, pf)
						}
					}
				}
				if len(cf.Addresses) == 0 {
					return nil, ErrMissingInterfaceAddress
				}

				for addr := range strings.SplitSeq(inif.GetDefault("interface", "dns", opts.DNS), ",") {
					if addr != "" {
						var a netip.Addr
						if a, err = mustAddress(addr, ErrInvalidInterfaceDNS); err != nil {
							return
						}
						cf.DNS = append(cf.DNS, a)
					}
				}

				for addr := range strings.SplitSeq(inif.GetDefault("peer", "allowedips", opts.AllowedIPs), ",") {
					if addr != "" {
						var pf netip.Prefix
						if pf, err = mustPrefix(addr, ErrInvalidPeerAllowedIPs); err != nil {
							return
						}
						cf.AllowedIPs = append(cf.AllowedIPs, pf)
					}
				}

				if v, ok := inif.Get("peer", "presharedkey"); ok {
					if cf.PresharedKey, err = decodePresharedKey(v); err != nil {
						err = errors.Join(ErrInvalidPeerPresharedKey, err)
					}
				}

				if err == nil {
					if v, ok := inif.Get("peer", "persistentkeepalive"); ok {
						if cf.PersistentKeepalive, err = strconv.Atoi(v); err != nil || cf.PersistentKeepalive < 0 || cf.PersistentKeepalive > 0xFFFF {
							err = errors.Join(ErrInvalidPeerPersistentKeepalive, err)
						}
					}
				}

				if err == nil {
					if v, ok := inif.Get("interface", "listenport"); ok {
						if cf.ListenPort, err = strconv.Atoi(v); err != nil || cf.ListenPort < 1 || cf.ListenPort > 0xFFFF {
							err = errors.Join(ErrInvalidInterfaceListenPort, err)
						}
					}
				}

				if err == nil {
					if v, ok := inif.Get("peer", "endpoint"); ok {
						cf.Endpoint, err = mustEndpoint(v, ErrInvalidPeerEndpoint)
					}
				}

				if err == nil {
					cf.LogLevel = opts.LogLevel
					cfg = &cf
				}
			}
		}
	}

	return
}

func decodeHexKey(key string) (decoded []byte, err error) {
	if decoded, err = hex.DecodeString(key); err == nil {
		if len(decoded) != 32 {
			err = ErrKeyLengthNot32Bytes
		}
	}
	return
}

func decodeKey(key string) (decoded []byte, err error) {
	if decoded, err = base64.StdEncoding.DecodeString(key); err == nil {
		if len(decoded) != 32 {
			err = ErrKeyLengthNot32Bytes
		}
	}
	return
}

func decodePresharedKey(key string) (decoded []byte, err error) {
	if decoded, err = decodeHexKey(key); err != nil {
		decoded, err = decodeKey(key)
	}
	return
}

func mustAddress(addr string, fail error) (a netip.Addr, err error) {
	if a, err = netip.ParseAddr(strings.TrimSpace(addr)); err != nil {
		err = errors.Join(fail, err)
	}
	return
}

func mustPrefix(addr string, fail error) (pf netip.Prefix, err error) {
	addr = strings.TrimSpace(addr)
	if pf, err = netip.ParsePrefix(addr); err != nil {
		var a netip.Addr
		if a, err = netip.ParseAddr(addr); err != nil {
			err = errors.Join(fail, err)
		} else {
			pf = netip.PrefixFrom(a, a.BitLen())
		}
	}
	return
}

func mustEndpoint(v string, fail error) (endpoint netip.AddrPort, err error) {
	v = strings.TrimSpace(v)
	if endpoint, err = netip.ParseAddrPort(v); err != nil {
		if host, port, splitErr := net.SplitHostPort(v); splitErr == nil {
			if portNum, atoiErr := strconv.Atoi(port); atoiErr == nil {
				if portNum >= 0 && portNum <= 0xFFFF {
					var addrs []netip.Addr
					if addrs, err = endpointLookupNetIP(context.Background(), "ip", host); err == nil {
						err = ErrMissingPeerEndpointAddress
						for _, addr := range addrs {
							if addr.Is4() {
								endpoint = netip.AddrPortFrom(addr, uint16(portNum))
								err = nil
								break
							}
						}
						if err != nil {
							for _, addr := range addrs {
								if addr.IsValid() {
									endpoint = netip.AddrPortFrom(addr, uint16(portNum))
									err = nil
									break
								}
							}
						}
					}
				} else {
					err = ErrInvalidPeerEndpointPort
				}
			} else {
				err = atoiErr
			}
		} else {
			err = splitErr
		}
		if err != nil {
			err = errors.Join(fail, err)
		}
	}
	return
}

func mustGet(inif inifile.File, section, key string, fail error) (v string, err error) {
	var ok bool
	if v, ok = inif.Get(section, key); !ok {
		err = fail
	}
	return
}

func mustDecode(inif inifile.File, section, key string, fail error) (v []byte, err error) {
	var s string
	if s, err = mustGet(inif, section, key, fail); err == nil {
		if v, err = decodeKey(s); err != nil {
			err = errors.Join(fail, err)
		}
	}
	return
}
