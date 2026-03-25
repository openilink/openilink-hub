package store

import "encoding/json"

type Plugin struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Namespace       string `json:"namespace,omitempty"`
	Description     string `json:"description"`
	Author          string `json:"author"`
	Icon            string `json:"icon,omitempty"`
	License         string `json:"license,omitempty"`
	Homepage        string `json:"homepage,omitempty"`
	OwnerID         string `json:"owner_id"`
	LatestVersionID string `json:"latest_version_id,omitempty"`
	InstallCount    int    `json:"install_count"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`

	// Joined
	OwnerName string `json:"owner_name,omitempty"`
}

type PluginVersion struct {
	ID             string          `json:"id"`
	PluginID       string          `json:"plugin_id"`
	Version        string          `json:"version"`
	Changelog      string          `json:"changelog,omitempty"`
	Script         string          `json:"script,omitempty"`
	ConfigSchema   json.RawMessage `json:"config_schema"`
	GithubURL      string          `json:"github_url,omitempty"`
	CommitHash     string          `json:"commit_hash,omitempty"`
	MatchTypes     string          `json:"match_types"`
	ConnectDomains string          `json:"connect_domains"`
	GrantPerms     string          `json:"grant_perms"`
	TimeoutSec     int             `json:"timeout_sec"`
	Status         string          `json:"status"`
	RejectReason   string          `json:"reject_reason,omitempty"`
	ReviewedBy     string          `json:"reviewed_by,omitempty"`
	CreatedAt      int64           `json:"created_at"`

	// Joined
	ReviewerName  string `json:"reviewer_name,omitempty"`
	PluginName    string `json:"name,omitempty"`
	PluginIcon    string `json:"icon,omitempty"`
	PluginDesc    string `json:"description,omitempty"`
	PluginAuthor  string `json:"author,omitempty"`
	SubmitterName string `json:"submitter_name,omitempty"`
}

type PluginWithLatest struct {
	Plugin
	Version        string          `json:"version,omitempty"`
	Changelog      string          `json:"changelog,omitempty"`
	MatchTypes     string          `json:"match_types,omitempty"`
	ConnectDomains string          `json:"connect_domains,omitempty"`
	GrantPerms     string          `json:"grant_perms,omitempty"`
	ConfigSchema   json.RawMessage `json:"config_schema,omitempty"`
}

type ConfigField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type PluginStore interface {
	CreatePlugin(p *Plugin) (*Plugin, error)
	GetPlugin(id string) (*Plugin, error)
	GetPluginByName(name string) (*Plugin, error)
	ListPlugins() ([]PluginWithLatest, error)
	ListPluginsByOwner(ownerID string) ([]PluginWithLatest, error)
	UpdatePluginMeta(id string, p *Plugin) error
	DeletePlugin(id string) error
	CreatePluginVersion(v *PluginVersion) (*PluginVersion, error)
	GetPluginVersion(id string) (*PluginVersion, error)
	ListPluginVersions(pluginID string) ([]PluginVersion, error)
	ListPendingVersions() ([]PluginVersion, error)
	SupersedeNonApprovedVersions(pluginID string)
	FindPendingVersion(pluginID string) (*PluginVersion, error)
	UpdatePluginVersion(id string, v *PluginVersion) error
	ReviewPluginVersion(id, status, reviewedBy, reason string) error
	DeletePluginVersion(id string) error
	RecordPluginInstall(pluginID, userID string) error
	CancelPluginVersion(id string) error
	FindPluginOwner(name string) (string, error)
	ResolvePluginScript(versionID string) (script, version string, timeoutSec int, err error)
}
