package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// BetaEndpoint reflects the default endpoint for this library
const BetaEndpoint = "https://mycluster.rackspacecloud.com"
const mimetypeJSON = "application/json"
const authHeaderKey = "X-Auth-Token"

// UserAuth setup
type UserAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse from user authentication
type AuthResponse struct {
	Token string `json:"token"`
}

// ClusterClient accesses RCS
type ClusterClient struct {
	client   *http.Client
	Username string
	Token    string
	Endpoint string
}

// Cluster is a cluster
type Cluster struct {
	AutoScale   bool        `json:"autoscale"`
	ClusterName string      `json:"cluster_name"`
	Flavor      string      `json:"flavor"`
	Image       string      `json:"image"`
	Nodes       json.Number `json:"nodes"`
	Status      string      `json:"status"`
	TaskID      string      `json:"task_id"`
	Token       string      `json:"token"`
	Username    string      `json:"username"`
}

// NewClusterClient creates a new ClusterClient
func NewClusterClient(endpoint, username, password string) (*ClusterClient, error) {
	userAuth := UserAuth{
		Username: username,
		Password: password,
	}

	client := &http.Client{}

	b, err := json.Marshal(userAuth)
	if err != nil {
		return nil, err
	}
	data := bytes.NewBuffer(b)

	req, err := http.NewRequest("POST", BetaEndpoint+"/auth", data)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", mimetypeJSON)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New(resp.Status)
		}
		return nil, errors.New(string(b))
	}

	var authResponse AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResponse)
	if err != nil {
		return nil, err
	}

	token := authResponse.Token

	return &ClusterClient{
		client:   client,
		Username: username,
		Token:    token,
	}, nil
}

// List the current clusteres
func (c *ClusterClient) List() ([]Cluster, error) {
	clusters := []Cluster{}

	req, err := http.NewRequest("GET", BetaEndpoint+"/clusters/"+c.Username, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", mimetypeJSON)
	req.Header.Add(authHeaderKey, c.Token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New(resp.Status)
		}
		return nil, errors.New(string(b))
	}

	err = json.NewDecoder(resp.Body).Decode(&clusters)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func main() {
	username := os.Getenv("RACKSPACE_USERNAME")
	password := os.Getenv("RACKSPACE_PASSWORD")

	if username == "" || password == "" {
		fmt.Println("Need the RACKSPACE_USERNAME and RACKSPACE_PASSWORD environment variables set.")
		os.Exit(1)
	}

	endpoint := BetaEndpoint

	clusterClient, err := NewClusterClient(endpoint, username, password)
	if err != nil {
		panic(err)
	}

	l, err := clusterClient.List()
	if err != nil {
		panic(err)
	}

	fmt.Println(l)

}
