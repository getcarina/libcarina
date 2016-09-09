# libcarina

[![GoDoc](https://godoc.org/github.com/rackerlabs/libcarina?status.png)](https://godoc.org/github.com/rackerlabs/libcarina)

Provisional Go bindings for the beta release of [Carina](https://getcarina.com) by Rackspace. If you're looking for a client binding, [`carina`](https://github.com/rackerlabs/carina) is your friend.

![](https://cloud.githubusercontent.com/assets/836375/10503963/e5bcca8c-72c0-11e5-8e14-2c1697297d7e.png)

## Examples

### Create
Create a new cluster

```go
package main

import (
	"time"

	"github.com/getcarina/libcarina"
)

func createCluster(username string, apikey string, clusterName string) error {
	// Connect to Carina
	cli, _ := libcarina.NewClusterClient(libcarina.BetaEndpoint, username, apikey)

	// Create a new cluster
	cluster, err := cli.Create(libcarina.Cluster{
	    Name: clusterName,
	    COE: "swarm",
	    HostType: "lxc",
	})

	// Wait for the cluster to become active
	for cluster.Status == "creating" {
		time.Sleep(10 * time.Second)
		cluster, err = cli.Get(cluster.ID)
	}

	return err
}
```

### Swarm
Connect to a Docker Swarm cluster

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/getcarina/libcarina"
	"github.com/samalba/dockerclient"
)

func connectCluster(username string, apikey string, clusterID string) {
	// Connect to Carina
	cli, _ := libcarina.NewClusterClient(libcarina.BetaEndpoint, username, apikey)

	// Download the cluster credentials
	creds, _ := cli.GetCredentials(clusterID)

	// Get the IP of the host and the TLS configuration
	host, _ := creds.ParseHost()
	cfg, _ := creds.GetTLSConfig()

	// Do the Dockers!
	docker, _ := dockerclient.NewDockerClient(host, cfg)
	info, _ := docker.Info()
	fmt.Println(info)
}
```

### Kubernetes
Connect to a Kubernetes cluster

```go
package main

import (
	"fmt"
	"github.com/getcarina/libcarina"

	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/api"
)

func connectCluster(username string, apikey string, clusterID string) {
	// Connect to Carina
	cli, _ := libcarina.NewClusterClient(libcarina.BetaEndpoint, username, apikey)

	// Download the cluster credentials
	creds, _ := cli.GetCredentials(clusterID)

	// K8s stuff and things!
	k8cfg := &restclient.Config{
		Host:     creds.ParseHost(),
		CertData: creds.Cert,
		CAData:   creds.CA,
		KeyData:  creds.Key,
	}
	client, err := client.New(config)
	pods, err := client.Pods(api.NamespaceDefault).List(api.ListOptions{})
	
	return err
}
```
