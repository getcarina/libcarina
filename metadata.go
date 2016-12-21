package libcarina

// SupportedAPIVersion is the version of the API against which this library was developed
const SupportedAPIVersion = "1.0"

// CarinaEndpointType is the endpoint type in the service catalog
const CarinaEndpointType = "rax:container"

// APIMetadata contains information about the API
type APIMetadata struct {
	// Versions is a list of supported API versions
	Versions []*APIVersion
}

// APIVersion defines a version of the API
type APIVersion struct {
	ID      string `json:"id"`
	Status  string `json:"current"`
	Minimum string `json:"min_version"`
	Maximum string `json:"max_version"`
}
