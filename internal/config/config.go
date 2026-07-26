package config

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddress string
	LeaseFile     string
	Interface     string
	NeighborFile  string
	IPCommand     string
	NetworkName   string
	PoolStart     netip.Addr
	PoolEnd       netip.Addr
	PollInterval  time.Duration
}

func Load() (Config, error) {
	poolStart, err := netip.ParseAddr(env("POOL_START", "192.168.1.100"))
	if err != nil || !poolStart.Is4() {
		return Config{}, fmt.Errorf("POOL_START must be a valid IPv4 address")
	}

	poolEnd, err := netip.ParseAddr(env("POOL_END", "192.168.1.249"))
	if err != nil || !poolEnd.Is4() {
		return Config{}, fmt.Errorf("POOL_END must be a valid IPv4 address")
	}
	if poolStart.Compare(poolEnd) > 0 {
		return Config{}, fmt.Errorf("POOL_START must not be greater than POOL_END")
	}

	pollSeconds, err := strconv.Atoi(env("POLL_INTERVAL_SECONDS", "2"))
	if err != nil || pollSeconds < 1 || pollSeconds > 60 {
		return Config{}, fmt.Errorf("POLL_INTERVAL_SECONDS must be between 1 and 60")
	}

	listenAddress := env("LISTEN_ADDRESS", "0.0.0.0:8080")
	if _, _, err := net.SplitHostPort(listenAddress); err != nil {
		return Config{}, fmt.Errorf("LISTEN_ADDRESS: %w", err)
	}

	return Config{
		ListenAddress: listenAddress,
		LeaseFile:     env("LEASE_FILE", "/var/lib/misc/dnsmasq.leases"),
		Interface:     env("NETWORK_INTERFACE", "eth0"),
		NeighborFile:  os.Getenv("NEIGHBOR_FILE"),
		IPCommand:     env("IP_COMMAND", "ip"),
		NetworkName:   env("NETWORK_NAME", "Local network"),
		PoolStart:     poolStart,
		PoolEnd:       poolEnd,
		PollInterval:  time.Duration(pollSeconds) * time.Second,
	}, nil
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
