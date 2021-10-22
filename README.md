# discox

discox是一个服务注册发现的继承库

通过接口封装，屏蔽后端实现细节



目前支持类型

- zookeeper
- etcd
- consul



## 示例

### zookeeper

**server**

```go
package main

import (
	"fmt"
	"github.com/goeasya/discox"
	"os"
)

func main() {
	cfg := discox.RegisterConfig{
		BackendType:      discox.ZookeeperBackend,
		BackendEndPoints: []string{"10.1.1.1:2181"},
		DiscoverPrefix:   "/soaservices",
		ServiceName:      "demo",
		HeartBeatPeriod:  5,
		ServiceEndPoint:  "127.0.0.1:8111",
	}
	service, err := discox.NewRegister(&cfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err = service.Start(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer service.Stop()
	select {}
}
```

**client**

```go
package main

import (
	"fmt"
	"github.com/goeasya/discox"
	"os"
	"time"
)

func main() {
	timer := time.NewTimer(time.Second * 5)
	cfg := discox.DiscoverConfig{
		BackendEndPoints: []string{"10.1.1.1:2181"},
		BackendType:      discox.ZookeeperBackend,
		DiscoverPrefix:   "/soaservices",
		ServiceName:      "demo",
	}
	server, err := discox.NewDiscover(&cfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	endpointCacher := discox.NewLiteEndpoint()
	go server.Start(endpointCacher)
	defer server.Stop()

	for {
		select {
		case <-timer.C:
			fmt.Println("time 5 seconds")
			endpoints := endpointCacher.List()
			fmt.Println(endpoints, len(endpoints))
			timer.Reset(time.Second * 5)
		}
	}
}

```



### etcd

**server**

```go
package main

import (
	"fmt"
	"os"
	
	"github.com/goeasya/discox"
)

func main() {
	cfg := discox.RegisterConfig{
		BackendType:      discox.EtcdBackend,
		BackendEndPoints: []string{"http://10.1.1.1:23790"},
		DiscoverPrefix:   "/discox/etcddemo",
		ServiceName:      "demo",
		HeartBeatPeriod:  5,
		ServiceEndPoint:  "127.0.0.1:8111",
	}
	service, err := discox.NewRegister(&cfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err = service.Start(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer service.Stop()
	select {}
}
```

**client**

```go
package main

import (
	"fmt"
	"os"
	"time"
	
	"github.com/goeasya/discox"
)

func main() {
	timer := time.NewTimer(time.Second * 5)
	cfg := discox.DiscoverConfig{
		BackendEndPoints: []string{"http://10.1.1.1:23790"},
		BackendType:      discox.EtcdBackend,
		DiscoverPrefix:   "/discox/etcddemo",
		ServiceName:      "demo",
	}
	server, err := discox.NewDiscover(&cfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	endpointCacher := discox.NewLiteEndpoint()
	go server.Start(endpointCacher)
	defer server.Stop()

	for {
		select {
		case <-timer.C:
			fmt.Println("time 5 seconds")
			endpoints := endpointCacher.List()
			fmt.Println(endpoints, len(endpoints))
			timer.Reset(time.Second * 5)
		}
	}
}

```



### consul

**server**

```go
package main

import (
	"fmt"
	"net/http"
	"os"
	
	"github.com/goeasya/discox"
)

func main() {
	cfg := discox.RegisterConfig{
		BackendType:         discox.ConsulBackend,
		BackendEndPoints:    []string{"consul.test.com"},
		DiscoverPrefix:      "/soaservices",
		ServiceName:         "demo",
		HeartBeatPeriod:     5,
		ServiceEndPoint:     "172.18.1.1:8080",
		HealthCheckEndPoint: "172.18.1.1:8080/check",
	}
	service, err := discox.NewRegister(&cfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err = service.Start(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	http.HandleFunc("/check", consulCheck)
	go http.ListenAndServe(":8080", nil)
	defer service.Stop()
	select {}

}

var count int64

func consulCheck(w http.ResponseWriter, r *http.Request) {

	s := "consulCheck" + fmt.Sprint(count) + "remote:" + r.RemoteAddr + " " + r.URL.String()
	fmt.Println(s)
	fmt.Fprintln(w, s)
	count++
}

```



**client**

```go
package main

import (
	"fmt"
	"os"
	"time"
	
	"github.com/goeasya/discox"
)

func main() {
	timer := time.NewTimer(time.Second * 5)
	cfg := discox.DiscoverConfig{
		BackendEndPoints: []string{"consul.test.com"},
		BackendType:      discox.ConsulBackend,
		DiscoverPrefix:   "/soaservices",
		ServiceName:      "demo",
	}
	server, err := discox.NewDiscover(&cfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	endpointCacher := discox.NewLiteEndpoint()
	go server.Start(endpointCacher)
	defer server.Stop()

	for {
		select {
		case <-timer.C:
			fmt.Println("time 5 seconds")
			endpoints := endpointCacher.List()
			fmt.Println(endpoints, len(endpoints))
			timer.Reset(time.Second * 5)
		}
	}
}

```

