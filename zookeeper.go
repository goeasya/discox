package discover

import (
	"context"
	"fmt"
	"time"

	"github.com/go-zookeeper/zk"
)

type zookeeperDiscover struct {
	ctx         context.Context
	cancel      context.CancelFunc
	conn        *zk.Conn
	prefix      string
	serviceName string
}

func newZookeeperDiscover(cfg *DiscoverConfig) (*zookeeperDiscover, error) {
	conn, _, err := zk.Connect(cfg.BackendEndPoints, 10*time.Second)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	d := &zookeeperDiscover{
		ctx:         ctx,
		cancel:      cancel,
		conn:        conn,
		prefix:      cfg.DiscoverPrefix,
		serviceName: cfg.ServiceName,
	}
	return d, nil
}

func (d *zookeeperDiscover) Start(callback EndpointCacher) {
	d.discover(callback)
}

func (d *zookeeperDiscover) Stop() {
	d.cancel()
	if d.conn != nil {
		d.conn.Close()
	}
}

func (d *zookeeperDiscover) discover(callback EndpointCacher) {
	if err := d.listService(callback); err != nil {
		callback.AddError(err)
		return
	}
	for {
		snapshot, _, ch, err := d.conn.ChildrenW(d.key())
		if err != nil {
			callback.AddError(err)
			return
		}
		select {
		case e := <-ch:
			switch e.Type {
			case zk.EventNodeCreated, zk.EventNodeChildrenChanged:
				for _, v := range snapshot {
					callback.Delete(v)
				}
				if err := d.listService(callback); err != nil {
					callback.AddError(err)
				}
			case zk.EventNodeDeleted:
				for _, v := range snapshot {
					callback.Delete(v)
				}
			}
		}
	}
}

func (d *zookeeperDiscover) getNodeProperty(path string) ([]byte, error) {
	value, _, err := d.conn.Get(path)
	return value, err
}

func (d *zookeeperDiscover) listService(callback EndpointCacher) error {
	childs, _, err := d.conn.Children(d.key())
	if err != nil {
		return err
	}

	for _, c := range childs {
		value, _, err := d.conn.Get(fmt.Sprintf("%s/%s", d.key(), c))
		if err != nil {
			return err
		}
		callback.AddOrUpdate(c, value)
	}
	return nil
}

func (d *zookeeperDiscover) key() string {
	return fmt.Sprintf("%s/%s", d.prefix, d.serviceName)
}

type zookeeperRegister struct {
	zkEndpoints []string
	prefix      string
	serviceName string
	endpoint    string
	attr        string
	ttl         int64
	stopCh      chan struct{}
	conn        *zk.Conn
}

func newZookeeperRegister(cfg *RegisterConfig) (*zookeeperRegister, error) {
	r := zookeeperRegister{
		zkEndpoints: cfg.BackendEndPoints,
		prefix:      cfg.DiscoverPrefix,
		serviceName: cfg.ServiceName,
		endpoint:    cfg.ServiceEndPoint,
		attr:        cfg.Attr,
		ttl:         cfg.HeartBeatPeriod,
	}
	return &r, nil
}

func (r *zookeeperRegister) Start() error {
	var err error
	r.conn, _, err = zk.Connect(r.zkEndpoints, time.Second*5)
	if err != nil {
		return err
	}

	return r.register()
}

func (r *zookeeperRegister) Stop() error {
	if r.conn != nil {
		r.conn.Close()
	}
	return nil
}

func (r *zookeeperRegister) register() error {
	if err := r.createIfNotExist(r.node(), nil, 0); err != nil {
		return err
	}

	return r.createOrUpdateEndpoint(r.key(), []byte(r.attr))
}

func (r *zookeeperRegister) createOrUpdateEndpoint(path string, data []byte) error {
	exist, _, err := r.conn.Exists(path)
	if err != nil {
		return err
	}

	if !exist {
		_, err = r.conn.Create(path, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
		if err != nil {
			return err
		}
		return nil
	}

	_, stat, err := r.conn.Get(path)
	if err != nil {
		return err
	}
	_, err = r.conn.Set(path, []byte(r.attr), stat.Version)
	if err != nil {
		return err
	}

	return nil

}

func (r *zookeeperRegister) createIfNotExist(path string, data []byte, flag int32) error {
	exist, _, err := r.conn.Exists(r.node())
	if err != nil {
		return err
	}

	if !exist {
		_, err = r.conn.Create(path, data, flag, zk.WorldACL(zk.PermAll))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *zookeeperRegister) node() string {
	return fmt.Sprintf("%s/%s", r.prefix, r.serviceName)
}

func (r *zookeeperRegister) key() string {
	return fmt.Sprintf("%s/%s/%s", r.prefix, r.serviceName, r.endpoint)
}
