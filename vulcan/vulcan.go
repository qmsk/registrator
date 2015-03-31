package vulcan

import (
	"encoding/json"
    "fmt"
	"log"
	"net/url"
    "path"

    "github.com/coreos/go-etcd/etcd"
	"github.com/gliderlabs/registrator/bridge"
)

const (
    FrontendType    = "http"
    BackendType     = "http"
)

type ServiceAttrs struct {
    Server          string
    Backend         string
    BackendType     string
    Frontend        string
    FrontendType    string
    Route           string
}

/*
 * etcd /vulcand/frontend/...
 */
type Frontend struct {
    Type        string
    BackendId   string
    Route       string
}

/*
 * etcd /vulcand/backends/...
 */
type Backend struct {
    Type    string
}

/*
 * etcd /vulcand/backends/.../servers/... JSON
 */
type BackendServer struct {
    URL     string
}

func init() {
	bridge.Register(new(Factory), "vulcan")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) bridge.RegistryAdapter {
	urls := make([]string, 0)
    prefix := "vulcand"

	if uri.Host != "" {
		urls = append(urls, "http://"+uri.Host)
	}

	if len(uri.Path) >= 2 {
        prefix = uri.Path
	}

	return &VulcanAdapter{client: etcd.NewClient(urls), prefix: prefix}
}

type VulcanAdapter struct {
	client *etcd.Client
	prefix  string
}

func (r *VulcanAdapter) Ping() error {
	rr := etcd.NewRawRequest("GET", "version", nil, nil)
	_, err := r.client.SendRequest(rr)
	if err != nil {
		return err
	}
	return nil
}

func (r *VulcanAdapter) setJSON(path string, item interface{}, ttl int) error {
    var buf []byte
    var err error
    if buf, err = json.Marshal(item); err != nil {
        log.Println("vulcan", path, ": failed to marshal:", err)
		return err
    }
    if _, err = r.client.Set(path, string(buf), uint64(ttl));  err != nil {
        log.Println("vulcan", path, ": failed to register service:", err)
        return err
    } else {
        log.Println("vulcan", path, ":", string(buf))
    }
    return nil
}

func (r *VulcanAdapter) serviceAttrs(service *bridge.Service) (out ServiceAttrs) {
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

    return out
}

func (r *VulcanAdapter) servicePath(service *bridge.Service) string {
    a := r.serviceAttrs(service)

    if a.Server != "" {
        return path.Join(r.prefix, "backends", a.Backend, "servers", a.Server)
    } else {
        return ""
    }
}

func (r *VulcanAdapter) Register(service *bridge.Service) error {
    a := r.serviceAttrs(service)

    if a.Backend != "" {
        if err := r.setJSON(path.Join(r.prefix, "backends", a.Backend, "backend"),
                Backend{
                    Type: a.BackendType,
                },
                service.TTL); err != nil {
            return err
        }
    }

    if path := r.servicePath(service); path != "" {
        if err := r.setJSON(path,
                BackendServer{
                    URL: fmt.Sprintf("http://%s:%d", service.IP, service.Port),
                },
                service.TTL); err != nil {
            return err
        }
    }

    if a.Frontend != "" {
        if err := r.setJSON(path.Join(r.prefix, "frontends", a.Frontend, "frontend"),
                Frontend{
                    Type:       a.FrontendType,
                    BackendId:  a.Backend,
                    Route:      a.Route,
                },
                service.TTL); err != nil {
            return err
        }
    }

    return nil
}

func (r *VulcanAdapter) Deregister(service *bridge.Service) error {
    if path := r.servicePath(service); path != "" {
        if _, err := r.client.Delete(r.servicePath(service), false); err != nil {
            log.Println("skydns2: failed to register service:", err)
            return err
        }
    }

    return nil
}

func (r *VulcanAdapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}
