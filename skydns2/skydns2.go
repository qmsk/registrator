package skydns2

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"

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

func (r *Skydns2Adapter) Register(service *bridge.Service) error {
	record, err := json.Marshal(Service{r.serviceHost(service), service.Port})
	if err != nil {
		log.Println("skydns2: failed to marshal service:", err)
		return err
	}

	_, err = r.client.Set(r.servicePath(service), string(record), uint64(service.TTL))
	if err != nil {
		log.Println("skydns2: failed to register service:", err)
	}
	return err
}

func (r *Skydns2Adapter) Deregister(service *bridge.Service) error {
	_, err := r.client.Delete(r.servicePath(service), false)
	if err != nil {
		log.Println("skydns2: failed to register service:", err)
	}
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

func (r *Skydns2Adapter) serviceHost(service *bridge.Service) string {
	if service.Port == 0 {
		return service.IP
	} else {
		return r.containerDomain(service)
	}
}

func (r *Skydns2Adapter) servicePath(service *bridge.Service) string {
	if service.Port == 0 {
		return domainPath(r.containerDomain(service))
	} else {
		return domainPath(r.serviceDomain(service))
	}
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


