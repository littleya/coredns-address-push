package address_push

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
)

func init() { plugin.Register("address_push", setup) }

type ApiClient interface {
	FetchAddress() []net.IPNet
	PushAddress(network net.IPNet, domain, listName string) error
	// ListName() string
	FetchAddressFromList(list string) []net.IPNet
}

// type addressList struct {
// 	ipv4ListName string
// 	ipv6ListName string
// 	ipv4Networks []net.IPNet
// 	ipv6Networks []net.IPNet
// }

type Config struct {
	// networks            addressList
	networks            []net.IPNet
	ipv4ListName        string
	ipv6ListName        string
	apiClient           ApiClient
	periodicSyncEnabled bool
	lock                *sync.RWMutex
}

var configs = map[string]*Config{}

func setup(c *caddy.Controller) error {
	// addrSource, listName, err := flagParse(c)
	// if err != nil {
	// 	panic(err)
	// }

	// var conf *Config
	// var ok bool
	// if conf, ok = configs[addrSource+listName]; !ok {
	// 	client := GetClient(addrSource, listName)
	// 	conf = &Config{networks: []net.IPNet{}, apiClient: client, periodicSyncEnabled: false, lock: &sync.RWMutex{}}
	// 	conf.syncAddrList()
	// 	configs[addrSource+listName] = conf
	// }

	// ap := AddressPush{Config: conf}
	ap, err := parseAddressPush(c)
	if err != nil {
		// plugin.Error("address_push", err)
		panic(err)
	}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		ap.Next = next
		return ap
	})

	return nil
}

func parseAddressPush(c *caddy.Controller) (*AddressPush, error) {
	keys := []string{}
	for c.Next() {
		rc, err := parseStanza(c)
		if err != nil {
			return nil, err
		}
		if !rc.Enabled {
			continue
		}
		var (
			conf *Config
			ok   bool
		)
		if _, ok = configs[rc.Hash()]; !ok {
			client := rc.GetClient()
			conf = &Config{
				networks:            []net.IPNet{},
				ipv4ListName:        rc.IPv4,
				ipv6ListName:        rc.IPv6,
				apiClient:           client,
				periodicSyncEnabled: false,
				lock:                &sync.RWMutex{},
			}
			conf.syncAddrList()
			configs[rc.Hash()] = conf
		}
		keys = append(keys, rc.Hash())
	}
	ap := &AddressPush{Keys: keys}

	return ap, nil
}

type RawConfig struct {
	Enabled  bool
	Type     string
	Host     string
	AuthUser string
	AuthKey  string
	IPv4     string // ipv4 address list
	IPv6     string // ipv6 address list
}

func parseStanza(c *caddy.Controller) (*RawConfig, error) {
	rc := &RawConfig{
		Enabled: true,
	}
	for c.NextBlock() {
		switch c.Val() {
		case Enabled:
			args := c.RemainingArgs()
			if len(args) != 1 {
				return rc, c.ArgErr()
			}
			if penabled, err := strconv.ParseBool(args[0]); err == nil {
				rc.Enabled = penabled
			} else {
				return rc, err
			}
		case Type:
			args := c.RemainingArgs()
			if len(args) == 0 {
				return rc, c.ArgErr()
			}
			if !(args[0] == "routeros" || args[0] == "vyos" || args[0] == "netmg" || args[0] == "ipsetapi") {
				return rc, c.ArgErr()
			}
			rc.Type = args[0]
		case Host:
			args := c.RemainingArgs()
			if len(args) != 1 {
				return rc, c.ArgErr()
			}
			if _, _, err := net.SplitHostPort(args[0]); err == nil {
				rc.Host = args[0]
			} else {
				return rc, err
			}
		case AuthUser:
			args := c.RemainingArgs()
			if len(args) != 1 {
				return rc, c.ArgErr()
			}
			rc.AuthUser = args[0]
		case AuthKey:
			args := c.RemainingArgs()
			if len(args) != 1 {
				return rc, c.ArgErr()
			}
			rc.AuthKey = args[0]
		case IPv4:
			args := c.RemainingArgs()
			if len(args) != 1 {
				return rc, c.ArgErr()
			}
			rc.IPv4 = args[0]
		case IPv6:
			args := c.RemainingArgs()
			if len(args) != 1 {
				return rc, c.ArgErr()
			}
			rc.IPv6 = args[0]
		default:
		}
	}
	if rc.Type == "" || rc.Host == "" || (rc.IPv4 == "" && rc.IPv6 == "") {
		return rc, errors.New("missing required fields")
	}

	return rc, nil
}

func (rc *RawConfig) GetClient() ApiClient {

	switch rc.Type {
	case "routeros":
		return initRosClient(rc.Host, rc.AuthUser, rc.AuthKey)
	case "vyos":
		return initVyosClient(rc.Host, rc.AuthKey)
	case "netmg":
		return initNetmgClient(rc.Host)
	case "ipsetapi":
		return initIpsetApiClient(rc.Host)
	}

	panic("The method not impl")
}

func (rc *RawConfig) Hash() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", rc)))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (c *Config) syncAddrList() {

	if len(c.networks) == 0 {
		c.lock.Lock()
		c.networks = c.apiClient.FetchAddress()
		c.lock.Unlock()
	}

	if !c.periodicSyncEnabled {
		c.periodicSyncEnabled = true
		go func() {
			for {
				time.Sleep(time.Second * 15)
				log.Infof("Syncing address list from plugin address push.")

				networks := c.apiClient.FetchAddress()
				c.lock.Lock()
				c.networks = networks
				c.lock.Unlock()
			}
		}()
	}
}
