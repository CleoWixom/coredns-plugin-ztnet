package ztnet

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var (
	log       = clog.NewWithPlugin("ztnet")
	zoneRegex = regexp.MustCompile(`^([a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)
)

// init registers the plugin with the CoreDNS plugin system.
func init() { plugin.Register("ztnet", setup) }

func setup(c *caddy.Controller) error {
	cfg, ft, err := parseConfig(c)
	if err != nil {
		return plugin.Error("ztnet", err)
	}

	z := &ZTNet{Config: cfg, Cache: &RecordCache{}, Client: NewClient(cfg.APIAddress, cfg.APIToken), Fall: ft}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		z.Next = next
		return z
	})

	ctx, cancel := context.WithCancel(context.Background())
	c.OnStartup(func() error {
		go z.Cache.refreshLoop(ctx, z.Client, z.Config)
		return nil
	})
	c.OnShutdown(func() error {
		cancel()
		return nil
	})

	return nil
}

func parseConfig(c *caddy.Controller) (*Config, fall.F, error) {
	cfg := &Config{RefreshTTL: DefaultRefreshTTL, DNSTTL: DefaultDNSTTL}
	ft := fall.Zero
	networkCount := 0

	for c.Next() {
		for c.NextBlock() {
			switch c.Val() {
			case "endpoint":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, fall.Zero, c.Errf("endpoint requires exactly one value")
				}
				cfg.APIAddress = args[0]
			case "token":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, fall.Zero, c.Errf("token requires exactly one value")
				}
				cfg.APIToken = args[0]
			case "network":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, fall.Zero, c.Errf("network requires zone:networkID")
				}
				parts := strings.Split(args[0], ":")
				if len(parts) != 2 {
					return nil, fall.Zero, c.Errf("network must be zone:networkID")
				}
				zone := strings.ToLower(parts[0])
				if !zoneRegex.MatchString(zone) {
					return nil, fall.Zero, c.Errf("invalid zone name %q", zone)
				}
				zone = strings.TrimSuffix(zone, ".") + "."
				cfg.Networks = append(cfg.Networks, NetworkZone{Zone: zone, NetworkID: strings.ToLower(parts[1])})
				networkCount++
			case "refresh":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, fall.Zero, c.Errf("refresh requires duration")
				}
				d, err := time.ParseDuration(args[0])
				if err != nil {
					return nil, fall.Zero, c.Errf("invalid refresh duration %q", args[0])
				}
				cfg.RefreshTTL = d
			case "dns_ttl":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, fall.Zero, c.Errf("dns_ttl requires duration")
				}
				d, err := time.ParseDuration(args[0])
				if err != nil {
					return nil, fall.Zero, c.Errf("invalid dns_ttl duration %q", args[0])
				}
				cfg.DNSTTL = d
			case "fallthrough":
				ft.SetZonesFromArgs(c.RemainingArgs())
			default:
				return nil, fall.Zero, c.Errf("unknown property %q", c.Val())
			}
		}
	}

	if cfg.APIAddress == "" {
		return nil, fall.Zero, fmt.Errorf("endpoint is required")
	}
	if cfg.APIToken == "" {
		cfg.APIToken = os.Getenv("ZTNET_API_TOKEN")
	}
	if cfg.APIToken == "" {
		return nil, fall.Zero, fmt.Errorf("token is required (or set ZTNET_API_TOKEN)")
	}
	if networkCount == 0 {
		return nil, fall.Zero, fmt.Errorf("at least one network must be configured")
	}

	return cfg, ft, nil
}
