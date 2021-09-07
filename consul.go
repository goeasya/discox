package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
)

type consulDiscover struct {
	ctx         context.Context
	cancel      context.CancelFunc
	prefix      string
	serviceName string
	client      *consulapi.Client
}

func newConsulDiscover(cfg *DiscoverConfig) (*consulDiscover, error) {
	config := consulapi.DefaultConfig()
	config.Address = strings.Join(cfg.BackendEndPoints, ",")
	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	d := &consulDiscover{
		ctx:         ctx,
		cancel:      cancel,
		prefix:      cfg.DiscoverPrefix,
		serviceName: cfg.ServiceName,
		client:      client,
	}
	return d, nil
}

func (d *consulDiscover) Start(callback EndpointCacher) {
	d.discover(callback)
}

func (d *consulDiscover) Stop() {
	d.cancel()
}

func (d *consulDiscover) discover(callback EndpointCacher) {
	var lastIndex uint64
	for {
		select {
		case <-d.ctx.Done():
			return
		default:
			services, queryMeta, err := d.client.Health().Service(
				d.serviceName, "", false, &consulapi.QueryOptions{
					WaitIndex: lastIndex,
				})
			if err != nil {
				callback.AddError(err)
			}
			lastIndex = queryMeta.LastIndex

			for _, service := range services {
				var attr []byte
				endpoint := fmt.Sprintf("%s:%v", service.Service.Address, service.Service.Port)
				switch service.Checks.AggregatedStatus() {
				case consulapi.HealthPassing:
					if service.Service.Meta != nil {
						attr, _ = json.Marshal(service.Service.Meta)
					}
					callback.AddOrUpdate(endpoint, attr)
				case consulapi.HealthCritical, consulapi.HealthWarning:
					callback.Delete(endpoint)
				}
			}
		}
	}
}

func (d *consulDiscover) listService() {
	d.client.Agent().Services()
}

type consulRegister struct {
	prefix              string
	serviceName         string
	serviceId           string
	endpoint            string
	healthCheckEndpoint string
	attr                string
	ttl                 int64
	stopCh              chan struct{}
	client              *consulapi.Client
}

func newConsulRegister(cfg *RegisterConfig) (*consulRegister, error) {
	config := consulapi.DefaultConfig()
	config.Address = strings.Join(cfg.BackendEndPoints, ",")
	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	r := &consulRegister{
		prefix:              cfg.DiscoverPrefix,
		serviceName:         cfg.ServiceName,
		endpoint:            cfg.ServiceEndPoint,
		healthCheckEndpoint: cfg.HealthCheckEndPoint,
		attr:                cfg.Attr,
		ttl:                 cfg.HeartBeatPeriod,
		stopCh:              make(chan struct{}),
		client:              client,
	}
	return r, nil
}

func (s *consulRegister) Start() error {
	return s.register()
}

func (s *consulRegister) Stop() error {
	if s.serviceId != "" {
		return s.client.Agent().ServiceDeregister(s.serviceId)
	}
	return nil
}

func (s *consulRegister) register() error {
	registration := new(consulapi.AgentServiceRegistration)

	address, port, err := net.SplitHostPort(s.endpoint)
	if err != nil {
		return err
	}

	registration.Address = address
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	registration.Port = portInt

	serviceId := fmt.Sprintf("%s_%s", s.serviceName, address)
	s.serviceId = serviceId

	registration.Name = s.serviceName
	registration.ID = serviceId

	serviceCheck := new(consulapi.AgentServiceCheck)
	serviceCheck.HTTP = fmt.Sprintf("http://%s", s.healthCheckEndpoint)
	serviceCheck.Timeout = "2s"
	serviceCheck.Interval = "2s"
	serviceCheck.DeregisterCriticalServiceAfter = "30s"

	registration.Check = serviceCheck
	return s.client.Agent().ServiceRegister(registration)
}
