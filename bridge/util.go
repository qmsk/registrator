package bridge

import (
	"path"
	"strconv"
	"strings"

	"github.com/cenkalti/backoff"
	dockerapi "github.com/fsouza/go-dockerclient"
)

func retry(fn func() error) error {
	return backoff.Retry(fn, backoff.NewExponentialBackOff())
}

func mapDefault(m map[string]string, key, default_ string) string {
	v, ok := m[key]
	if !ok || v == "" {
		return default_
	}
	return v
}

func combineTags(tagParts ...string) []string {
	tags := make([]string, 0)
	for _, element := range tagParts {
		if element != "" {
			tags = append(tags, strings.Split(element, ",")...)
		}
	}
	return tags
}

func serviceMetaData(meta []string, port string) map[string]string {
	metadata := make(map[string]string)
	for _, kv := range meta {
		kvp := strings.SplitN(kv, "=", 2)
		if strings.HasPrefix(kvp[0], "SERVICE_") && len(kvp) > 1 {
			key := strings.ToLower(strings.TrimPrefix(kvp[0], "SERVICE_"))
			portkey := strings.SplitN(key, "_", 2)
			_, err := strconv.Atoi(portkey[0])
			if err == nil && len(portkey) > 1 {
				if portkey[0] != port {
					continue
				}
				metadata[portkey[1]] = kvp[1]
			} else {
				metadata[key] = kvp[1]
			}
		}
	}
	return metadata
}

func serviceContainer(container *dockerapi.Container) ServiceContainer {
	meta := container.Config.Env
	for k, v := range container.Config.Labels {
		meta = append(meta, k + "=" + v)
	}
	return ServiceContainer{
		Hostname:    container.Config.Hostname,
		ID:          container.ID,
		InternalIP:  container.NetworkSettings.IPAddress,
		ImageName:   strings.Split(path.Base(container.Config.Image), ":")[0],
		Meta:        meta,
	}
}

func servicePort(port dockerapi.Port, published []dockerapi.PortBinding) ServicePort {
	servicePort := ServicePort{
		ExposedPort: port.Port(),
		PortType:    port.Proto(),
	}

	if len(published) > 0 {
		servicePort.HostIP = published[0].HostIP
		servicePort.HostPort = published[0].HostPort
	}

	return servicePort
}
