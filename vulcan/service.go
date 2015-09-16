package vulcan

import (
	"fmt"
	"path"

	"github.com/gliderlabs/registrator/bridge"
)

type ServiceAttrs struct {
	Server       string
	ServerURL    string
	Backend      string
	BackendType  string
	Frontend     string
	FrontendType string
	Route        string
}

// Interpret a bridge.Service's vulcan_* Attrs
func serviceAttrs(service *bridge.Service) (out ServiceAttrs) {
	out.FrontendType = FrontendType
	out.BackendType = BackendType

	if route, set := service.Attrs["vulcan_route"]; set {
		out.Route = route
	} else if host, set := service.Attrs["vulcan_host"]; set {
		out.Route = fmt.Sprintf("Host(\"%s\")", host)
	}

	if backend, set := service.Attrs["vulcan_backend"]; set {
		out.Backend = backend
	} else if vulcan, set := service.Attrs["vulcan"]; set {
		out.Backend = vulcan
	} else if out.Route != "" {
		out.Backend = service.Name
	}

	if frontend, set := service.Attrs["vulcan_frontend"]; set {
		out.Frontend = frontend
	} else if out.Route != "" {
		out.Frontend = out.Backend
	}

	if server, set := service.Attrs["vulcan_server"]; set {
		out.Server = server
	} else if _, set := service.Attrs["vulcan"]; set {
		out.Server = service.ID
	}

	out.ServerURL = fmt.Sprintf("http://%s:%d", service.IP, service.Port)

	return
}

func (self ServiceAttrs) server() (serverPath string, server BackendServer) {
	if self.Server == "" {
		return
	}

	return path.Join("backends", self.Backend, "servers", self.Server), BackendServer{
		URL: self.ServerURL,
	}
}

func (self *ServiceAttrs) backend() (backendPath string, backend Backend) {
	if self.Backend == "" {
		return
	}

	return path.Join("backends", self.Backend, "backend"), Backend{
		Type: self.BackendType,
	}
}

func (self *ServiceAttrs) frontend() (frontendPath string, frontend Frontend) {
	if self.Frontend == "" {
		return
	}

	return path.Join("frontends", self.Frontend, "frontend"), Frontend{
		Type:      self.FrontendType,
		BackendId: self.Backend,
		Route:     self.Route,
	}
}
