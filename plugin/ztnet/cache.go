package ztnet

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// RecordCache is a concurrency-safe in-memory DNS record store.
type RecordCache struct {
	mu      sync.RWMutex
	records map[string][]net.IP
}

// Replace atomically swaps the entire record set.
func (rc *RecordCache) Replace(newRecords map[string][]net.IP) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if newRecords == nil {
		rc.records = map[string][]net.IP{}
		return
	}
	rc.records = newRecords
}

// Lookup returns IPs for a FQDN, or (nil, false) if not found.
func (rc *RecordCache) Lookup(fqdn string) ([]net.IP, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	ips, ok := rc.records[strings.ToLower(fqdn)]
	if !ok {
		return nil, false
	}
	out := make([]net.IP, len(ips))
	copy(out, ips)
	return out, true
}

func (rc *RecordCache) refreshLoop(ctx context.Context, c *Client, cfg *Config) {
	if err := rc.refresh(ctx, c, cfg); err != nil {
		log.Errorf("refresh failed: %v", err)
	}
	ticker := time.NewTicker(cfg.RefreshTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := rc.refresh(ctx, c, cfg); err != nil {
				log.Errorf("refresh failed: %v", err)
			}
		}
	}
}

func (rc *RecordCache) refresh(ctx context.Context, c *Client, cfg *Config) error {
	records := make(map[string][]net.IP)
	for _, nz := range cfg.Networks {
		netInfo, err := c.GetNetworkInfo(ctx, nz.NetworkID)
		if err != nil {
			return fmt.Errorf("ztnet: cache: %w", err)
		}
		members, err := c.GetMembers(ctx, nz.NetworkID)
		if err != nil {
			return fmt.Errorf("ztnet: cache: %w", err)
		}
		for _, member := range members {
			names := []string{member.Name + "." + nz.Zone, member.ID + "." + nz.Zone}
			for _, name := range names {
				fqdn := strings.ToLower(strings.TrimSuffix(name, ".") + ".")
				for _, ip := range member.IPs {
					records[fqdn] = append(records[fqdn], ip)
				}
				if netInfo.RFC4193 {
					ip, err := RFC4193(nz.NetworkID, member.ID)
					if err != nil {
						return fmt.Errorf("ztnet: cache: %w", err)
					}
					records[fqdn] = append(records[fqdn], ip)
				}
				if netInfo.SixPlane {
					ip, err := SixPlane(nz.NetworkID, member.ID)
					if err != nil {
						return fmt.Errorf("ztnet: cache: %w", err)
					}
					records[fqdn] = append(records[fqdn], ip)
				}
			}
		}
	}
	rc.Replace(records)
	return nil
}
