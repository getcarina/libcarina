package libcarina

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockQuotas = `{
    "max_clusters": 10,
    "max_nodes_per_cluster": 20
}`

func mockGetQuotas() *httptest.Server {
	s := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, mockQuotas)
	}

	return httptest.NewServer(http.HandlerFunc(s))
}

func TestQuotasFromResponse(t *testing.T) {
	mockGetQuotas := mockGetQuotas()
	defer mockGetQuotas.Close()
	var expectedMaxClusters Number = 10
	var expectedMaxNodesPerCluster Number = 20
	resp, err := http.Get(mockGetQuotas.URL)
	if err != nil {
		t.Fatal(err)
	}
	quotas, err := quotasFromResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	if quotas.MaxClusters != expectedMaxClusters {
		t.Errorf("Received MaxClusters = %v, expected MaxClusters = %v", quotas.MaxClusters, expectedMaxClusters)
	}
	if quotas.MaxNodesPerCluster != expectedMaxNodesPerCluster {
		t.Errorf("Received MaxNodesPerCluster = %v, expected MaxNodesPerCluster = %v", quotas.MaxNodesPerCluster, expectedMaxNodesPerCluster)
	}
}
