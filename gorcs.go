package rcs

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
	"reflect"
	"strconv"
	"strings"
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
	// Sadly it comes back as string instead of int in all cases
	// with the API
	Nodes number `json:"nodes,omitempty"`

	AutoScale bool   `json:"autoscale,omitempty"`
	Status    string `json:"status,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Token     string `json:"token,omitempty"`
}

// Specify this type for any struct fields that
// might be unmarshaled from JSON numbers of the following
// types: floats, integers, scientific notation, or strings
type number float64

func (n number) Int64() int64 {
	return int64(n)
}

func (n number) Int() int {
	return int(n)
}

func (n number) Float64() float64 {
	return float64(n)
}

// Required to enforce that string values are attempted to be parsed as numbers
func (n *number) UnmarshalJSON(data []byte) error {
	var f float64
	var err error
	if data[0] == '"' {
		f, err = strconv.ParseFloat(string(data[1:len(data)-1]), 64)
		if err != nil {
			return &json.UnmarshalTypeError{
				Value: string(data),
				Type:  reflect.TypeOf(*n),
			}
		}
	} else {
		if err := json.Unmarshal(data, &f); err != nil {
			return &json.UnmarshalTypeError{
				Value: string(data),
				Type:  reflect.TypeOf(*n),
			}
		}
	}
	*n = number(f)
	return nil
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
	// Even though username is in the URI path, the API expects the username
	// inside the body
	if clusterOpts.Username == "" {
		clusterOpts.Username = c.Username
	}
	clusterOptsJSON, err := json.Marshal(clusterOpts)
	if err != nil {
		return nil, err
	}

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

// temporary struct for dumping contents into
type credentials struct {
	README    []byte
	Cert      []byte
	Key       []byte
	CA        []byte
	CAKey     []byte
	DockerEnv []byte
	UUID      UUID
}

// Credentials holds the keys to the kingdom
type Credentials struct {
	README     string
	Cert       string
	Key        string
	CA         string
	CAKey      string
	DockerEnv  string
	DockerHost string
}

// UUID represents a UUID value. UUIDs can be compared and set to other values
// and accessed by byte.
type UUID [16]byte

func extractUUID(s string) (UUID, error) {
	s = strings.Trim(s, "/")
	var u UUID
	var err error

	if len(s) != 36 {
		return UUID{}, fmt.Errorf("Invalid UUID")
	}
	format := "%08x-%04x-%04x-%04x-%012x"

	// create stack addresses for each section of the uuid.
	p := make([][]byte, 5)

	if _, err := fmt.Sscanf(s, format, &p[0], &p[1], &p[2], &p[3], &p[4]); err != nil {
		return u, err
	}

	copy(u[0:4], p[0])
	copy(u[4:6], p[1])
	copy(u[6:8], p[2])
	copy(u[8:10], p[3])
	copy(u[10:16], p[4])

	return u, err
}

// GetCredentials returns a Credentials struct for the given cluster name
func (c *ClusterClient) GetCredentials(clusterName string) (*Credentials, error) {
	url, err := c.ZipURL(clusterName)
	if err != nil {
		return nil, err
	}
	zr, err := fetchZip(url)
	if err != nil || len(zr.File) < 6 {
		return nil, err
	}

	// fetch the contents for each credential/note
	creds := new(credentials)
	for _, zf := range zr.File {
		// dir should be the UUID that comes out in the bundle
		dir, fname := path.Split(zf.Name)
		fi := zf.FileInfo()

		if fi.IsDir() {
			// get uuid that's part of the zip dump
			creds.UUID, err = extractUUID(dir)
			if err != nil {
				return nil, errors.New("Unable to read UUID from directory name in zip file: " + err.Error())
			}
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
		}
	}

	cleanCreds := Credentials{
		Cert:      string(creds.Cert),
		Key:       string(creds.Key),
		CA:        string(creds.CA),
		CAKey:     string(creds.CAKey),
		DockerEnv: string(creds.DockerEnv),
	}

	sourceLines := strings.Split(cleanCreds.DockerEnv, "\n")
	for _, line := range sourceLines {
		if strings.Index(line, "export ") == 0 {
			varDecl := strings.TrimRight(line[7:], "\n")
			eqLocation := strings.Index(varDecl, "=")

			varName := varDecl[:eqLocation]
			varValue := varDecl[eqLocation+1:]

			switch varName {
			case "DOCKER_HOST":
				cleanCreds.DockerHost = varValue
			}

		}
	}

	return &cleanCreds, nil
}

func fetchZip(zipurl string) (*zip.Reader, error) {
	resp, err := http.Get(zipurl)
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
