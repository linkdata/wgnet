module github.com/linkdata/wgnet

go 1.24

require (
	github.com/linkdata/deadlock v0.5.5
	github.com/linkdata/inifile v1.0.1
	golang.org/x/net v0.39.0
	golang.zx2c4.com/wireguard v0.0.0-20250521234502-f333402bd9cb
)

require (
	github.com/google/btree v1.1.3 // indirect
	github.com/petermattis/goid v0.0.0-20250813065127-a731cc31b4fe // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	gvisor.dev/gvisor v0.0.0-20250503011706-39ed1f5ac29c // indirect
)

// The following version combinations are known to work. Be careful updating them.
//
// Using zx2c4:
//	golang.zx2c4.com/wireguard v0.0.0-20250521234502-f333402bd9cb
//	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
//	gvisor.dev/gvisor v0.0.0-20250503011706-39ed1f5ac29c // indirect
//
// Using tailscale:
//  github.com/tailscale/wireguard-go v0.0.0-20250107165329-0b8b35511f19
//	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
//  gvisor.dev/gvisor v0.0.0-20230927004350-cbd86285d259 // indirect
