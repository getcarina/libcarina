package libcarina

import (
	"net/http"
	"reflect"
	"testing"

	"fmt"
	"github.com/pkg/errors"
	"net/http/httptest"
)

const mockUsername = "test-user"
const mockAPIKey = "1234"
const mockRegion = "DFW"
const mockToken = ""

const microversionUnsupportedJSON = `{"errors":[{"code":"make-coe-api.microverion-unsupported","detail":"If the api-version header is sent, it must be in the format 'rax:container X.Y' where 1.0 <= X.Y <= 1.0","links":[{"href":"https://getcarina.com/docs/","rel":"help"}],"max_version":"1.0","min_version":"1.0","request_id":"620c8d81-b8f9-4bb0-952b-6d08ae42eda0","status":406,"title":"Microversion unsupported"}]}`

type handler func(w http.ResponseWriter, r *http.Request)

func identityHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.RequestURI {
	case "/v2.0/tokens":
		fmt.Fprintln(w, `{"access":{"serviceCatalog":[{"endpoints":[{"tenantId":"963451","publicURL":"https:\/\/api.dfw.getcarina.com","region":"DFW"}],"name":"cloudContainer","type":"rax:container"}],"user":{"name":"fake-user","id":"fake-userid"},"token":{"expires":"3000-01-01T12:00:00Z","id":"fake-token","tenant":{"name":"fake-tenantname","id":"fake-tenantid"}}}}`)
	default:
		w.WriteHeader(404)
		fmt.Fprintln(w, "unexpected request: "+r.RequestURI)
	}
}

func microversionUnsupportedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(406)
	fmt.Fprintln(w, microversionUnsupportedJSON)
}

func createMockCarina(h handler) (*httptest.Server, *httptest.Server) {
	return httptest.NewServer(http.HandlerFunc(h)), httptest.NewServer(http.HandlerFunc(identityHandler))
}

func createMockCarinaClient(identity string, endpoint string) (*CarinaClient, error) {
	client, err := NewClient(mockUsername, mockAPIKey, mockRegion, identity, mockToken, endpoint)
	if err != nil {
		return client, err
	}
	client.Endpoint = endpoint
	return client, nil
}

func assertMicroversionUnsupportedHandled(t *testing.T, err error) {
	if err == nil {
		t.Error("expected to get error")
	}
	cause := errors.Cause(err)
	if httpErr, ok := cause.(HTTPErr); ok {
		if httpErr.StatusCode != 406 {
			t.Error("expected StatusCode of 406, got", httpErr.StatusCode)
		}
	} else {
		t.Error("expected to get HTTPErr, got", reflect.TypeOf(err))
	}
}

func TestMicroversionUnsupportedNewRequest(t *testing.T) {
	mockCarina, mockIdentity := createMockCarina(microversionUnsupportedHandler)
	defer mockCarina.Close()
	defer mockIdentity.Close()

	carinaClient, err := createMockCarinaClient(mockIdentity.URL+"/v2.0/", mockCarina.URL)
	if err != nil {
		t.Error("wasn't able to create carinaClient pointed at mockCarina.URL with error:", err)
		t.FailNow()
	}
	resp, err := carinaClient.NewRequest("GET", "/clusters", nil)
	if resp != nil {
		t.Error("expected nil response, got", resp)
	}
	assertMicroversionUnsupportedHandled(t, err)
}

func TestMicroversionUnsupportedGet(t *testing.T) {
	mockCarina, mockIdentity := createMockCarina(microversionUnsupportedHandler)
	defer mockCarina.Close()
	defer mockIdentity.Close()

	carinaClient, err := createMockCarinaClient(mockIdentity.URL+"/v2.0/", mockCarina.URL)
	if err != nil {
		t.Error("wasn't able to create carinaClient pointed at mockCarina.URL with error:", err)
		t.FailNow()
	}
	resp, err := carinaClient.Get("9f18f7f9-aeb4-4c7c-91ef-e13ff94e352c")
	if resp != nil {
		t.Error("expected nil response, got", resp)
	}
	assertMicroversionUnsupportedHandled(t, err)
}

func TestMicroversionUnsupportedList(t *testing.T) {
	mockCarina, mockIdentity := createMockCarina(microversionUnsupportedHandler)
	defer mockCarina.Close()
	defer mockIdentity.Close()

	carinaClient, err := createMockCarinaClient(mockIdentity.URL+"/v2.0/", mockCarina.URL)
	if err != nil {
		t.Error("wasn't able to create carinaClient pointed at mockCarina.URL with error:", err)
		t.FailNow()
	}
	resp, err := carinaClient.List()
	if resp != nil {
		t.Error("expected nil response, got", resp)
	}
	assertMicroversionUnsupportedHandled(t, err)
}

func TestMicroversionUnsupportedListClusterTypes(t *testing.T) {
	mockCarina, mockIdentity := createMockCarina(microversionUnsupportedHandler)
	defer mockCarina.Close()
	defer mockIdentity.Close()

	carinaClient, err := createMockCarinaClient(mockIdentity.URL+"/v2.0/", mockCarina.URL)
	if err != nil {
		t.Error("wasn't able to create carinaClient pointed at mockCarina.URL with error:", err)
		t.FailNow()
	}
	resp, err := carinaClient.ListClusterTypes()
	if resp != nil {
		t.Error("expected nil response, got", resp)
	}
	assertMicroversionUnsupportedHandled(t, err)
}

func TestMicroversionUnsupportedCreate(t *testing.T) {
	mockCarina, mockIdentity := createMockCarina(microversionUnsupportedHandler)
	defer mockCarina.Close()
	defer mockIdentity.Close()

	carinaClient, err := createMockCarinaClient(mockIdentity.URL+"/v2.0/", mockCarina.URL)
	if err != nil {
		t.Error("wasn't able to create carinaClient pointed at mockCarina.URL with error:", err)
		t.FailNow()
	}
	clusterOpts := &CreateClusterOpts{
		Name:          "test-cluster",
		ClusterTypeID: 1,
		Nodes:         2,
	}
	resp, err := carinaClient.Create(clusterOpts)
	if resp != nil {
		t.Error("expected nil response, got", resp)
	}
	assertMicroversionUnsupportedHandled(t, err)
}

func TestMicroversionUnsupportedDelete(t *testing.T) {
	mockCarina, mockIdentity := createMockCarina(microversionUnsupportedHandler)
	defer mockCarina.Close()
	defer mockIdentity.Close()

	carinaClient, err := createMockCarinaClient(mockIdentity.URL+"/v2.0/", mockCarina.URL)
	if err != nil {
		t.Error("wasn't able to create carinaClient pointed at mockCarina.URL with error:", err)
		t.FailNow()
	}
	resp, err := carinaClient.Delete("9f18f7f9-aeb4-4c7c-91ef-e13ff94e352c")
	if resp != nil {
		t.Error("expected nil response, got", resp)
	}
	assertMicroversionUnsupportedHandled(t, err)
}

func TestMicroversionUnsupportedResize(t *testing.T) {
	mockCarina, mockIdentity := createMockCarina(microversionUnsupportedHandler)
	defer mockCarina.Close()
	defer mockIdentity.Close()

	carinaClient, err := createMockCarinaClient(mockIdentity.URL+"/v2.0/", mockCarina.URL)
	if err != nil {
		t.Error("wasn't able to create carinaClient pointed at mockCarina.URL with error:", err)
		t.FailNow()
	}
	resp, err := carinaClient.Resize("9f18f7f9-aeb4-4c7c-91ef-e13ff94e352c", 3)
	if resp != nil {
		t.Error("expected nil response, got", resp)
	}
	assertMicroversionUnsupportedHandled(t, err)
}

func TestMicroversionUnsupportedGetCredentials(t *testing.T) {
	mockCarina, mockIdentity := createMockCarina(microversionUnsupportedHandler)
	defer mockCarina.Close()
	defer mockIdentity.Close()

	carinaClient, err := createMockCarinaClient(mockIdentity.URL+"/v2.0/", mockCarina.URL)
	if err != nil {
		t.Error("wasn't able to create carinaClient pointed at mockCarina.URL with error:", err)
		t.FailNow()
	}
	resp, err := carinaClient.GetCredentials("9f18f7f9-aeb4-4c7c-91ef-e13ff94e352c")
	if resp != nil {
		t.Error("expected nil response, got", resp)
	}
	assertMicroversionUnsupportedHandled(t, err)
}

func TestMicroversionUnsupportedGetAPIMetadata(t *testing.T) {
	mockCarina, mockIdentity := createMockCarina(microversionUnsupportedHandler)
	defer mockCarina.Close()
	defer mockIdentity.Close()

	carinaClient, err := createMockCarinaClient(mockIdentity.URL+"/v2.0/", mockCarina.URL)
	if err != nil {
		t.Error("wasn't able to create carinaClient pointed at mockCarina.URL with error:", err)
		t.FailNow()
	}
	resp, err := carinaClient.GetAPIMetadata()
	if resp != nil {
		t.Error("expected nil response, got", resp)
	}
	assertMicroversionUnsupportedHandled(t, err)
}
