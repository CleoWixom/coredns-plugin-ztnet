package ztnet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetMembersFiltersAndNormalizes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/network/8056c2e21c000001/member/" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`[
			{"id":"efcc1b0947","name":"node one","authorized":true,"ipAssignments":["10.0.0.2","fc00::1"]},
			{"id":"efcc1b0948","name":"node two","authorized":true,"ipAssignments":[]},
			{"id":"efcc1b0949","name":"node three","authorized":true,"ipAssignments":["invalid"]},
			{"id":"deadbeef00","name":"ignored","authorized":false,"ipAssignments":["10.0.0.3"]}
		]`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "token")
	members, err := c.GetMembers(context.Background(), "8056c2e21c000001")
	if err != nil {
		t.Fatalf("GetMembers error: %v", err)
	}
	if len(members) != 3 {
		t.Fatalf("want 3 authorized members, got %d", len(members))
	}
	if members[0].Name != "node_one" {
		t.Fatalf("want normalized name node_one, got %q", members[0].Name)
	}
	if len(members[0].IPs) != 1 || members[0].IPs[0].String() != "10.0.0.2" {
		t.Fatalf("unexpected IP list %#v", members[0].IPs)
	}
	if len(members[1].IPs) != 0 {
		t.Fatalf("expected empty IP list, got %#v", members[1].IPs)
	}
	if len(members[2].IPs) != 0 {
		t.Fatalf("expected empty IP list for invalid assignment, got %#v", members[2].IPs)
	}
}

func TestGetNetworkInfo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"v6AssignMode":{"6plane":true,"rfc4193":false}}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "token")
	info, err := c.GetNetworkInfo(context.Background(), "8056c2e21c000001")
	if err != nil {
		t.Fatalf("GetNetworkInfo error: %v", err)
	}
	if !info.SixPlane || info.RFC4193 {
		t.Fatalf("unexpected info %#v", info)
	}
}

func TestAPIHTTPErrorWrapped(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "token")
	if _, err := c.GetMembers(context.Background(), "8056c2e21c000001"); err == nil {
		t.Fatal("expected error")
	}
}
