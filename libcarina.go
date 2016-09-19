package libcarina

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/rackspace"
)

// BetaEndpoint reflects the default endpoint for this library
const BetaEndpoint = "https://app.getcarina.com"
const mimetypeJSON = "application/json"
const userAgent = "getcarina/libcarina"

// ClusterClient accesses Carina directly
type ClusterClient struct {
	Client   *http.Client
	Username string
	Token    string
	Endpoint string
}

// HTTPErr is returned when API requests are not successful
type HTTPErr struct {
	Method     string
	URL        string
	StatusCode int
	Status     string
	Body       string
}

func (err HTTPErr) Error() string {
	return fmt.Sprintf("%s %s (%d-%s)", err.Method, err.URL, err.StatusCode, err.Status)
}

// Cluster is a cluster of Docker nodes
type Cluster struct {
	// ID of the cluster
	ID string `json:"id"`

	// Name of the cluster
	Name string `json:"name"`

	// COE (container orchestration engine) used by the cluster
	COE string `json:"coe"`

	// Underlying type of the host nodes, such as lxc or vm
	HostType string `json:"host_type"`

	// Nodes in the cluster
	Nodes int `json:"node_count,omitempty"`

	// Status of the cluster
	Status string `json:"status,omitempty"`
}

// Quotas is the set of account quotas
type Quotas struct {
	MaxClusters        int `json:"max_clusters"`
	MaxNodesPerCluster int `json:"max_nodes_per_cluster"`
}

func newClusterClient(endpoint string, ao *gophercloud.AuthOptions) (*ClusterClient, error) {
	provider, err := rackspace.AuthenticatedClient(*ao)
	if err != nil {
		return nil, err
	}

	return &ClusterClient{
		Client:   &http.Client{},
		Username: ao.Username,
		Token:    provider.TokenID,
		Endpoint: endpoint,
	}, nil
}

// NewClusterClient create a new clusterclient by API Key
func NewClusterClient(endpoint string, username string, apikey string) (*ClusterClient, error) {
	ao := &gophercloud.AuthOptions{
		Username:         username,
		APIKey:           apikey,
		IdentityEndpoint: rackspace.RackspaceUSIdentity,
	}

	return newClusterClient(endpoint, ao)
}

// NewRequest handles a request using auth used by Carina
func (c *ClusterClient) NewRequest(method string, uri string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.Endpoint+uri, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Add("Content-Type", mimetypeJSON)
	req.Header.Add("Accept", mimetypeJSON)
	req.Header.Add("X-Auth-Token", c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		err := HTTPErr{
			Method:     method,
			URL:        uri,
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
		defer resp.Body.Close()
		b, _ := ioutil.ReadAll(resp.Body)
		err.Body = string(b)
		return nil, err
	}

	return resp, nil
}

// List the current clusters
func (c *ClusterClient) List() ([]*Cluster, error) {
	resp, err := c.NewRequest("GET", "/clusters", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Clusters []*Cluster `json:"clusters"`
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Clusters, nil
}

func clusterFromResponse(resp *http.Response, err error) (*Cluster, error) {
	if err != nil {
		return nil, err
	}

	cluster := new(Cluster)
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func isClusterID(token string) bool {
	r := regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[8|9|aA|bB][a-f0-9]{3}-[a-f0-9]{12}$")
	return r.MatchString(token)
}

func (c *ClusterClient) lookupClusterID(token string) (string, error) {
	if isClusterID(token) {
		return token, nil
	}

	clusters, err := c.List()
	if err != nil {
		return "", err
	}

	var id string
	for _, cluster := range clusters {
		if strings.ToLower(cluster.Name) == strings.ToLower(token) {
			if id != "" {
				return "", fmt.Errorf("The cluster (%s) is not unique. Retry the request using the cluster id", token)
			}
			id = cluster.ID
		}
	}

	if id == "" {
		return "", HTTPErr{
			StatusCode: http.StatusNotFound,
			Status:     "404 NOT FOUND",
			Body:       `{"message": "Cluster "` + token + ` not found"}`}
	}

	return id, nil
}

// Get a cluster by cluster by its name or id
func (c *ClusterClient) Get(token string) (*Cluster, error) {
	id, err := c.lookupClusterID(token)
	if err != nil {
		return nil, err
	}

	uri := path.Join("/clusters", id)
	resp, err := c.NewRequest("GET", uri, nil)
	return clusterFromResponse(resp, err)
}

// Create a new cluster with cluster options
func (c *ClusterClient) Create(clusterOpts *Cluster) (*Cluster, error) {
	clusterOptsJSON, err := json.Marshal(clusterOpts)
	if err != nil {
		return nil, err
	}

	body := bytes.NewReader(clusterOptsJSON)
	resp, err := c.NewRequest("POST", "/clusters", body)
	return clusterFromResponse(resp, err)
}

// GetCredentials returns a Credentials struct for the given cluster name
func (c *ClusterClient) GetCredentials(token string) (*CredentialsBundle, error) {
	id, err := c.lookupClusterID(token)
	if err != nil {
		return nil, err
	}

	url := c.Endpoint + path.Join("/clusters", id, "credentials/zip")
	zr, err := fetchZip(c.Client, url)
	if err != nil {
		return nil, err
	}

	// fetch the contents for each file
	creds := NewCredentialsBundle()
	for _, zf := range zr.File {
		_, fname := path.Split(zf.Name)
		fi := zf.FileInfo()

		if fi.IsDir() {
			// Explicitly skip past directories (the UUID directory from a previous release)
			continue
		}

		rc, err := zf.Open()
		if err != nil {
			return nil, err
		}

		b, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		creds.Files[fname] = b
	}

	return creds, nil
}

func fetchZip(client *http.Client, zipurl string) (*zip.Reader, error) {
	req, err := http.NewRequest("GET", zipurl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
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

	buf := &bytes.Buffer{}

	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, err
	}

	b := bytes.NewReader(buf.Bytes())
	return zip.NewReader(b, int64(b.Len()))
}

// Grow increases a cluster by the provided number of nodes
func (c *ClusterClient) Grow(clusterName string, nodes int) (*Cluster, error) {
	incr := map[string]int{
		"nodes": nodes,
	}

	growthRequest, err := json.Marshal(incr)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(growthRequest)

	uri := path.Join("/clusters", c.Username, clusterName, "grow")
	resp, err := c.NewRequest("POST", uri, r)
	return clusterFromResponse(resp, err)
}

// SetAutoScale enables or disables autoscale on an already running cluster
func (c *ClusterClient) SetAutoScale(clusterName string, autoscale bool) (*Cluster, error) {
	setAutoscale := "false"
	if autoscale {
		setAutoscale = "true"
	}
	uri := path.Join("/clusters", c.Username, clusterName, "autoscale", setAutoscale)
	resp, err := c.NewRequest("PUT", uri, nil)
	return clusterFromResponse(resp, err)
}

const rebuildSwarmAction = "rebuild-swarm"

type actionRequest struct {
	Action string `json:"action"`
}

func (c *ClusterClient) doAction(clusterName, action string) (*Cluster, error) {
	actionReq, err := json.Marshal(actionRequest{Action: action})
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(actionReq)
	uri := path.Join("/clusters", c.Username, clusterName, "action")
	resp, err := c.NewRequest("POST", uri, r)
	return clusterFromResponse(resp, err)
}

// Rebuild creates a wholly new Swarm cluster
func (c *ClusterClient) Rebuild(clusterName string) (*Cluster, error) {
	return c.doAction(clusterName, rebuildSwarmAction)
}

// Delete nukes a cluster out of existence
func (c *ClusterClient) Delete(token string) (*Cluster, error) {
	id, err := c.lookupClusterID(token)
	if err != nil {
		return nil, err
	}

	uri := path.Join("/clusters", id)
	resp, err := c.NewRequest("DELETE", uri, nil)
	return clusterFromResponse(resp, err)
}

func quotasFromResponse(resp *http.Response) (*Quotas, error) {
	quotas := new(Quotas)
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(&quotas)
	if err != nil {
		return nil, err
	}
	return quotas, nil
}

// GetQuotas returns the account's quotas
func (c *ClusterClient) GetQuotas() (*Quotas, error) {
	uri := path.Join("/quotas", c.Username)
	resp, err := c.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	return quotasFromResponse(resp)
}
