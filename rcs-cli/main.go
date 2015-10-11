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
	var username, password, endpoint string

	flag.Usage = usage

	flag.StringVar(&username, "username", "", "Rackspace username")
	flag.StringVar(&password, "password", "", "Rackspace password")
	flag.StringVar(&endpoint, "endpoint", rcs.BetaEndpoint, "RCS API Endpoint")
	flag.Parse()

	if username == "" && os.Getenv("RACKSPACE_USERNAME") != "" {
		username = os.Getenv("RACKSPACE_USERNAME")
	}
	if password == "" && os.Getenv("RACKSPACE_PASSWORD") != "" {
		password = os.Getenv("RACKSPACE_PASSWORD")
	}

	if username == "" || password == "" {
		fmt.Println("Either set --username and --password or set the " +
			"RACKSPACE_USERNAME and RACKSPACE_PASSWORD environment variables.")
		fmt.Println()
		usage()
		os.Exit(1)
	}

	var command, clusterName string

	command = flag.Arg(0)
	clusterName = flag.Arg(1)

	switch {
	case flag.NArg() < 1 || (command != "list" && flag.NArg() < 2):
		usage()
		os.Exit(2)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)

	clusterClient, err := rcs.NewClusterClient(endpoint, username, password)
	if err != nil {
		simpleErr(w, err)
		os.Exit(3)
	}

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

		// credentials is the least thought through bit of this code
		// pretend you never saw this
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
		simpleErr(w, err)
		os.Exit(4)
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

func simpleErr(w *tabwriter.Writer, err error) {
	fmt.Fprintf(w, "ERROR: %v", err)
	w.Flush()
}

func usage() {
	fmt.Println("NAME:")
	fmt.Println("  rcs-cli - command line interface to manage swarm clusters")
	fmt.Println("USAGE:")
	fmt.Println("  rcs-cli <command> [clustername] [-username <username>] [-password <password>] [-endpoint <endpoint>]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  rcs-cli list")
	fmt.Println("  rcs-cli create <clustername>")
	fmt.Println("  rcs-cli get <clustername>")
	fmt.Println("  rcs-cli delete <clustername>")
	fmt.Println("  rcs-cli zipurl <clustername>")
	fmt.Println()
	fmt.Println("FLAGS:")
	fmt.Println("  -endpoint string")
	fmt.Println("    RCS API Endpoint (default \"https://mycluster.rackspacecloud.com\")")
	fmt.Println("  -password string")
	fmt.Println("    Rackspace password")
	fmt.Println("  -username string")
	fmt.Println("    Rackspace username")
	fmt.Println()
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("  RACKSPACE_USERNAME - set instead of -username")
	fmt.Println("  RACKSPACE_PASSWORD - set instead of -password")
}
