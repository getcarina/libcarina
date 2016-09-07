package libcarina

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"fmt"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/rackspace"
	"regexp"
)

// BetaEndpoint reflects the default endpoint for this library
const BetaEndpoint = "https://app.getcarina.com"
const mimetypeJSON = "application/json"
const authHeaderKey = "X-Auth-Token"
const userAgent = "getcarina/libcarina"

// ZipURLResponse is the response that comes back from the zip endpoint
type ZipURLResponse struct {
	URL string `json:"zip_url"`
}

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
	Status    string `json:"status,omitempty"`
}

// Credentials holds the keys to the kingdom
type Credentials struct {
	README     []byte
	Cert       []byte
	Key        []byte
	CA         []byte
	CAKey      []byte
	DockerEnv  []byte
	DockerCmd  []byte
	DockerPS1  []byte
	DockerHost string
	Files      map[string][]byte
	DockerFish []byte
}

// Quotas is the set of account quotas
type Quotas struct {
	MaxClusters        int `json:"max_clusters"`
	MaxNodesPerCluster int `json:"max_nodes_per_cluster"`
}

func newClusterClient(endpoint string, ao gophercloud.AuthOptions) (*ClusterClient, error) {
	provider, err := rackspace.AuthenticatedClient(ao)
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
func NewClusterClient(endpoint, username, apikey string) (*ClusterClient, error) {
	ao := gophercloud.AuthOptions{
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
	req.Header.Add(authHeaderKey, c.Token)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		err := HTTPErr{
			Method:     method,
			URL:        uri,
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
		if resp.Body == nil {
			body, _ := ioutil.ReadAll(resp.Body)
			err.Body = string(body)
		}
		return nil, err
	}

	return resp, nil
}

// List the current clusters
func (c *ClusterClient) List() ([]Cluster, error) {
	resp, err := c.NewRequest("GET", "/clusters", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Clusters []Cluster `json:"clusters"`
	}
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

	for _, cluster := range clusters {
		if strings.ToLower(cluster.Name) == strings.ToLower(token) {
			return cluster.ID, nil
		}
	}

	return "", fmt.Errorf("Could not find cluster named: %s", token)
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
func (c *ClusterClient) Create(clusterOpts Cluster) (*Cluster, error) {
	clusterOptsJSON, err := json.Marshal(clusterOpts)
	if err != nil {
		return nil, err
	}

	body := bytes.NewReader(clusterOptsJSON)
	resp, err := c.NewRequest("POST", "/clusters", body)
	return clusterFromResponse(resp, err)
}

// GetZipURL returns the URL for downloading credentials
func (c *ClusterClient) GetZipURL(clusterName string) (string, error) {
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

// GetCredentials returns a Credentials struct for the given cluster name
func (c *ClusterClient) GetCredentials(clusterName string) (*Credentials, error) {
	url, err := c.GetZipURL(clusterName)
	if err != nil {
		return nil, err
	}
	zr, err := fetchZip(c.Client, url)
	if err != nil || len(zr.File) < 6 {
		return nil, err
	}

	// fetch the contents for each credential/note
	creds := new(Credentials)
	creds.Files = make(map[string][]byte)
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

		switch fname {
		case "ca.pem":
			creds.CA = b
		case "README.md":
			creds.README = b
		case "ca-key.pem":
			creds.CAKey = b
		case "docker.env":
			creds.DockerEnv = b
		case "cert.pem":
			creds.Cert = b
		case "key.pem":
			creds.Key = b
		case "docker.ps1":
			creds.DockerPS1 = b
		case "docker.cmd":
			creds.DockerCmd = b
		case "docker.fish":
			creds.DockerFish = b
		}

	}

	sourceLines := strings.Split(string(creds.DockerEnv), "\n")
	for _, line := range sourceLines {
		if strings.Index(line, "export ") == 0 {
			varDecl := strings.TrimRight(line[7:], "\n")
			eqLocation := strings.Index(varDecl, "=")

			varName := varDecl[:eqLocation]
			varValue := varDecl[eqLocation+1:]

			switch varName {
			case "DOCKER_HOST":
				creds.DockerHost = varValue
			}

		}
	}

	return creds, nil
}

// GetDockerConfig returns the hostname and tls.Config for a given clustername
func (c *ClusterClient) GetDockerConfig(clusterName string) (hostname string, tlsConfig *tls.Config, err error) {
	creds, err := c.GetCredentials(clusterName)
	if err != nil {
		return "", nil, err
	}
	tlsConfig, err = creds.GetTLSConfig()
	return creds.DockerHost, tlsConfig, err
}

// GetTLSConfig returns a tls.Config for a credential set
func (creds *Credentials) GetTLSConfig() (*tls.Config, error) {
	// TLS config
	var tlsConfig tls.Config
	tlsConfig.InsecureSkipVerify = true
	certPool := x509.NewCertPool()

	certPool.AppendCertsFromPEM(creds.CA)
	tlsConfig.RootCAs = certPool
	keypair, err := tls.X509KeyPair(creds.Cert, creds.Key)
	if err != nil {
		return &tlsConfig, err
	}
	tlsConfig.Certificates = []tls.Certificate{keypair}

	return &tlsConfig, nil
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
	defer resp.Body.Close()

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
