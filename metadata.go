package libcarina

import (
	"sort"

	"github.com/Masterminds/semver"
	"fmt"
)

// SupportedAPIVersion is the version of the API against which this library was developed
const SupportedAPIVersion = "1.0"

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

// IsSupportedVersion determines if the current library version supports the connected Carina API version
func (metadata *APIMetadata) IsSupportedVersion() bool {
	// If we can't parse a version, assume it isn't supported
	supportedVersion, _ := semver.NewVersion(SupportedAPIVersion)

	for _, version := range metadata.Versions {
		versionRange, err := semver.NewConstraint(fmt.Sprintf(">= %s, <= %s", version.Minimum, version.Maximum))
		if err != nil {
			continue
		}

		if versionRange.Check(supportedVersion) {
			return true
		}
	}

	return false
}

// GetSupportedVersionRange returns the lowest and highest supported Carina API versions
func (metadata *APIMetadata) GetSupportedVersionRange() (min string, max string) {
	versions := []*semver.Version{}
	for _, version := range metadata.Versions {
		min, err := semver.NewVersion(version.Minimum)
		if err != nil {
			continue
		}
		max, _ := semver.NewVersion(version.Maximum)
		if err != nil {
			continue
		}
		versions = append(versions, min, max)
	}
	sort.Sort(semver.Collection(versions))

	numVersions := len(versions)
	if numVersions < 2 {
		return "0", "0"
	}

	minV := versions[0]
	min = fmt.Sprintf("%d.%d",minV.Major(), minV.Minor())

	maxV := versions[numVersions-1]
	max = fmt.Sprintf("%d.%d",maxV.Major(), maxV.Minor())

	return min, max
}
