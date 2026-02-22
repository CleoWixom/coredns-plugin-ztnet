package ztnet

import (
	"os"
	"testing"
	"time"

	"github.com/coredns/caddy"
)

func TestParseConfigValid(t *testing.T) {
	t.Setenv("ZTNET_API_TOKEN", "")
	c := caddy.NewTestController("dns", `ztnet {
		endpoint http://localhost:3000
		token abc
		network ztnet.network:8056c2e21c000001
		network home.lan:abcdef01234567aa
		refresh 90s
		dns_ttl 45s
		fallthrough
	}`)
	cfg, fall, err := parseConfig(c)
	if err != nil {
		t.Fatalf("parseConfig error: %v", err)
	}
	if cfg.APIAddress != "http://localhost:3000" || cfg.APIToken != "abc" {
		t.Fatalf("unexpected cfg %#v", cfg)
	}
	if len(cfg.Networks) != 2 || cfg.Networks[0].Zone != "ztnet.network." {
		t.Fatalf("unexpected networks %#v", cfg.Networks)
	}
	if cfg.RefreshTTL != 90*time.Second || cfg.DNSTTL != 45*time.Second {
		t.Fatalf("unexpected durations %#v", cfg)
	}
	if !fall.Through("anything.") {
		t.Fatalf("expected fallthrough enabled")
	}
}

func TestParseConfigDefaults(t *testing.T) {
	t.Setenv("ZTNET_API_TOKEN", "env-token")
	c := caddy.NewTestController("dns", `ztnet {
		endpoint http://localhost:3000
		network home.lan:abcdef01234567aa
	}`)
	cfg, _, err := parseConfig(c)
	if err != nil {
		t.Fatalf("parseConfig error: %v", err)
	}
	if cfg.APIToken != "env-token" {
		t.Fatalf("expected token from env, got %q", cfg.APIToken)
	}
	if cfg.RefreshTTL != DefaultRefreshTTL || cfg.DNSTTL != DefaultDNSTTL {
		t.Fatalf("expected defaults got %#v", cfg)
	}
}

func TestParseConfigErrors(t *testing.T) {
	_ = os.Unsetenv("ZTNET_API_TOKEN")
	cases := []string{
		`ztnet { token t network home.lan:abcdef01234567aa }`,
		`ztnet { endpoint http://localhost:3000 network home.lan:abcdef01234567aa }`,
		`ztnet { endpoint http://localhost:3000 token t }`,
		`ztnet { endpoint http://localhost:3000 token t network bad_zone:abcdef01234567aa }`,
		`ztnet { endpoint http://localhost:3000 token t network home.lan:abcdef01234567aa refresh x }`,
		`ztnet { endpoint http://localhost:3000 token t network home.lan:abcdef01234567aa dns_ttl x }`,
	}
	for _, input := range cases {
		c := caddy.NewTestController("dns", input)
		if _, _, err := parseConfig(c); err == nil {
			t.Fatalf("expected error for %s", input)
		}
	}
}
