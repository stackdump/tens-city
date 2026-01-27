package activitypub

import (
	"encoding/json"
	"net/http"
)

// NodeInfoWellKnown represents the /.well-known/nodeinfo response
type NodeInfoWellKnown struct {
	Links []NodeInfoLink `json:"links"`
}

// NodeInfoLink represents a link in the nodeinfo well-known response
type NodeInfoLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// NodeInfo represents the NodeInfo 2.0 schema response
type NodeInfo struct {
	Version           string           `json:"version"`
	Software          NodeInfoSoftware `json:"software"`
	Protocols         []string         `json:"protocols"`
	Services          NodeInfoServices `json:"services"`
	Usage             NodeInfoUsage    `json:"usage"`
	OpenRegistrations bool             `json:"openRegistrations"`
	Metadata          NodeInfoMetadata `json:"metadata"`
}

// NodeInfoServices describes inbound/outbound service integrations
type NodeInfoServices struct {
	Inbound  []string `json:"inbound"`
	Outbound []string `json:"outbound"`
}

// NodeInfoMetadata describes the node
type NodeInfoMetadata struct {
	NodeName        string               `json:"nodeName"`
	NodeDescription string               `json:"nodeDescription"`
	Software        NodeInfoMetaSoftware `json:"software"`
}

// NodeInfoMetaSoftware has links about the software project
type NodeInfoMetaSoftware struct {
	HomePage string `json:"homepage"`
	GitHub   string `json:"github"`
}

// NodeInfoSoftware describes the server software
type NodeInfoSoftware struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NodeInfoUsage describes server usage statistics
type NodeInfoUsage struct {
	Users      NodeInfoUsers `json:"users"`
	LocalPosts int           `json:"localPosts"`
}

// NodeInfoUsers describes user statistics
type NodeInfoUsers struct {
	Total          int `json:"total"`
	ActiveMonth    int `json:"activeMonth"`
	ActiveHalfyear int `json:"activeHalfyear"`
}

// HandleNodeInfoWellKnown handles /.well-known/nodeinfo requests
func (a *Actor) HandleNodeInfoWellKnown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := NodeInfoWellKnown{
		Links: []NodeInfoLink{
			{
				Rel:  "http://nodeinfo.diaspora.software/ns/schema/2.0",
				Href: "https://" + a.config.Domain + "/nodeinfo/2.0",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleNodeInfo handles /nodeinfo/2.0 requests
func (a *Actor) HandleNodeInfo(w http.ResponseWriter, r *http.Request, localPosts int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	softwareName := a.config.SoftwareName
	if softwareName == "" {
		softwareName = "tens-city"
	}
	softwareVersion := a.config.SoftwareVersion
	if softwareVersion == "" {
		softwareVersion = "1.0.0"
	}

	response := NodeInfo{
		Version: "2.0",
		Software: NodeInfoSoftware{
			Name:    softwareName,
			Version: softwareVersion,
		},
		Protocols: []string{"activitypub"},
		Services: NodeInfoServices{
			Inbound:  []string{},
			Outbound: []string{"rss2.0"},
		},
		Usage: NodeInfoUsage{
			Users: NodeInfoUsers{
				Total:          1,
				ActiveMonth:    1,
				ActiveHalfyear: 1,
			},
			LocalPosts: localPosts,
		},
		OpenRegistrations: false,
		Metadata: NodeInfoMetadata{
			NodeName:        a.config.NodeName,
			NodeDescription: a.config.NodeDescription,
			Software: NodeInfoMetaSoftware{
				HomePage: a.config.NodeHomePage,
				GitHub:   a.config.NodeGitHub,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=1800")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
