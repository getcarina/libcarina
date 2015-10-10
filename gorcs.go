package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
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

// ZipURLResponse is the response that comes back from the zip endpoint
type ZipURLResponse struct {
	URL string `json:"zip_url"`
}

// ClusterClient accesses RCS
type ClusterClient struct {
	client   *http.Client
	Username string
	Token    string
	Endpoint string
}

// ErrorResponse is the JSON formatted error response from RCS
type ErrorResponse struct {
	Error string `json:"error"`
}

// Cluster is a cluster
type Cluster struct {
	ClusterName string `json:"cluster_name"`
	Username    string `json:"username"`

	// Flavor of compute to use for cluster, should be a default value currently
	Flavor string `json:"flavor,omitempty"`

	// UUID of image to use for cluster, should be a default value currently
	Image string `json:"image,omitempty"`

	// Node is optional, but allowed on create
	Nodes json.Number `json:"nodes,omitempty"`

	AutoScale bool   `json:"autoscale,omitempty"`
	Status    string `json:"status,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Token     string `json:"token,omitempty"`
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

// NewRequest handles a request using auth used by RCS
func (c *ClusterClient) NewRequest(method string, uri string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, BetaEndpoint+uri, body)
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

	return resp, nil
}

// List the current clusters
func (c *ClusterClient) List() ([]Cluster, error) {
	clusters := []Cluster{}

	resp, err := c.NewRequest("GET", "/clusters/"+c.Username, nil)
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&clusters)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func clusterFromResponse(resp *http.Response, err error) (*Cluster, error) {
	if err != nil {
		return nil, err
	}
	cluster := new(Cluster)
	err = json.NewDecoder(resp.Body).Decode(&cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// Get a cluster by cluster name
func (c *ClusterClient) Get(clusterName string) (*Cluster, error) {
	uri := path.Join("/clusters", c.Username, clusterName)
	resp, err := c.NewRequest("GET", uri, nil)
	return clusterFromResponse(resp, err)
}

// Create a new cluster with cluster options
func (c *ClusterClient) Create(clusterOpts Cluster) (*Cluster, error) {
	clusterOptsJSON, err := json.Marshal(clusterOpts)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(clusterOptsJSON))

	body := bytes.NewReader(clusterOptsJSON)
	uri := path.Join("/clusters", c.Username)
	resp, err := c.NewRequest("POST", uri, body)
	return clusterFromResponse(resp, err)
}

// ZipURL returns the URL for downloading credentials
func (c *ClusterClient) ZipURL(clusterName string) (string, error) {
	uri := path.Join("/clusters", c.Username, clusterName, "zip")
	resp, err := c.NewRequest("GET", uri, nil)
	if err != nil {
		return "", err
	}

	var zipURLResp ZipURLResponse

	err = json.NewDecoder(resp.Body).Decode(&zipURLResp)

	if err != nil {
		return "", err
	}

	return zipURLResp.URL, nil
}

// Grow increases a cluster by the provided number of nodes
func (c *ClusterClient) Grow(clusterName string, nodes int) (*Cluster, error) {
	incr := make(map[string]json.Number)
	incr["nodes"] = json.Number(nodes)
	growthRequest, err := json.Marshal(incr)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(growthRequest)

	uri := path.Join("/clusters", c.Username, clusterName, "grow")
	resp, err := c.NewRequest("POST", uri, r)
	return clusterFromResponse(resp, err)
}

// Delete nukes a cluster out of existence
func (c *ClusterClient) Delete(clusterName string) (*Cluster, error) {
	uri := path.Join("/clusters", c.Username, clusterName)
	resp, err := c.NewRequest("DELETE", uri, nil)
	return clusterFromResponse(resp, err)
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

	//c, err := clusterClient.ZipURL(l[0].ClusterName)

	clusterOpts := Cluster{
		Username:    clusterClient.Username,
		ClusterName: "WHOA",
		Nodes:       json.Number(5),
	}

	c, err := clusterClient.Create(clusterOpts)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(c)

}
