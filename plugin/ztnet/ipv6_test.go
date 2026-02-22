package ztnet

import (
	"net"
	"testing"
)

func TestRFC4193(t *testing.T) {
	tests := []struct {
		networkID string
		nodeID    string
		want      string
	}{
		{networkID: "8056c2e21c000001", nodeID: "efcc1b0947", want: "fd80:56c2:e21c:0:199:93ef:cc1b:947"},
		{networkID: "0000000000000001", nodeID: "0000000001", want: "fd00::199:9300:0:1"},
		{networkID: "ffffffffffffffff", nodeID: "ffffffffff", want: "fdff:ffff:ffff:ffff:ff99:93ff:ffff:ffff"},
	}
	for _, tc := range tests {
		got, err := RFC4193(tc.networkID, tc.nodeID)
		if err != nil {
			t.Fatalf("RFC4193(%q,%q) error: %v", tc.networkID, tc.nodeID, err)
		}
		want := net.ParseIP(tc.want)
		if !got.Equal(want) {
			t.Fatalf("RFC4193 got %s want %s", got.String(), want.String())
		}
	}
}

func TestSixPlane(t *testing.T) {
	tests := []struct {
		networkID string
		nodeID    string
		want      string
	}{
		{networkID: "8056c2e21c000001", nodeID: "efcc1b0947", want: "fc9c:56c2:e3ef:cc1b:947::1"},
		{networkID: "0000000000000001", nodeID: "0000000001", want: "fc00:0:100:0:1::1"},
		{networkID: "ffffffffffffffff", nodeID: "ffffffffff", want: "fc00:0:ff:ffff:ffff::1"},
	}
	for _, tc := range tests {
		got, err := SixPlane(tc.networkID, tc.nodeID)
		if err != nil {
			t.Fatalf("SixPlane(%q,%q) error: %v", tc.networkID, tc.nodeID, err)
		}
		want := net.ParseIP(tc.want)
		if !got.Equal(want) {
			t.Fatalf("SixPlane got %s want %s", got.String(), want.String())
		}
	}
}

func TestIPv6InvalidInput(t *testing.T) {
	if _, err := RFC4193("", "efcc1b0947"); err == nil {
		t.Fatal("expected error for invalid networkID length")
	}
	if _, err := SixPlane("8056c2e21c000001", ""); err == nil {
		t.Fatal("expected error for invalid nodeID length")
	}
	if _, err := RFC4193("8056c2e21c00000Z", "efcc1b0947"); err == nil {
		t.Fatal("expected error for invalid networkID hex")
	}
}
