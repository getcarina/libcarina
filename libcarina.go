package libcarina

import (
	"archive/zip"
	"bytes"
	"encoding/json"
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

// CarinaEndpoint is the public Carina API endpoint
const CarinaEndpoint = "https://api.getcarina.com"

// UserAgentPrefix is the default user agent string, consumers should append their application version to CarinaClient.UserAgent
const UserAgentPrefix = "getcarina/libcarina"

// CarinaClient accesses Carina directly
type CarinaClient struct {
	Client    *http.Client
	Username  string
	Token     string
	Endpoint  string
	UserAgent string
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

// NewClient create an authenticated CarinaClient
func NewClient(endpoint string, username string, apikey string, token string) (*CarinaClient, error) {

	verifyToken := func() error {
		req, err := http.NewRequest("HEAD", rackspace.RackspaceUSIdentity+"tokens/"+token, nil)
		if err != nil {
			return err
		}

		req.Header.Add("Accept", "application/json")
		req.Header.Add("X-Auth-Token", token)
		req.Header.Add("User-Agent", UserAgentPrefix)

		httpClient := &http.Client{}
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("Cached token is invalid")
		}

		return nil
	}

	// Attempt to authenticate with the cached token first, falling back on the apikey
	if token == "" || verifyToken() != nil {
		ao := &gophercloud.AuthOptions{
			Username:         username,
			APIKey:           apikey,
			IdentityEndpoint: rackspace.RackspaceUSIdentity,
		}

		provider, err := rackspace.AuthenticatedClient(*ao)
		if err != nil {
			return nil, err
		}
		token = provider.TokenID
	}

	return &CarinaClient{
		Client:    &http.Client{},
		Username:  username,
		Token:     token,
		Endpoint:  endpoint,
		UserAgent: UserAgentPrefix,
	}, nil
}

// NewRequest handles a request using auth used by Carina
func (c *CarinaClient) NewRequest(method string, uri string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.Endpoint+uri, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-Auth-Token", c.Token)
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Add("API-Version", "rax:container-infra "+SupportedAPIVersion)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		err := HTTPErr{
			Method:     req.Method,
			URL:        req.URL.String(),
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
func (c *CarinaClient) List() ([]*Cluster, error) {
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

	cluster := &Cluster{}
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

func (c *CarinaClient) lookupClusterName(token string) (string, error) {
	if !isClusterID(token) {
		return token, nil
	}

	clusters, err := c.List()
	if err != nil {
		return "", err
	}

	var name string
	for _, cluster := range clusters {
		if strings.ToLower(cluster.ID) == strings.ToLower(token) {
			name = cluster.Name
			break
		}
	}

	if name == "" {
		return "", HTTPErr{
			StatusCode: http.StatusNotFound,
			Status:     "404 NOT FOUND",
			Body:       `{"message": "Cluster "` + token + ` not found"}`}
	}

	return name, nil
}

func (c *CarinaClient) lookupClusterID(token string) (string, error) {
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

// ListClusterTypes returns a list of cluster types
func (c *CarinaClient) ListClusterTypes() ([]*ClusterType, error) {
	resp, err := c.NewRequest("GET", "/cluster_types", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Types []*ClusterType `json:"cluster_types"`
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Types, nil
}

// Get a cluster by cluster by its name or id
func (c *CarinaClient) Get(token string) (*Cluster, error) {
	id, err := c.lookupClusterID(token)
	if err != nil {
		return nil, err
	}

	uri := path.Join("/clusters", id)
	resp, err := c.NewRequest("GET", uri, nil)
	return clusterFromResponse(resp, err)
}

// Create a new cluster with cluster options
func (c *CarinaClient) Create(clusterOpts *CreateClusterOpts) (*Cluster, error) {
	clusterOptsJSON, err := json.Marshal(clusterOpts)
	if err != nil {
		return nil, err
	}

	body := bytes.NewReader(clusterOptsJSON)
	resp, err := c.NewRequest("POST", "/clusters", body)
	return clusterFromResponse(resp, err)
}

// GetCredentials returns a Credentials struct for the given cluster name
func (c *CarinaClient) GetCredentials(token string) (*CredentialsBundle, error) {
	id, err := c.lookupClusterID(token)
	if err != nil {
		return nil, err
	}

	name, err := c.lookupClusterName(token)
	if err != nil {
		return nil, err
	}

	url := c.Endpoint + path.Join("/clusters", id, "credentials/zip")
	zr, err := c.fetchZip(url)
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

	appendClusterName(name, creds)

	return creds, nil
}

// Set the CLUSTER_NAME environment variable in the scripts
func appendClusterName(name string, creds *CredentialsBundle) {
	addStmt := func(fileName string, stmt string) {
		script := creds.Files[fileName]
		script = append(script, []byte(stmt)...)
		creds.Files[fileName] = script
	}

	for fileName := range creds.Files {
		switch fileName {
		case "docker.env", "kubectl.env":
			addStmt(fileName, fmt.Sprintf("export CARINA_CLUSTER_NAME=%s\n", name))
		case "docker.fish", "kubectl.fish":
			addStmt(fileName, fmt.Sprintf("set -x CARINA_CLUSTER_NAME %s\n", name))
		case "docker.ps1", "kubectl.ps1":
			addStmt(fileName, fmt.Sprintf("$env:CARINA_CLUSTER_NAME=\"%s\"\n", name))
		case "docker.cmd", "kubectl.cmd":
			addStmt(fileName, fmt.Sprintf("set CARINA_CLUSTER_NAME=%s\n", name))
		}
	}
}

func (c *CarinaClient) fetchZip(zipurl string) (*zip.Reader, error) {
	req, err := http.NewRequest("GET", zipurl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		err := HTTPErr{
			Method:     req.Method,
			URL:        req.URL.String(),
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
		defer resp.Body.Close()
		b, _ := ioutil.ReadAll(resp.Body)
		err.Body = string(b)
		return nil, err
	}

	buf := &bytes.Buffer{}

	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, err
	}

	b := bytes.NewReader(buf.Bytes())
	return zip.NewReader(b, int64(b.Len()))
}

// Delete nukes a cluster out of existence
func (c *CarinaClient) Delete(token string) (*Cluster, error) {
	id, err := c.lookupClusterID(token)
	if err != nil {
		return nil, err
	}

	uri := path.Join("/clusters", id)
	resp, err := c.NewRequest("DELETE", uri, nil)
	return clusterFromResponse(resp, err)
}

// GetAPIMetadata returns metadata about the Carina API
func (c *CarinaClient) GetAPIMetadata() (*APIMetadata, error) {
	resp, err := c.NewRequest("GET", "/", nil)
	if err != nil {
		return nil, err
	}

	metadata := &APIMetadata{}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}
