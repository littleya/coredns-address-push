package address_push

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"gitea.littleya.me/littleya/vyos"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/tidwall/gjson"
)

type VyosClient struct {
	Client *vyos.Client
}

func initVyosClient(host, authKey string) ApiClient {

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	apiClient, err := vyos.NewClient(fmt.Sprintf("https://%s", host), authKey, tlsConfig)
	if err != nil {
		panic(err)
	}

	return &VyosClient{
		Client: apiClient,
	}
}

func (v *VyosClient) FetchAddress() []net.IPNet {
	var cidrs []net.IPNet

	resp, err := v.Client.RetrieveShowConfig("firewall", "group")
	if err != nil {
		log.Error(err)
		return []net.IPNet{}
	}
	if !resp.Success {
		log.Error(resp.Err)
		return []net.IPNet{}
	}

	resp.Data.Get("network-group").ForEach(func(key, value gjson.Result) bool {
		for _, addr := range value.Get("network").Array() {
			addrString := addr.String()
			if !strings.Contains(addrString, "/") {
				addrString += "/32"
			}
			_, cidr, _ := net.ParseCIDR(addrString)
			cidrs = append(cidrs, *cidr)
		}
		return true
	})

	return cidrs
}

func (v *VyosClient) PushAddress(network net.IPNet, domain, listName string) error {
	var networkGroup string
	if ip4 := network.IP.To4(); ip4 != nil {
		networkGroup = "network-group"
	} else {
		networkGroup = "ipv6-network-group"
	}

	resp, err := v.Client.ConfigureSet("firewall", "group", networkGroup, listName, "network", network.String())
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(resp.Err)
	}

	resp, err = v.Client.ConfigFileSave()
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(resp.Err)
	}
	return nil
}

func (v *VyosClient) FetchAddressFromList(list string) []net.IPNet {
	var cidrs []net.IPNet

	resp, err := v.Client.RetrieveShowConfig("firewall", "group", "network-group", list)
	if err != nil {
		log.Error(err)
		return []net.IPNet{}
	}
	if !resp.Success {
		log.Error(resp.Err)
		return []net.IPNet{}
	}

	for _, addr := range resp.Data.Get("network").Array() {
		addrString := addr.String()
		if !strings.Contains(addrString, "/") {
			addrString += "/32"
		}

		_, cidr, _ := net.ParseCIDR(addrString)
		cidrs = append(cidrs, *cidr)
	}

	return cidrs
}
