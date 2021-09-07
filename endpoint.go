package discover

import "sync"

type EndpointCacher interface {
	AddOrUpdate(endpoint string, attribute []byte)
	Delete(endpoint string)
	AddError(err error)
	Error(err error)
}

// LiteEndpoint EndpointCacher lite impl
type LiteEndpoint struct {
	Endpoints map[string][]byte `json:"value"`
	lock      sync.Mutex
	Err       error
}

func NewLiteEndpoint() *LiteEndpoint {
	return &LiteEndpoint{
		Endpoints: map[string][]byte{},
		lock:      sync.Mutex{},
	}
}

func (e *LiteEndpoint) AddOrUpdate(endpoint string, attribute []byte) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.Endpoints[endpoint] = attribute
}

func (e *LiteEndpoint) Delete(endpoint string) {
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.Endpoints, endpoint)
}

func (e *LiteEndpoint) Error(err error) {
	e.Err = err
}

func (e *LiteEndpoint) List() []string {
	var endpointSlice []string
	for k, _ := range e.Endpoints {
		endpointSlice = append(endpointSlice, k)
	}
	return endpointSlice
}

func (e *LiteEndpoint) Attr(endpoint string) []byte {
	return e.Endpoints[endpoint]
}
