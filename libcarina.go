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

	"github.com/pkg/errors"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/rackspace"
)

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

// CarinaGenericErrorResponse represents the response returned by Carina when a request fails
type CarinaGenericErrorResponse struct {
	Errors []CarinaError `json:"errors"`
}

// CarinaError represents an error message from the Carina API
type CarinaError struct {
	Code      string `json:"code"`
	Detail    string `json:"detail"`
	RequestID string `json:"request_id"`
	Status    int    `json:"status"`
	Title     string `json:"title"`
}

// CarinaUnacceptableErrorResonse represents the response returned by Carina when the StatusCode is 406
type CarinaUnacceptableErrorResonse struct {
	Errors []CarinaUnacceptableError `json:"errors"`
}

// CarinaUnacceptableError represents a 406 response from the Carina API
type CarinaUnacceptableError struct {
	CarinaError
	MaxVersion string `json:"max_version"`
	MinVersion string `json:"min_version"`
}

// genericError is a multi-purpose error formatter for generic errors from the Carina API
func (err HTTPErr) genericError() string {
	var carinaResp CarinaGenericErrorResponse

	jsonErr := json.Unmarshal([]byte(err.Body), &carinaResp)
	if jsonErr != nil {
		return fmt.Sprintf("%s %s (%d)", err.Method, err.URL, err.StatusCode)
	}

	var errorMessages bytes.Buffer
	for _, carinaErr := range carinaResp.Errors {
		errorMessages.WriteString("\nMessage: ")
		errorMessages.WriteString(carinaErr.Title)
		errorMessages.WriteString(" - ")
		errorMessages.WriteString(carinaErr.Detail)
	}
	return fmt.Sprintf("%s %s (%d)%s", err.Method, err.URL, err.StatusCode, errorMessages.String())
}

// unacceptableError is a error formatter for parsing a 406 response from the Carina API
func (err HTTPErr) unacceptableError() string {
	var carinaResp CarinaUnacceptableErrorResonse

	jsonErr := json.Unmarshal([]byte(err.Body), &carinaResp)
	if jsonErr != nil {
		return err.genericError()
	}

	var errorMessages bytes.Buffer
	for _, carinaErr := range carinaResp.Errors {
		errorMessages.WriteString("\nMessage: ")
		errorMessages.WriteString(carinaErr.Title)
		errorMessages.WriteString(" - The client supports ")
		errorMessages.WriteString(SupportedAPIVersion)
		errorMessages.WriteString(" while the server supports ")
		errorMessages.WriteString(carinaErr.MinVersion)
		errorMessages.WriteString(" - ")
		errorMessages.WriteString(carinaErr.MaxVersion)
		errorMessages.WriteString(".")
	}
	return fmt.Sprintf("%s %s (%d)%s", err.Method, err.URL, err.StatusCode, errorMessages.String())
}

// Error routes to either genericError or other, more-specific, response formatters to give provide a user-friendly error
func (err HTTPErr) Error() string {
	switch err.StatusCode {
	default:
		return err.genericError()
	case 406:
		return err.unacceptableError()
	}
}

// NewClient create an authenticated CarinaClient
func NewClient(username string, apikey string, region string, authEndpointOverride string, cachedToken string, cachedEndpoint string) (*CarinaClient, error) {
	authEndpoint := rackspace.RackspaceUSIdentity
	if authEndpointOverride != "" {
		authEndpoint = authEndpointOverride
	}

	verifyToken := func() error {
		req, err := http.NewRequest("HEAD", authEndpoint+"tokens/"+cachedToken, nil)
		if err != nil {
			return errors.WithStack(err)
		}

		req.Header.Add("Accept", "application/json")
		req.Header.Add("X-Auth-Token", cachedToken)
		req.Header.Add("User-Agent", UserAgentPrefix)

		httpClient := &http.Client{}
		resp, err := httpClient.Do(req)
		if err != nil {
			return errors.WithStack(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("Cached token is invalid")
		}

		return nil
	}

	// Attempt to authenticate with the cached token first, falling back on the apikey
	if cachedToken == "" || verifyToken() != nil {
		ao := &gophercloud.AuthOptions{
			Username:         username,
			APIKey:           apikey,
			IdentityEndpoint: authEndpoint,
		}

		provider, err := rackspace.AuthenticatedClient(*ao)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		cachedToken = provider.TokenID

		eo := gophercloud.EndpointOpts{Region: region}
		eo.ApplyDefaults(CarinaEndpointType)
		url, err := provider.EndpointLocator(eo)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		cachedEndpoint = strings.TrimRight(url, "/")
	}

	return &CarinaClient{
		Client:    &http.Client{},
		Username:  username,
		Token:     cachedToken,
		Endpoint:  cachedEndpoint,
		UserAgent: UserAgentPrefix,
	}, nil
}

// NewRequest handles a request using auth used by Carina
func (c *CarinaClient) NewRequest(method string, uri string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.Endpoint+uri, body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-Auth-Token", c.Token)
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Add("API-Version", CarinaEndpointType+" "+SupportedAPIVersion)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
	}

	return result.Clusters, nil
}

func clusterFromResponse(resp *http.Response, err error) (*Cluster, error) {
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cluster := &Cluster{}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&cluster)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cluster, nil
}

func isClusterID(token string) bool {
	r := regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")
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
		return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
	}

	body := bytes.NewReader(clusterOptsJSON)
	resp, err := c.NewRequest("POST", "/clusters", body)
	return clusterFromResponse(resp, err)
}

// Resize a cluster with resize task options
func (c *CarinaClient) Resize(token string, nodes int) (*Cluster, error) {
	id, err := c.lookupClusterID(token)
	if err != nil {
		return nil, err
	}

	resizeOpts := newResizeOpts(nodes)
	resizeOptsJSON, err := json.Marshal(resizeOpts)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	body := bytes.NewReader(resizeOptsJSON)
	uri := path.Join("/clusters", id, "tasks")
	resp, err := c.NewRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.Get(token)
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

	uri := path.Join("/clusters", id, "credentials/zip")
	resp, err := c.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	// Read the body as a zip file
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	b := bytes.NewReader(buf.Bytes())
	zipr, err := zip.NewReader(b, int64(b.Len()))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Fetch the contents for each file in the zipfile
	creds := NewCredentialsBundle()
	for _, zf := range zipr.File {
		_, fname := path.Split(zf.Name)
		fi := zf.FileInfo()

		if fi.IsDir() {
			// Explicitly skip past directories (the UUID directory from a previous release)
			continue
		}

		rc, err := zf.Open()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		b, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
	}

	return metadata, nil
}
