package libcarina

const (
	resizeTaskType = "resize"
)

// ResizeInput is an input params for a resize task
type resizeInput struct {
	// Node count to resize cluster to
	NodeCount int `json:"node_count"`
}

// ResizeTaskOpts is a data structure for the resize task
type resizeTaskOpts struct {
	Type  string       `json:"type"`
	Input *resizeInput `json:"input"`
}

func newResizeOpts(nodes int) *resizeTaskOpts {
	return &resizeTaskOpts{Type: resizeTaskType, Input: &resizeInput{NodeCount: nodes}}
}
