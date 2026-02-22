package ztnet

import (
	"context"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

// ZTNet is the CoreDNS ztnet plugin handler.
type ZTNet struct {
	Next   plugin.Handler
	Config *Config
	Cache  *RecordCache
	Client *Client
	Fall   fall.F
}

// Name implements plugin.Handler.
func (z *ZTNet) Name() string { return "ztnet" }

// ServeDNS implements plugin.Handler.
func (z *ZTNet) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	qname := state.QName()

	if !z.matchesZone(qname) {
		if z.Fall.Through(qname) {
			return plugin.NextOrFailure(z.Name(), z.Next, ctx, w, r)
		}
		return dns.RcodeRefused, nil
	}

	if state.QType() != dns.TypeA && state.QType() != dns.TypeAAAA {
		if z.Fall.Through(qname) {
			return plugin.NextOrFailure(z.Name(), z.Next, ctx, w, r)
		}
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		if err := w.WriteMsg(m); err != nil {
			return dns.RcodeServerFailure, err
		}
		return dns.RcodeSuccess, nil
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	ttl := uint32(z.Config.DNSTTL.Seconds())

	switch state.QType() {
	case dns.TypeA:
		ips, ok := z.Cache.LookupA(qname)
		if ok {
			answers := make([]dns.RR, 0, len(ips))
			for _, ip := range ips {
				ip4 := ip.To4()
				if ip4 == nil {
					continue
				}
				answers = append(answers, &dns.A{Hdr: dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl}, A: ip4})
			}
			m.Answer = answers
		}
	case dns.TypeAAAA:
		ips, ok := z.Cache.LookupAAAA(qname)
		if ok {
			answers := make([]dns.RR, 0, len(ips))
			for _, ip := range ips {
				if ip.To4() != nil {
					continue
				}
				ip16 := ip.To16()
				if ip16 == nil {
					continue
				}
				answers = append(answers, &dns.AAAA{Hdr: dns.RR_Header{Name: qname, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl}, AAAA: ip16})
			}
			m.Answer = answers
		}
	}

	if err := w.WriteMsg(m); err != nil {
		return dns.RcodeServerFailure, err
	}
	return dns.RcodeSuccess, nil
}

func (z *ZTNet) matchesZone(qname string) bool {
	longest := ""
	for _, nz := range z.Config.Networks {
		if strings.HasSuffix(qname, nz.Zone) && len(nz.Zone) > len(longest) {
			longest = nz.Zone
		}
	}
	return longest != ""
}
