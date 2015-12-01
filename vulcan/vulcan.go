package vulcan

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"

	"github.com/coreos/go-etcd/etcd"
	"github.com/gliderlabs/registrator/bridge"
)

func init() {
	bridge.Register(new(VulcanFactory), "vulcan")
}

type VulcanFactory struct{}

func (f *VulcanFactory) New(uri *url.URL) bridge.RegistryAdapter {
	urls := make([]string, 0)
	prefix := "vulcand"

	if uri.Host != "" {
		urls = append(urls, "http://"+uri.Host)
	}

	if len(uri.Path) >= 2 {
		prefix = uri.Path
	}

	return &VulcanAdapter{
		etcdClient: etcd.NewClient(urls),
		prefix:     prefix,
		log:        log.New(os.Stderr, fmt.Sprintf("%s: ", uri), 0),
	}
}

type VulcanAdapter struct {
	etcdClient *etcd.Client
	prefix     string
	log        *log.Logger
}

func (self *VulcanAdapter) Ping() error {
	rr := etcd.NewRawRequest("GET", "version", nil, nil)
	_, err := self.etcdClient.SendRequest(rr)
	if err != nil {
		return err
	}
	return nil
}

func (self *VulcanAdapter) setJSON(vulcanPath string, item interface{}, ttl int) error {
	var buf []byte
	var err error
	if buf, err = json.Marshal(item); err != nil {
		self.log.Println(vulcanPath, ": failed to marshal:", err)
		return err
	}
	if _, err = self.etcdClient.Set(path.Join(self.prefix, vulcanPath), string(buf), uint64(ttl)); err != nil {
		self.log.Println(vulcanPath, ": failed to register:", err)
		return err
	} else {
		self.log.Println(vulcanPath, ":", string(buf))
	}
	return nil
}

func (self *VulcanAdapter) del(vulcanPath string) error {
	if _, err := self.etcdClient.Delete(path.Join(self.prefix, vulcanPath), false); err != nil {
		self.log.Println(vulcanPath, ": failed to unregister:", err)
		return err
	}
	return nil
}

func (self *VulcanAdapter) Register(service *bridge.Service) error {
	attrs := serviceAttrs(service)

	if backendPath, backend := attrs.backend(); backendPath != "" {
		// shared across all servers; persistent
		if err := self.setJSON(backendPath, backend, 0); err != nil {
			return err
		}
	}

	if serverPath, server := attrs.server(); serverPath != "" {
		if err := self.setJSON(serverPath, server, service.TTL); err != nil {
			return err
		}
	}

	if frontendPath, frontend := attrs.frontend(); frontendPath != "" {
		// shared across all servers; persistent
		if err := self.setJSON(frontendPath, frontend, 0); err != nil {
			return err
		}
	}

	return nil
}

func (self *VulcanAdapter) Deregister(service *bridge.Service) error {
	attrs := serviceAttrs(service)

	if serverPath, _ := attrs.server(); serverPath != "" {
		if err := self.del(serverPath); err != nil {
			return err
		}
	}

	return nil
}

func (self *VulcanAdapter) Refresh(service *bridge.Service) error {
	attrs := serviceAttrs(service)

	// server only
	if serverPath, server := attrs.server(); serverPath != "" {
		if err := self.setJSON(serverPath, server, service.TTL); err != nil {
			return err
		}
	}
}
