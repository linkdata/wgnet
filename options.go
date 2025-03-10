package wgnet

// device.LogLevel from wireguard-go/device/logger.go
const (
	LogLevelSilent = iota
	LogLevelError
	LogLevelVerbose
)

var DefaultOptions = &Options{
	AllowedIPs: "0.0.0.0/0",
}

type Options struct {
	AllowedIPs string
	DNS        string
	LogLevel   int
	AllowIpv6  bool
}
