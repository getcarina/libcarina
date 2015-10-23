# libcarina

[![GoDoc](https://godoc.org/github.com/rackerlabs/libcarina?status.png)](https://godoc.org/github.com/rackerlabs/libcarina)

Provisional Go bindings for the beta release of [Carina](https://getcarina.com) by Rackspace. If you're looking for a client binding, [`carina`](https://github.com/rackerlabs/carina) is your friend.

![](https://cloud.githubusercontent.com/assets/836375/10503963/e5bcca8c-72c0-11e5-8e14-2c1697297d7e.png)

## Examples

Getting straight to Docker with only your username and API Key:

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/getcarina/libcarina"
	"github.com/samalba/dockerclient"
)

func main() {
	var err error

	username := os.Args[1]
	apiKey := os.Args[2]
	clusterName := os.Args[3]

	// Connect to Carina
	cli, _ := libcarina.NewClusterClient(libcarina.BetaEndpoint, username, apiKey)

	// Create a new cluster
	cluster, _ := cli.Create(libcarina.Cluster{ClusterName: clusterName})

	// Wait for it to come up...
	for cluster.Status == "new" || cluster.Status == "building" {
		time.Sleep(10 * time.Second)
		cluster, err = cli.Get(clusterName)
		if err != nil {
			break
		}
	}

	// Get the IP of the host and a *tls.Config
	host, tlsConfig, _ := cli.GetDockerConfig(clusterName)

	// Straight to Docker, do what you need
	docker, _ := dockerclient.NewDockerClient(host, tlsConfig)
	info, _ := docker.Info()
	fmt.Println(info)

}
```
