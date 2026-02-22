package ztnet

import (
	"net"
	"sync"
	"testing"
)

func TestCacheReplaceLookup(t *testing.T) {
	rc := &RecordCache{}
	rc.Replace(map[string][]net.IP{"host.example.": {net.ParseIP("10.0.0.1")}})
	ips, ok := rc.Lookup("host.example.")
	if !ok || len(ips) != 1 || ips[0].String() != "10.0.0.1" {
		t.Fatalf("unexpected result ok=%v ips=%v", ok, ips)
	}
}

func TestCacheLookupUnknown(t *testing.T) {
	rc := &RecordCache{}
	if ips, ok := rc.Lookup("unknown.example."); ok || ips != nil {
		t.Fatalf("expected nil,false got %v,%v", ips, ok)
	}
}

func TestCacheReplaceNil(t *testing.T) {
	rc := &RecordCache{}
	rc.Replace(nil)
	if _, ok := rc.Lookup("anything."); ok {
		t.Fatal("did not expect found record")
	}
}

func TestCacheConcurrentReplaceLookup(t *testing.T) {
	rc := &RecordCache{}
	rc.Replace(map[string][]net.IP{"host.example.": {net.ParseIP("10.0.0.1")}})

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = rc.Lookup("host.example.")
		}()
		go func() {
			defer wg.Done()
			rc.Replace(map[string][]net.IP{"host.example.": {net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2")}})
		}()
	}
	wg.Wait()
}
