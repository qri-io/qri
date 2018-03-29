package config

// RPC configures a Remote Procedure Call (RPC) listener
type RPC struct {
	Enabled bool
	Port    string
}

// DefaultRPCPort is local the port RPC serves on by default
var DefaultRPCPort = "2505"

// Default creates a new default RPC configuration
func (RPC) Default() *RPC {
	return &RPC{
		Enabled: true,
		Port:    DefaultRPCPort,
	}
}
