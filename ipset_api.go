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

type ipsetClient struct {
	host string
}

type ipsetApiNetworks struct {
	Network []string `json:"network"`
}

type IpsetApiClient struct {
	Client *ipsetClient
}

func initIpsetApiClient(host string) ApiClient {

	apiClient := ipsetClient{
		host: host,
	}

	return &IpsetApiClient{
		Client: &apiClient,
	}
}

func (c *IpsetApiClient) FetchAddress() []net.IPNet {
	var cidrs []net.IPNet

	resp, err := http.Get(fmt.Sprintf("http://%s/v2/networks", c.Client.host))
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
	var networks []string
	if err := json.Unmarshal(body, &networks); err != nil {
		log.Error(err)
		return cidrs
	}

	for _, network := range networks {
		_, cidr, _ := net.ParseCIDR(network)
		cidrs = append(cidrs, *cidr)
	}

	return cidrs
}

func (c *IpsetApiClient) PushAddress(network net.IPNet, domain, listName string) error {
	data := ipsetApiNetworks{
		Network: []string{network.String()},
	}
	body, _ := json.Marshal(data)

	resp, err := http.Post(fmt.Sprintf("http://%s/v2/networks/%s", c.Client.host, listName), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	if resp.StatusCode/200 != 1 {
		return fmt.Errorf("the request status code not in 2xx: %d", resp.StatusCode)
	}
	return nil
}

func (c *IpsetApiClient) FetchAddressFromList(list string) []net.IPNet {
	var cidrs []net.IPNet

	resp, err := http.Get(fmt.Sprintf("http://%s/v2/networks/%s", c.Client.host, list))
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
	var networks []string
	if err := json.Unmarshal(body, &networks); err != nil {
		log.Error(err)
		return cidrs
	}

	for _, network := range networks {
		_, cidr, _ := net.ParseCIDR(network)
		cidrs = append(cidrs, *cidr)
	}

	return cidrs
}
