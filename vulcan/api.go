package vulcan

// etcd /vulcand/... API definitions

const (
	FrontendType = "http"
	BackendType  = "http"
)

// etcd /vulcand/frontend/...
type Frontend struct {
	Type      string
	BackendId string
	Route     string
}

// etcd /vulcand/backends/...
type Backend struct {
	Type string
}

// etcd /vulcand/backends/.../servers/... JSON
type BackendServer struct {
	URL string
}
