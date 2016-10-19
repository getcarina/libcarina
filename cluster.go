package libcarina

// Cluster is a cluster of Docker nodes
type Cluster struct {
	// ID of the cluster
	ID string `json:"id"`

	// Name of the cluster
	Name string `json:"name"`

	// Type of cluster
	Type *ClusterType `json:"cluster_type"`

	// Nodes in the cluster
	Nodes int `json:"node_count,omitempty"`

	// Status of the cluster
	Status string `json:"status,omitempty"`
}

// ClusterType defines a type of cluster
// Essentially the template used to create a new cluster
type ClusterType struct {
	// ID of the cluster type
	ID int `json:"id"`

	// Name of the cluster type
	Name string `json:"name"`

	// Specifies if the cluster type is available to be used for new clusters
	IsActive bool `json:"active"`

	// COE (container orchestration engine) used by the cluster
	COE string `json:"coe"`

	// Underlying type of the host nodes, such as lxc or vm
	HostType string `json:"host_type"`
}

// CreateClusterOpts defines the set of parameters when creating a cluster
type CreateClusterOpts struct {
	// Name of the cluster
	Name string `json:"name"`

	// Type of cluster
	ClusterTypeID int `json:"cluster_type_id"`

	// Nodes in the cluster
	Nodes int `json:"node_count,omitempty"`
}
