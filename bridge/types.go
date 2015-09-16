//go:generate go-extpoints . AdapterFactory
package bridge

import (
	"net/url"
)

type AdapterFactory interface {
	New(uri *url.URL) RegistryAdapter
}

type RegistryAdapter interface {
	Ping() error
	Register(service *Service) error
	Deregister(service *Service) error
	Refresh(service *Service) error
}

type Config struct {
	HostIp            string
	Internal          bool
	ForceTags         string
	RefreshTtl        int
	RefreshInterval   int
	DeregisterCheck   string
	RegisterContainer bool
}

type Service struct {
	ID    string
	Name  string
	Port  int
	IP    string
	Tags  []string
	Attrs map[string]string
	TTL   int

	Host         ServiceHost
	Container    ServiceContainer
	Origin       ServicePort
}

type DeadContainer struct {
	TTL      int
	Services []*Service
}

type ServiceHost struct {
	Hostname string
	IP       string
}

type ServiceContainer struct {
	Hostname    string
	ID          string
	ImageName   string
	InternalIP  string
	Meta        []string
}

type ServicePort struct {
	HostPort    string
	HostIP      string
	ExposedPort string
	PortType    string
}
