package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

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

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	var clusterName string
	command := flag.Arg(0)
	if command != "list" {
		clusterName = flag.Arg(1)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	switch command {
	case "list":
		clusters, err := clusterClient.List()
		if err == nil {
			for _, cluster := range clusters {
				writeCluster(w, &cluster, err)
			}
		}
	case "get":
		cluster, err := clusterClient.Get(clusterName)
		writeCluster(w, cluster, err)
	case "delete":
		cluster, err := clusterClient.Delete(clusterName)
		writeCluster(w, cluster, err)
	case "zipurl":
		zipurl, err := clusterClient.GetZipURL(clusterName)
		if err != nil {
			w.Write([]byte(zipurl))
		}
	case "create":
		c := rcs.Cluster{
			ClusterName: clusterName,
		}
		cluster, err := clusterClient.Create(c)
		writeCluster(w, cluster, err)
	case "credentials":
		creds, err := clusterClient.GetCredentials(clusterName)
		if err != nil {
			fmt.Println(creds)
		}
	default:
		usage()
		err = errors.New("command " + command + " not recognized")
	}
	w.Flush()

	if err != nil {
		fmt.Fprintf(w, "ERROR: %v", err)
		w.Flush()
		os.Exit(2)
	}
}

func writeCluster(w *tabwriter.Writer, cluster *rcs.Cluster, err error) {
	if err != nil {
		return
	}
	s := strings.Join([]string{cluster.ClusterName,
		cluster.Username,
		cluster.Flavor,
		cluster.Image,
		fmt.Sprintf("%v", cluster.Nodes),
		cluster.Status}, "\t")
	w.Write([]byte(s + "\n"))
}

func usage() {
	fmt.Println("NAME:")
	fmt.Println("  rcs-cli - command line interface to manage swarm clusters")
	fmt.Println("USAGE:")
	fmt.Println("  rcs-cli <command> [clustername]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  rcs-cli list")
	fmt.Println("  rcs-cli create clustername")
	fmt.Println("  rcs-cli get clustername")
	fmt.Println("  rcs-cli delete clustername")
	fmt.Println("  rcs-cli zipurl clustername")
}
