package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/tabwriter"

	rcs "github.com/rgbkrk/gorcs"
	"github.com/samalba/dockerclient"
)

func dockerInfo(creds *rcs.Credentials) (*dockerclient.Info, error) {
	tlsConfig, err := creds.GetTLSConfig()
	if err != nil {
		return nil, err
	}

	docker, err := dockerclient.NewDockerClient(creds.DockerHost, tlsConfig)
	if err != nil {
		return nil, err
	}
	info, err := docker.Info()
	return info, err
}

func writeCredentials(w *tabwriter.Writer, creds *rcs.Credentials, pth string) (err error) {
	statusFormat := "%s\t%s\n"
	for fname, b := range creds.Files {
		p := path.Join(pth, fname)
		err = ioutil.WriteFile(p, b, 0600)
		if err != nil {
			fmt.Fprintf(w, statusFormat, fname, "🚫")
			return err
		}
		fmt.Fprintf(w, statusFormat, fname, "✅")
	}
	return nil
}

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
		w.Flush()
		os.Exit(3)
	}

	switch command {
	case "list":
		var clusters []rcs.Cluster
		clusters, err = clusterClient.List()
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
	case "create":
		c := rcs.Cluster{
			ClusterName: clusterName,
		}
		cluster, err := clusterClient.Create(c)
		writeCluster(w, cluster, err)
	case "credentials":
		creds, err := clusterClient.GetCredentials(clusterName)
		if err == nil {
			err = writeCredentials(w, creds, ".")
		}

		// Snuck in as an example
	case "docker-info":
		creds, err := clusterClient.GetCredentials(clusterName)
		if err != nil {
			break
		}
		info, err := dockerInfo(creds)
		fmt.Fprintf(w, "%+v\n", info)
	default:
		usage()
		err = errors.New("command " + command + " not recognized")
	}
	exitCode := 0

	if err != nil {
		simpleErr(w, err)
		exitCode = 4
	}

	w.Flush()
	os.Exit(exitCode)
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
	fmt.Fprintf(w, "ERROR: %v\n", err)
}

func usage() {
	exe := os.Args[0]

	fmt.Printf("NAME:\n")
	fmt.Printf("  %s - command line interface to manage swarm clusters\n", exe)
	fmt.Printf("USAGE:\n")
	fmt.Printf("  %s <command> [clustername] [-username <username>] [-password <password>] [-endpoint <endpoint>]\n", exe)
	fmt.Println()
	fmt.Printf("COMMANDS:\n")
	fmt.Printf("  %s list\n", exe)
	fmt.Printf("  %s create <clustername>      - create a new cluster\n", exe)
	fmt.Printf("  %s get <clustername>         - get a cluster by name\n", exe)
	fmt.Printf("  %s delete <clustername>      - delete a cluster by name\n", exe)
	fmt.Printf("  %s credentials <clustername> - download credentials to the current directory\n", exe)
	fmt.Println()
	fmt.Printf("FLAGS:\n")
	fmt.Printf("  -endpoint string\n")
	fmt.Printf("    RCS API Endpoint (default \"https://mycluster.rackspacecloud.com\")\n")
	fmt.Printf("  -password string\n")
	fmt.Printf("    Rackspace password\n")
	fmt.Printf("  -username string\n")
	fmt.Printf("    Rackspace username\n")
	fmt.Println()
	fmt.Printf("ENVIRONMENT VARIABLES:\n")
	fmt.Printf("  RACKSPACE_USERNAME - set instead of -username\n")
	fmt.Printf("  RACKSPACE_PASSWORD - set instead of -password\n")
}
