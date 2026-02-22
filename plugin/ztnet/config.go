package ztnet

import "time"

const (
	// DefaultRefreshTTL is the default API polling interval.
	DefaultRefreshTTL = 60 * time.Second
	// DefaultDNSTTL is the default TTL used for DNS records served by ztnet.
	DefaultDNSTTL = 30 * time.Second
	// DefaultHTTPTimeout is the default timeout for API HTTP calls.
	DefaultHTTPTimeout = 10 * time.Second
)

// Config holds all ztnet plugin configuration.
type Config struct {
	APIAddress string
	APIToken   string
	Networks   []NetworkZone
	RefreshTTL time.Duration
	DNSTTL     time.Duration
}

// NetworkZone pairs a DNS zone with a ZeroTier network ID.
type NetworkZone struct {
	Zone      string
	NetworkID string
}
