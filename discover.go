package discover

import "fmt"

const (
	EtcdBackend      = "etcd"
	ZookeeperBackend = "zookeeper"
	ConsulBackend    = "consul"
)

type Discover interface {
	// Start watch with block
	Start(callback EndpointCacher)
	Stop()
}

type DiscoverConfig struct {
	BackendType      string   // one of etcd|consul|zookeeper
	BackendEndPoints []string // register backend endpoint
	DiscoverPrefix   string
	ServiceName      string
	HostName         string
}

func NewDiscover(cfg *DiscoverConfig) (Discover, error) {
	switch cfg.BackendType {
	case EtcdBackend:
		return newEtcdDiscover(cfg)
	case ConsulBackend:
		return newConsulDiscover(cfg)
	case ZookeeperBackend:
		return newZookeeperDiscover(cfg)
	}
	return nil, fmt.Errorf("unknown backend: %s, use etcd|consul|zookeeper", cfg.BackendType)
}

type Register interface {
	Start() error
	Stop() error
}

type RegisterConfig struct {
	BackendType         string   // one of etcd|consul|zookeeper
	BackendEndPoints    []string // register backend endpoint
	DiscoverPrefix      string
	ServiceName         string
	HeartBeatPeriod     int64
	ServiceEndPoint     string // register service endpoint to backend
	Attr                string // custom attribute. like: {"hostname": "xxx", "weight": 1}
	HealthCheckEndPoint string
}

func NewRegister(cfg *RegisterConfig) (Register, error) {
	switch cfg.BackendType {
	case EtcdBackend:
		return newEtcdRegister(cfg)
	case ConsulBackend:
		return newConsulRegister(cfg)
	case ZookeeperBackend:
		return newZookeeperRegister(cfg)
	}
	return nil, fmt.Errorf("unknown backend: %s, use etcd|consul|zookeeper", cfg.BackendType)
}
