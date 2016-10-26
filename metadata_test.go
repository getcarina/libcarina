package libcarina

import (
	"testing"
)

func TestAPIMetadata_IsSupportedVersion_Equal(t *testing.T) {
	metadata := &APIMetadata{
		Versions: []*APIVersion{
			{Minimum: "1.0", Maximum: "1.0"},
		},
	}

	if !metadata.IsSupportedVersion() {
		t.Fail()
	}
}

func TestAPIMetadata_IsSupportedVersion_InRange(t *testing.T) {
	metadata := &APIMetadata{
		Versions: []*APIVersion{
			{Minimum: "0.9", Maximum: "2.0"},
		},
	}

	if !metadata.IsSupportedVersion() {
		t.Fail()
	}
}

func TestAPIMetadata_IsSupportedVersion_LessThan(t *testing.T) {
	metadata := &APIMetadata{
		Versions: []*APIVersion{
			{Minimum: "2.0", Maximum: "2.0"},
		},
	}

	if metadata.IsSupportedVersion() {
		t.Fail()
	}
}

func TestAPIMetadata_IsSupportedVersion_GreaterThan(t *testing.T) {
	metadata := &APIMetadata{
		Versions: []*APIVersion{
			{Minimum: "0.9", Maximum: "0.9"},
		},
	}

	if metadata.IsSupportedVersion() {
		t.Fail()
	}
}

func TestAPIMetadata_GetSupportedVersionRange(t *testing.T) {
	metadata := &APIMetadata{
		Versions: []*APIVersion{
			{Minimum: "1.0", Maximum: "1.5"},
			{Minimum: "0.5", Maximum: "0.9"},
		},
	}

	min, max := metadata.GetSupportedVersionRange()
	t.Log("min=%s, max=%s", min, max)
	if min != "0.5" {
		t.Logf("Expected min: 0.5 but got %s\n", min)
		t.Fail()
	}
	if max != "1.5" {
		t.Logf("Expected max: 1.5 but got %s\n", max)
		t.Fail()
	}
}
