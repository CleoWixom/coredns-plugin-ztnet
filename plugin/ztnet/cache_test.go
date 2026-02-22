package ztnet

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestCacheReplaceLookup(t *testing.T) {
	rc := &RecordCache{}
	rc.Replace(
		map[string][]net.IP{"host.example.": {net.ParseIP("10.0.0.1")}},
		map[string][]net.IP{"host.example.": {net.ParseIP("fc00::1")}},
	)
	ips4, ok4 := rc.LookupA("host.example.")
	if !ok4 || len(ips4) != 1 || ips4[0].String() != "10.0.0.1" {
		t.Fatalf("unexpected A result ok=%v ips=%v", ok4, ips4)
	}
	ips6, ok6 := rc.LookupAAAA("host.example.")
	if !ok6 || len(ips6) != 1 || ips6[0].String() != "fc00::1" {
		t.Fatalf("unexpected AAAA result ok=%v ips=%v", ok6, ips6)
	}
}

func TestCacheLookupUnknown(t *testing.T) {
	rc := &RecordCache{}
	if ips, ok := rc.LookupA("unknown.example."); ok || ips != nil {
		t.Fatalf("expected nil,false got %v,%v", ips, ok)
	}
	if ips, ok := rc.LookupAAAA("unknown.example."); ok || ips != nil {
		t.Fatalf("expected nil,false got %v,%v", ips, ok)
	}
}

func TestCacheReplaceNil(t *testing.T) {
	rc := &RecordCache{}
	rc.Replace(nil, nil)
	if _, ok := rc.LookupA("anything."); ok {
		t.Fatal("did not expect found A record")
	}
	if _, ok := rc.LookupAAAA("anything."); ok {
		t.Fatal("did not expect found AAAA record")
	}
}

func TestCacheConcurrentReplaceLookup(t *testing.T) {
	rc := &RecordCache{}
	rc.Replace(map[string][]net.IP{"host.example.": {net.ParseIP("10.0.0.1")}}, nil)

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = rc.LookupA("host.example.")
		}()
		go func() {
			defer wg.Done()
			rc.Replace(map[string][]net.IP{"host.example.": {net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2")}}, nil)
		}()
	}
	wg.Wait()
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload string) {
	t.Helper()
	if _, err := fmt.Fprint(w, payload); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func TestBuildNetworkRecordsNameCollisionAndEmptyIPv4(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/network/8056c2e21c000001/":
			writeJSON(t, w, `{"v6AssignMode":{"6plane":false,"rfc4193":false}}`)
		case "/api/v1/network/8056c2e21c000001/member/":
			writeJSON(t, w, `[
				{"id":"efcc1b0947","name":"dup","authorized":true,"ipAssignments":["10.0.0.2"]},
				{"id":"efcc1b0948","name":"dup","authorized":true,"ipAssignments":["10.0.0.3"]},
				{"id":"efcc1b0949","name":"nov4","authorized":true,"ipAssignments":["fc00::1"]}
			]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	v4, v6, err := buildNetworkRecords(context.Background(), NewClient(ts.URL, "token"), NetworkZone{Zone: "home.lan.", NetworkID: "8056c2e21c000001"})
	if err != nil {
		t.Fatalf("buildNetworkRecords error: %v", err)
	}
	if len(v6) != 0 {
		t.Fatalf("expected no IPv6 records, got %v", v6)
	}

	dup := v4["dup.home.lan."]
	if len(dup) != 2 {
		t.Fatalf("expected merged dup name with 2 IPv4s, got %v", dup)
	}
	if _, ok := v4["nov4.home.lan."]; ok {
		t.Fatalf("did not expect key for member without IPv4: %v", v4["nov4.home.lan."])
	}
}
