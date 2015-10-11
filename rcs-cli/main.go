package main

import (
	"flag"
	"fmt"
	"os"

	rcs "github.com/rgbkrk/gorcs"
)

func main() {
	username := os.Getenv("RACKSPACE_USERNAME")
	password := os.Getenv("RACKSPACE_PASSWORD")

	if username == "" || password == "" {
		fmt.Println("Need the RACKSPACE_USERNAME and RACKSPACE_PASSWORD environment variables set.")
		os.Exit(1)
	}

	endpoint := rcs.BetaEndpoint

	clusterClient, err := rcs.NewClusterClient(endpoint, username, password)
	if err != nil {
		panic(err)
	}

	flag.Parse()

	command := flag.Arg(0)
	clusterName := flag.Arg(1)

	var i interface{}

	switch command {
	case "list":
		i, err = clusterClient.List()
	case "get":
		i, err = clusterClient.Get(clusterName)
	case "delete":
		i, err = clusterClient.Delete(clusterName)
	case "zipurl":
		i, err = clusterClient.ZipURL(clusterName)
	case "create":
		c := rcs.Cluster{
			ClusterName: clusterName,
		}
		i, err = clusterClient.Create(c)
	case "credentials":
		i, err = clusterClient.GetCredentials(clusterName)
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Println(i)
}
