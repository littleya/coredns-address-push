package address_push

import (
	"context"
	"net"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/miekg/dns"
)

type AddressPush struct {
	Next plugin.Handler
	Keys []string
}

func (addrPush AddressPush) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	wrr := &addrPushResponseWriter{w, addrPush}
	return plugin.NextOrFailure(addrPush.Name(), addrPush.Next, ctx, wrr, r)
}

func (addrPush AddressPush) Name() string { return "address_push" }

type addrPushResponseWriter struct {
	dns.ResponseWriter
	AddressPush
}

func (rrw *addrPushResponseWriter) WriteMsg(res *dns.Msg) error {
	for _, key := range rrw.AddressPush.Keys {
		config := configs[key]
		config.lock.RLock()
		networks := config.networks
		config.lock.RUnlock()

		addrs := []net.IP{}
		for _, answer := range res.Answer {
			switch rr := answer.(type) {
			case *dns.A:
				isExist := false
				for _, cidr := range networks {
					if cidr.Contains(rr.A) {
						isExist = true
						break
					}
				}
				if !isExist {
					addrs = append(addrs, rr.A)
				}
			case *dns.AAAA:
				isExist := false
				for _, cidr := range networks {
					if cidr.Contains(rr.AAAA) {
						isExist = true
						break
					}
				}
				if !isExist {
					addrs = append(addrs, rr.AAAA)
				}
			}
		}

		if len(addrs) > 0 {
			for _, addr := range addrs {
				var (
					mask     net.IPMask
					listName string
				)
				ipVersion := "ipv4"
				ipListName := config.ipv4ListName
				if ip4 := addr.To4(); ip4 != nil {
					mask = net.CIDRMask(24, 32)
					listName = config.ipv4ListName
				} else {
					mask = net.CIDRMask(64, 128)
					listName = config.ipv6ListName
					ipVersion = "ipv6"
					ipListName = config.ipv6ListName
				}
				ipNet := net.IPNet{IP: addr.Mask(mask), Mask: mask}

				log.Infof("Found address %s not in %s list %s.", ipNet.String(), ipVersion, ipListName)
				err := config.apiClient.PushAddress(ipNet, res.Question[len(res.Question)-1].Name, listName)
				if err != nil {
					log.Error(err)
					return rrw.ResponseWriter.WriteMsg(res)
				}

				config.lock.Lock()
				config.networks = append(config.networks, ipNet)
				config.lock.Unlock()
			}
		}
	}

	return rrw.ResponseWriter.WriteMsg(res)
}

const (
	Enabled     = "enabled"
	Type        = "type"
	Host        = "host"
	AuthUser    = "auth_user"
	AuthKey     = "auth_key"
	IPv4        = "ipv4"
	IPv6        = "ipv6"
	EmptyString = ""
)
