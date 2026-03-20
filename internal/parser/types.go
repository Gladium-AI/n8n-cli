package parser

import "strings"

type ParsedWorkflow struct {
	Meta        WorkflowMeta
	Nodes       []*ParsedNode
	Edges       []*ParsedEdge
	Indexes     WorkflowIndexes
	RawWorkflow map[string]interface{}
}

type WorkflowMeta struct {
	ID        string
	Name      string
	Active    bool
	Tags      []TagInfo
	Settings  map[string]interface{}
	CreatedAt string
	UpdatedAt string
	VersionID string
}

type TagInfo struct {
	ID   string
	Name string
}

type ParsedNode struct {
	Ref              string
	ID               string
	Name             string
	Type             string
	TypeVersion      int
	Position         [2]float64
	Disabled         bool
	AlwaysOutputData bool
	Parameters       map[string]interface{}
	Credentials      map[string]interface{}
	Notes            string
	OnError          string // "stopWorkflow", "continueRegularOutput", "continueErrorOutput"
	Inbound          []*ParsedEdge
	Outbound         []*ParsedEdge
	RawJSON          map[string]interface{} // exact native node JSON from workflow
}

type ParsedEdge struct {
	FromRef    string
	ToRef      string
	FromOutput int
	ToInput    int
	FromName   string
	ToName     string
}

type WorkflowIndexes struct {
	ByRef  map[string]*ParsedNode
	ByName map[string][]*ParsedNode
	ByID   map[string]*ParsedNode
}

type GraphAnalysis struct {
	Roots            []string
	Leaves           []string
	TopologicalOrder []string
	Orphans          []string
	HasCycles        bool
	CycleNodes       []string
	BranchingPoints  []string
	NodeCount        int
	EdgeCount        int
	AdjacencyList    map[string][]string
}

type NodeInput struct {
	Name             string
	Type             string
	TypeVersion      int
	Position         [2]float64
	Disabled         bool
	AlwaysOutputData bool
	Parameters       map[string]interface{}
	Credentials      map[string]interface{}
	Notes            string
	RawJSON          map[string]interface{}
}

type EdgeInput struct {
	FromRef    string
	FromOutput int
	ToRef      string
	ToInput    int
}

type NodePatch struct {
	Path  string
	Value interface{}
}

type DeleteOptions struct {
	Cascade        bool
	RewireStrategy string // "none", "skip", "bridge"
}

type UpdateResult struct {
	Node         *ParsedNode
	ChangedPaths []string
}

type NodeView string

const (
	ViewSummary     NodeView = "summary"
	ViewDetails     NodeView = "details"
	ViewJSON        NodeView = "json"
	ViewParams      NodeView = "params"
	ViewConnections NodeView = "connections"
)

func ParseNodeView(s string) NodeView {
	switch strings.ToLower(s) {
	case "details":
		return ViewDetails
	case "json":
		return ViewJSON
	case "params":
		return ViewParams
	case "connections":
		return ViewConnections
	default:
		return ViewSummary
	}
}
