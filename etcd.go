package discover

import (
	"context"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/sirupsen/logrus"
)

// etcd discover impl
type etcdDiscover struct {
	ctx        context.Context
	cancel     context.CancelFunc
	etcdClient *clientv3.Client
	prefix     string
}

func newEtcdDiscover(cfg *DiscoverConfig) (*etcdDiscover, error) {
	cli, err := clientv3.New(clientv3.Config{Endpoints: cfg.BackendEndPoints})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &etcdDiscover{
		ctx:        ctx,
		cancel:     cancel,
		etcdClient: cli,
		prefix:     cfg.DiscoverPrefix,
	}, nil
}

func (d *etcdDiscover) Start(callback EndpointCacher) {
	d.discover(callback)
}

func (d *etcdDiscover) discover(callback EndpointCacher) {
	ctx, cancel := context.WithCancel(d.ctx)
	defer cancel()

	if err := d.listService(ctx, callback); err != nil {
		callback.AddError(err)
	}

	watch := d.etcdClient.Watch(ctx, d.prefix, clientv3.WithPrefix())
	for {
		select {
		case <-d.ctx.Done():
			return
		case resp := <-watch:
			if err := resp.Err(); err != nil {
				callback.AddError(err)
				return
			}
			for _, event := range resp.Events {
				if event.Kv == nil {
					continue
				}
				switch event.Type {
				case mvccpb.PUT:
					callback.AddOrUpdate(tailKey(event.Kv.Key), event.Kv.Value)
				case mvccpb.DELETE:
					callback.Delete(tailKey(event.Kv.Key))
				}
			}
		}
	}
}

func (d *etcdDiscover) Stop() {
	d.cancel()
}

func (d *etcdDiscover) listService(ctx context.Context, callback EndpointCacher) error {
	resp, err := d.etcdClient.Get(ctx, d.prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, kv := range resp.Kvs {
		callback.AddOrUpdate(tailKey(kv.Key), kv.Value)
	}
	return nil
}

// etcd register impl
type etcdRegister struct {
	etcdEndpoints  []string
	discoverPrefix string
	serviceName    string
	endpoint       string
	attr           string
	ttl            int64
	stopCh         chan struct{}
	leaseID        clientv3.LeaseID
	etcdClient     *clientv3.Client
}

func newEtcdRegister(cfg *RegisterConfig) (*etcdRegister, error) {
	var err error
	etcdClient, err := clientv3.New(clientv3.Config{Endpoints: cfg.BackendEndPoints})
	if err != nil {
		return nil, err
	}

	r := &etcdRegister{
		etcdEndpoints:  cfg.BackendEndPoints,
		discoverPrefix: cfg.DiscoverPrefix,
		serviceName:    cfg.ServiceName,
		endpoint:       cfg.ServiceEndPoint,
		attr:           cfg.Attr,
		ttl:            cfg.HeartBeatPeriod,
		stopCh:         make(chan struct{}),
		etcdClient:     etcdClient,
	}
	return r, nil
}

func (r *etcdRegister) Start() error {

	go r.keepAlive()
	return nil
}

func (r *etcdRegister) Stop() error {
	close(r.stopCh)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := r.etcdClient.Delete(ctx, r.key())
	//if r.grpcResolver != nil {
	//	return r.grpcResolver.Update(ctx, r.key(), grpcnaming.Update{Op: grpcnaming.Delete, Addr: r.endpoint})
	//}
	return err
}

func (r *etcdRegister) keepAlive() {
	duration := time.Duration(r.ttl) * time.Second
	timer := time.NewTimer(duration)
	for {
		select {
		case <-r.stopCh:
			return
		case <-timer.C:
			if r.leaseID > 0 {
				if err := r.leaseRenewal(); err != nil {
					logrus.Warnf("%s leaseid[%x] keepAlive err: %s, try to reset...", r.endpoint, r.leaseID, err.Error())
					r.leaseID = 0
				}
			} else {
				if err := r.register(); err != nil {
					logrus.Warnf("register endpoint %s error: %s", r.endpoint, err.Error())
				} else {
					logrus.Infof("register endppint %s success", r.endpoint)
				}
			}
			timer.Reset(duration)
		}
	}
}

func (r *etcdRegister) register() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	resp, err := r.etcdClient.Grant(ctx, r.ttl+3)
	if err != nil {
		return err
	}

	_, err = r.etcdClient.Put(ctx, r.key(), r.attr, clientv3.WithLease(resp.ID))
	r.leaseID = resp.ID
	return err
}

func (r *etcdRegister) leaseRenewal() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := r.etcdClient.KeepAliveOnce(ctx, r.leaseID)
	return err
}

func (r *etcdRegister) key() string {
	return toEtcdKey(r.discoverPrefix, r.serviceName, r.endpoint)
}

func toEtcdKey(elem ...string) string {
	return strings.Join(elem, "/")
}

func tailKey(key []byte) string {
	keyStr := string(key)
	topicSlice := strings.Split(keyStr, "/")
	if len(topicSlice) != 0 {
		return topicSlice[len(topicSlice)-1]
	}
	return keyStr
}
