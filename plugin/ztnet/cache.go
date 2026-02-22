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
	mu   sync.RWMutex
	ipv4 map[string][]net.IP
	ipv6 map[string][]net.IP
}

// Replace atomically swaps the entire record set.
func (rc *RecordCache) Replace(newIPv4, newIPv6 map[string][]net.IP) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if newIPv4 == nil {
		rc.ipv4 = map[string][]net.IP{}
	} else {
		rc.ipv4 = newIPv4
	}
	if newIPv6 == nil {
		rc.ipv6 = map[string][]net.IP{}
	} else {
		rc.ipv6 = newIPv6
	}
}

// LookupA returns IPv4 addresses for a FQDN, or (nil, false) if not found.
func (rc *RecordCache) LookupA(fqdn string) ([]net.IP, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	ips, ok := rc.ipv4[strings.ToLower(fqdn)]
	if !ok {
		return nil, false
	}
	out := make([]net.IP, len(ips))
	copy(out, ips)
	return out, true
}

// LookupAAAA returns IPv6 addresses for a FQDN, or (nil, false) if not found.
func (rc *RecordCache) LookupAAAA(fqdn string) ([]net.IP, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	ips, ok := rc.ipv6[strings.ToLower(fqdn)]
	if !ok {
		return nil, false
	}
	out := make([]net.IP, len(ips))
	copy(out, ips)
	return out, true
}

func (rc *RecordCache) refreshLoop(ctx context.Context, c *Client, cfg *Config) {
	rc.refresh(ctx, c, cfg)
	ticker := time.NewTicker(cfg.RefreshTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rc.refresh(ctx, c, cfg)
		}
	}
}

func (rc *RecordCache) refresh(ctx context.Context, c *Client, cfg *Config) {
	newIPv4 := make(map[string][]net.IP)
	newIPv6 := make(map[string][]net.IP)

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	for _, nz := range cfg.Networks {
		nz := nz
		wg.Add(1)
		go func() {
			defer wg.Done()
			v4, v6, err := buildNetworkRecords(ctx, c, nz)
			if err != nil {
				log.Errorf("ztnet: cache: refresh %s: %v", nz.NetworkID, err)
				return
			}
			mu.Lock()
			mergeRecords(newIPv4, v4)
			mergeRecords(newIPv6, v6)
			mu.Unlock()
		}()
	}

	wg.Wait()
	rc.Replace(newIPv4, newIPv6)
}

func buildNetworkRecords(ctx context.Context, c *Client, nz NetworkZone) (map[string][]net.IP, map[string][]net.IP, error) {
	netInfo, err := c.GetNetworkInfo(ctx, nz.NetworkID)
	if err != nil {
		return nil, nil, fmt.Errorf("ztnet: cache: %w", err)
	}

	members, err := c.GetMembers(ctx, nz.NetworkID)
	if err != nil {
		return nil, nil, fmt.Errorf("ztnet: cache: %w", err)
	}

	v4 := make(map[string][]net.IP)
	v6 := make(map[string][]net.IP)
	for _, member := range members {
		hasV4 := len(member.IPs) > 0
		hasV6 := netInfo.RFC4193 || netInfo.SixPlane
		if !hasV4 && !hasV6 {
			continue
		}

		names := []string{member.Name + "." + nz.Zone, member.ID + "." + nz.Zone}
		for _, name := range names {
			fqdn := strings.ToLower(strings.TrimSuffix(name, ".") + ".")
			if hasV4 {
				v4[fqdn] = append(v4[fqdn], member.IPs...)
			}
			if netInfo.RFC4193 {
				ip, rfcErr := RFC4193(nz.NetworkID, member.ID)
				if rfcErr != nil {
					log.Errorf("ztnet: cache: rfc4193 %s/%s: %v", nz.NetworkID, member.ID, rfcErr)
				} else {
					v6[fqdn] = append(v6[fqdn], ip)
				}
			}
			if netInfo.SixPlane {
				ip, spErr := SixPlane(nz.NetworkID, member.ID)
				if spErr != nil {
					log.Errorf("ztnet: cache: 6plane %s/%s: %v", nz.NetworkID, member.ID, spErr)
				} else {
					v6[fqdn] = append(v6[fqdn], ip)
				}
			}
		}
	}

	return v4, v6, nil
}

func mergeRecords(dst map[string][]net.IP, src map[string][]net.IP) {
	for fqdn, ips := range src {
		dst[fqdn] = append(dst[fqdn], ips...)
	}
}
