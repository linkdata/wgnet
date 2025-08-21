[![build](https://github.com/linkdata/wgnet/actions/workflows/build.yml/badge.svg)](https://github.com/linkdata/wgnet/actions/workflows/build.yml)
[![coverage](https://github.com/linkdata/wgnet/blob/coverage/main/badge.svg)](https://htmlpreview.github.io/?https://github.com/linkdata/wgnet/blob/coverage/main/report.html)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/wgnet)](https://goreportcard.com/report/github.com/linkdata/wgnet)
[![Docs](https://godoc.org/github.com/linkdata/wgnet?status.svg)](https://godoc.org/github.com/linkdata/wgnet)

# wgnet

WireGuard ContextDialer for Go.

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
