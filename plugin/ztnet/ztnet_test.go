package ztnet

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
)

func newPlugin() *ZTNet {
	rc := &RecordCache{}
	rc.Replace(
		map[string][]net.IP{"node.home.lan.": {net.ParseIP("10.0.0.2")}},
		map[string][]net.IP{"node.home.lan.": {net.ParseIP("fc00::1"), net.ParseIP("fc00::2")}},
	)
	return &ZTNet{Config: &Config{Networks: []NetworkZone{{Zone: "home.lan.", NetworkID: "8056c2e21c000001"}}, DNSTTL: 30 * time.Second}, Cache: rc}
}

func TestServeDNSA(t *testing.T) {
	z := newPlugin()
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	m := new(dns.Msg)
	m.SetQuestion("node.home.lan.", dns.TypeA)
	rcode, err := z.ServeDNS(context.Background(), rec, m)
	if err != nil || rcode != dns.RcodeSuccess {
		t.Fatalf("rcode=%d err=%v", rcode, err)
	}
	if len(rec.Msg.Answer) != 1 || rec.Msg.Answer[0].Header().Rrtype != dns.TypeA {
		t.Fatalf("unexpected answer %#v", rec.Msg.Answer)
	}
	if !rec.Msg.Authoritative {
		t.Fatal("expected AA=true")
	}
	for _, rr := range rec.Msg.Answer {
		if rr.Header().Rrtype != dns.TypeA {
			t.Fatalf("expected only A records, got %v", rr.Header().Rrtype)
		}
	}
}

func TestServeDNSAAAA(t *testing.T) {
	z := newPlugin()
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	m := new(dns.Msg)
	m.SetQuestion("node.home.lan.", dns.TypeAAAA)
	rcode, err := z.ServeDNS(context.Background(), rec, m)
	if err != nil || rcode != dns.RcodeSuccess {
		t.Fatalf("rcode=%d err=%v", rcode, err)
	}
	if len(rec.Msg.Answer) != 2 {
		t.Fatalf("expected 2 AAAA records, got %#v", rec.Msg.Answer)
	}
	for _, rr := range rec.Msg.Answer {
		if rr.Header().Rrtype != dns.TypeAAAA {
			t.Fatalf("expected only AAAA records, got %v", rr.Header().Rrtype)
		}
	}
}

func TestServeDNSSOAEmptyWhenNoFallthrough(t *testing.T) {
	z := newPlugin()
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	m := new(dns.Msg)
	m.SetQuestion("node.home.lan.", dns.TypeSOA)
	rcode, err := z.ServeDNS(context.Background(), rec, m)
	if err != nil || rcode != dns.RcodeSuccess {
		t.Fatalf("rcode=%d err=%v", rcode, err)
	}
	if len(rec.Msg.Answer) != 0 {
		t.Fatalf("expected empty answer got %#v", rec.Msg.Answer)
	}
}

func TestServeDNSUnknownNameInZone(t *testing.T) {
	z := newPlugin()
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	m := new(dns.Msg)
	m.SetQuestion("unknown.home.lan.", dns.TypeA)
	rcode, err := z.ServeDNS(context.Background(), rec, m)
	if err != nil || rcode != dns.RcodeSuccess || len(rec.Msg.Answer) != 0 {
		t.Fatalf("rcode=%d err=%v answer=%v", rcode, err, rec.Msg.Answer)
	}
}

func TestServeDNSOutsideZoneRefused(t *testing.T) {
	z := newPlugin()
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	m := new(dns.Msg)
	m.SetQuestion("example.org.", dns.TypeA)
	rcode, err := z.ServeDNS(context.Background(), rec, m)
	if err != nil || rcode != dns.RcodeRefused {
		t.Fatalf("rcode=%d err=%v", rcode, err)
	}
}

func TestServeDNSOutsideZoneFallthrough(t *testing.T) {
	z := newPlugin()
	z.Fall.SetZonesFromArgs(nil)
	z.Next = plugin.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		m := new(dns.Msg)
		m.SetReply(r)
		if err := w.WriteMsg(m); err != nil {
			return dns.RcodeServerFailure, err
		}
		return dns.RcodeSuccess, nil
	})
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	m := new(dns.Msg)
	m.SetQuestion("example.org.", dns.TypeA)
	rcode, err := z.ServeDNS(context.Background(), rec, m)
	if err != nil || rcode != dns.RcodeSuccess {
		t.Fatalf("rcode=%d err=%v", rcode, err)
	}
}

func TestLongestSuffixMatch(t *testing.T) {
	z := &ZTNet{Config: &Config{Networks: []NetworkZone{{Zone: "b.c.", NetworkID: "1"}, {Zone: "a.b.c.", NetworkID: "2"}}}}
	if !z.matchesZone("x.a.b.c.") {
		t.Fatal("expected zone match")
	}
}
