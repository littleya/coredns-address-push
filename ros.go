package address_push

import (
	"crypto/tls"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin/pkg/log"
	routeros "github.com/littleya/routeros-rest"
	routeros_types "github.com/littleya/routeros-rest/types"
)

type RosClient struct {
	Client *routeros.Client
}

func initRosClient(host, authUser, authKey string) ApiClient {

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	apiClient, err := routeros.NewClient(host, authUser, authKey, tlsConfig)
	if err != nil {
		panic(err)
	}

	return &RosClient{
		Client: apiClient,
	}

}

func (r *RosClient) FetchAddress() []net.IPNet {

	ret, err := r.Client.GetIPFirewallAddresslist(map[string]string{})
	if err != nil {
		log.Error(err)
		return []net.IPNet{}
	}

	return handleRosAddress(ret)
}

func (r *RosClient) PushAddress(network net.IPNet, domain, listName string) error {
	if ip4 := network.IP.To4(); ip4 == nil {
		// TODO(ywang): Current routeros-rest do not support ipv6 push
		return nil
	}
	_, err := r.Client.PutIPFirewallAddresslist(routeros_types.IPFirewallAddresslist{
		List:    listName,
		Address: network.String(),
	})
	return err
}

func (r *RosClient) FetchAddressFromList(list string) []net.IPNet {

	ret, err := r.Client.GetIPFirewallAddresslist(map[string]string{
		"list": list,
	})
	if err != nil {
		log.Error(err)
		return []net.IPNet{}
	}

	return handleRosAddress(ret)
}

func handleRosAddress(ret routeros_types.IPFirewallAddresslists) []net.IPNet {
	var cidrs []net.IPNet
	for _, r := range ret {
		addr := r.Address
		if !strings.Contains(addr, "/") {
			addr += "/32"
		}
		_, cidr, _ := net.ParseCIDR(addr)
		cidrs = append(cidrs, *cidr)
	}

	return cidrs
}
