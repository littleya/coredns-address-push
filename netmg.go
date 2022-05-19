package address_push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/coredns/coredns/plugin/pkg/log"
)

type netmgApiClient struct {
	host string
}

type apiNetworks struct {
	Networks []apiNetwork `json:"networks"`
}

type apiNetwork struct {
	Cidr   string `json:"cidr"`
	Domain string `json:"domain"`
	Group  string `json:"group"`
}

type NetmgClient struct {
	Client *netmgApiClient
}

func initNetmgClient(host string) ApiClient {

	apiClient := netmgApiClient{
		host: host,
	}

	return &NetmgClient{
		Client: &apiClient,
	}
}

func (n *NetmgClient) FetchAddress() []net.IPNet {
	var cidrs []net.IPNet

	resp, err := http.Get(fmt.Sprintf("http://%s/v1/nets", n.Client.host))
	if err != nil {
		log.Error(err)
		return cidrs
	}
	if resp.StatusCode/200 != 1 {
		log.Error("Failed to fetch address list")
		return cidrs
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return cidrs
	}
	var apiNetworksInstance apiNetworks
	if err := json.Unmarshal(body, &apiNetworksInstance); err != nil {
		log.Error(err)
		return cidrs
	}

	for _, apiNetworkInstance := range apiNetworksInstance.Networks {
		_, cidr, _ := net.ParseCIDR(apiNetworkInstance.Cidr)
		cidrs = append(cidrs, *cidr)
	}

	return cidrs
}

func (n *NetmgClient) PushAddress(network net.IPNet, domain, listName string) error {
	data := apiNetworks{
		Networks: []apiNetwork{
			{
				Cidr:   network.String(),
				Group:  listName,
				Domain: domain,
			},
		},
	}
	body, _ := json.Marshal(data)
	resp, err := http.Post(fmt.Sprintf("http://%s/v1/nets", n.Client.host), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	if resp.StatusCode/200 != 1 {
		return fmt.Errorf("the request status code not in 2xx: %d", resp.StatusCode)
	}
	return nil
}

func (n *NetmgClient) FetchAddressFromList(list string) []net.IPNet {
	var cidrs []net.IPNet

	resp, err := http.Get(fmt.Sprintf("http://%s/v1/nets/group/%s", n.Client.host, list))
	if err != nil {
		log.Error(err)
		return cidrs
	}
	if resp.StatusCode/200 != 1 {
		log.Error("Failed to fetch address list")
		return cidrs
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return cidrs
	}
	var apiNetworksInstance apiNetworks
	if err := json.Unmarshal(body, &apiNetworksInstance); err != nil {
		log.Error(err)
		return cidrs
	}

	for _, apiNetworkInstance := range apiNetworksInstance.Networks {
		_, cidr, _ := net.ParseCIDR(apiNetworkInstance.Cidr)
		cidrs = append(cidrs, *cidr)
	}

	return cidrs
}
