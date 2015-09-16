package skydns2

import (
	"fmt"
	"encoding/json"
	"log"
	"net"
	"strings"
	"net/url"

	"github.com/coreos/go-etcd/etcd"
	"github.com/gliderlabs/registrator/bridge"
)

type Service struct {
	Host	string	`json:"host"`
	Port	int	`json:"port,omitempty"`
}

func init() {
	bridge.Register(new(Factory), "skydns2")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) bridge.RegistryAdapter {
	urls := make([]string, 0)
	if uri.Host != "" {
		urls = append(urls, "http://"+uri.Host)
	}

	if len(uri.Path) < 2 {
		log.Fatal("skydns2: dns domain required e.g.: skydns2://<host>/<domain>")
	}

	return &Skydns2Adapter{client: etcd.NewClient(urls), domain: uri.Path[1:]}
}

type Skydns2Adapter struct {
	client *etcd.Client
	domain string
}

func (r *Skydns2Adapter) Ping() error {
	rr := etcd.NewRawRequest("GET", "version", nil, nil)
	_, err := r.client.SendRequest(rr)
	if err != nil {
		return err
	}
	return nil
}

func (r *Skydns2Adapter) register(path string, service Service, ttl int) error {
	value, err := json.Marshal(service)
	if err != nil {
		log.Println("skydns2: failed to marshal service", path, ":", err)
	}

	_, err = r.client.Set(path, string(value), uint64(ttl))
	if err != nil {
		log.Println("skydns2: failed to register service", path, ":", err)
	}

	log.Println("skydns2: register service", path, ":", string(value))

	return nil
}

func (r *Skydns2Adapter) deregister(path string) error {
	_, err := r.client.Delete(path, false)
	if err != nil {
		log.Println("skydns2: failed to deregister service", path, ":", err)
	}
	return err
}

func (r *Skydns2Adapter) Register(service *bridge.Service) error {
	if reversePath := r.reversePath(service); reversePath != "" {
		if err := r.register(reversePath, Service{Host: r.containerDomain(service)}, service.TTL); err != nil {
			return err
		}
	}
	if containerPath := r.containerPath(service); containerPath != "" {
		if err := r.register(containerPath, Service{Host: service.IP}, service.TTL); err != nil {
			return err
		}
	}
	if servicePath := r.servicePath(service); servicePath != "" {
		if err := r.register(servicePath, Service{Host: r.containerDomain(service), Port: service.Port}, service.TTL); err != nil {
			return err
		}
	}
	return nil
}

func (r *Skydns2Adapter) Deregister(service *bridge.Service) error {
	err := r.deregister(r.servicePath(service))
	return err
}

func (r *Skydns2Adapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}

func (r *Skydns2Adapter) containerDomain(service *bridge.Service) string {
	return domainJoin(service.Container.Hostname, service.Host.Hostname, service.Name, r.domain)
}

func (r *Skydns2Adapter) serviceDomain(service *bridge.Service) string {
	return domainJoin(service.Container.Hostname, service.Host.Hostname, "_" + service.Origin.ExposedPort, "_" + service.Origin.PortType, service.Name, r.domain)
}

func (r *Skydns2Adapter) reversePath(service *bridge.Service) string {
	if service.Port != 0 {
		return ""
	}

	ip := net.ParseIP(service.IP)
	domain := reverseDomain(ip)
	if domain == "" {
		return ""
	}

	return domainPath(domain)
}

func (r *Skydns2Adapter) containerPath(service *bridge.Service) string {
	if service.Port != 0 {
		return ""
	}

	return domainPath(r.containerDomain(service))
}

func (r *Skydns2Adapter) servicePath(service *bridge.Service) string {
	if service.Port == 0 {
		return ""
	}

	return domainPath(r.serviceDomain(service))
}

func domainJoin(components... string) string {
	return strings.Join(components, ".")
}

func domainPath(domain string) string {
	components := strings.Split(domain, ".")
	for i, j := 0, len(components)-1; i < j; i, j = i+1, j-1 {
		components[i], components[j] = components[j], components[i]
	}
	return "/skydns/" + strings.Join(components, "/")
}

func reverseDomain(ip net.IP) string {
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa", ip4[3], ip4[2], ip4[1], ip4[0])
	} else {
		return ""
	}
}
