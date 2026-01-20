package repositories

import (
	"github.com/grafana/grafana/pkg/services/provisioning/values"
)

// ConfigVersion is used to figure out which API version a config uses.
type configVersion struct {
	APIVersion int64 `json:"apiVersion" yaml:"apiVersion"`
}

// configs is the normalized internal representation of repository configurations
type configs struct {
	APIVersion         int64
	Repositories       []*repositoryFromConfig
	DeleteRepositories []*deleteRepositoryConfig
}

// repositoryFromConfig represents a repository to be created/updated
type repositoryFromConfig struct {
	Name        string
	Title       string
	Description string
	Type        string // local, github, gitlab, bitbucket

	// Sync configuration
	SyncEnabled         bool
	SyncTarget          string // folder, instance
	SyncIntervalSeconds int64
	SyncAllowsEdits     bool // hybrid mode: allow UI edits in synced folders

	// Local repository settings
	LocalPath string

	// Git repository settings (shared by github, gitlab, bitbucket, git)
	GitURL                  string
	GitBranch               string
	GitPath                 string // subdirectory within repository
	GitToken                string
	GitGeneratePreviewLinks bool

	// GitHub-specific settings
	GitHubOwner      string
	GitHubRepository string
	GitHubAppID      int64
	GitHubInstallID  int64
	GitHubPrivateKey string

	// Workflow settings
	WorkflowWriteDirect   bool   // write directly to branch vs PR
	WorkflowBranch        string // branch for write workflow
	WorkflowPRLabels      []string
	WorkflowPRAssignees   []string
	WorkflowCommitMessage string
}

// deleteRepositoryConfig represents a repository to be deleted
type deleteRepositoryConfig struct {
	Name string
}

// configsV1 is the v1 file format for repository provisioning
type configsV1 struct {
	configVersion

	Repositories       []*repositoryFromConfigV1   `json:"repositories" yaml:"repositories"`
	DeleteRepositories []*deleteRepositoryConfigV1 `json:"deleteRepositories" yaml:"deleteRepositories"`
}

// repositoryFromConfigV1 is the v1 format for a repository configuration
type repositoryFromConfigV1 struct {
	Name        values.StringValue `json:"name" yaml:"name"`
	Title       values.StringValue `json:"title" yaml:"title"`
	Description values.StringValue `json:"description" yaml:"description"`
	Type        values.StringValue `json:"type" yaml:"type"`

	// Sync configuration
	Sync *syncConfigV1 `json:"sync" yaml:"sync"`

	// Local repository settings
	Local *localConfigV1 `json:"local" yaml:"local"`

	// GitHub settings
	GitHub *githubConfigV1 `json:"github" yaml:"github"`

	// GitLab settings
	GitLab *gitlabConfigV1 `json:"gitlab" yaml:"gitlab"`

	// Bitbucket settings
	Bitbucket *bitbucketConfigV1 `json:"bitbucket" yaml:"bitbucket"`

	// Generic Git settings
	Git *gitConfigV1 `json:"git" yaml:"git"`

	// Workflow settings
	Workflow *workflowConfigV1 `json:"workflow" yaml:"workflow"`
}

type syncConfigV1 struct {
	Enabled         values.BoolValue   `json:"enabled" yaml:"enabled"`
	Target          values.StringValue `json:"target" yaml:"target"`
	IntervalSeconds values.Int64Value  `json:"intervalSeconds" yaml:"intervalSeconds"`
	AllowsEdits     values.BoolValue   `json:"allowsEdits" yaml:"allowsEdits"` // hybrid mode: allow UI edits in synced folders
}

type localConfigV1 struct {
	Path values.StringValue `json:"path" yaml:"path"`
}

type githubConfigV1 struct {
	URL                  values.StringValue `json:"url" yaml:"url"`
	Owner                values.StringValue `json:"owner" yaml:"owner"`
	Repository           values.StringValue `json:"repository" yaml:"repository"`
	Branch               values.StringValue `json:"branch" yaml:"branch"`
	Path                 values.StringValue `json:"path" yaml:"path"`
	Token                values.StringValue `json:"token" yaml:"token"`
	AppID                values.Int64Value  `json:"appId" yaml:"appId"`
	InstallationID       values.Int64Value  `json:"installationId" yaml:"installationId"`
	PrivateKey           values.StringValue `json:"privateKey" yaml:"privateKey"`
	GeneratePreviewLinks values.BoolValue   `json:"generatePreviewLinks" yaml:"generatePreviewLinks"`
}

type gitlabConfigV1 struct {
	URL                  values.StringValue `json:"url" yaml:"url"`
	Branch               values.StringValue `json:"branch" yaml:"branch"`
	Path                 values.StringValue `json:"path" yaml:"path"`
	Token                values.StringValue `json:"token" yaml:"token"`
	GeneratePreviewLinks values.BoolValue   `json:"generatePreviewLinks" yaml:"generatePreviewLinks"`
}

type bitbucketConfigV1 struct {
	URL                  values.StringValue `json:"url" yaml:"url"`
	Branch               values.StringValue `json:"branch" yaml:"branch"`
	Path                 values.StringValue `json:"path" yaml:"path"`
	Token                values.StringValue `json:"token" yaml:"token"`
	GeneratePreviewLinks values.BoolValue   `json:"generatePreviewLinks" yaml:"generatePreviewLinks"`
}

type gitConfigV1 struct {
	URL                  values.StringValue `json:"url" yaml:"url"`
	Branch               values.StringValue `json:"branch" yaml:"branch"`
	Path                 values.StringValue `json:"path" yaml:"path"`
	Token                values.StringValue `json:"token" yaml:"token"`
	GeneratePreviewLinks values.BoolValue   `json:"generatePreviewLinks" yaml:"generatePreviewLinks"`
}

type workflowConfigV1 struct {
	WriteDirect   values.BoolValue   `json:"writeDirect" yaml:"writeDirect"`
	Branch        values.StringValue `json:"branch" yaml:"branch"`
	PRLabels      []string           `json:"prLabels" yaml:"prLabels"`
	PRAssignees   []string           `json:"prAssignees" yaml:"prAssignees"`
	CommitMessage values.StringValue `json:"commitMessage" yaml:"commitMessage"`
}

type deleteRepositoryConfigV1 struct {
	Name values.StringValue `json:"name" yaml:"name"`
}

// mapToRepositoryFromConfig converts v1 config to normalized format
func (cfg *configsV1) mapToRepositoryFromConfig(apiVersion int64) *configs {
	r := &configs{
		APIVersion: apiVersion,
	}

	if cfg == nil {
		return r
	}

	for _, repo := range cfg.Repositories {
		normalized := &repositoryFromConfig{
			Name:        repo.Name.Value(),
			Title:       repo.Title.Value(),
			Description: repo.Description.Value(),
			Type:        repo.Type.Value(),
		}

		// Sync settings
		if repo.Sync != nil {
			normalized.SyncEnabled = repo.Sync.Enabled.Value()
			normalized.SyncTarget = repo.Sync.Target.Value()
			normalized.SyncIntervalSeconds = repo.Sync.IntervalSeconds.Value()
			normalized.SyncAllowsEdits = repo.Sync.AllowsEdits.Value()
		}

		// Local settings
		if repo.Local != nil {
			normalized.LocalPath = repo.Local.Path.Value()
		}

		// GitHub settings
		if repo.GitHub != nil {
			normalized.GitURL = repo.GitHub.URL.Value()
			normalized.GitHubOwner = repo.GitHub.Owner.Value()
			normalized.GitHubRepository = repo.GitHub.Repository.Value()
			normalized.GitBranch = repo.GitHub.Branch.Value()
			normalized.GitPath = repo.GitHub.Path.Value()
			normalized.GitToken = repo.GitHub.Token.Value()
			normalized.GitHubAppID = repo.GitHub.AppID.Value()
			normalized.GitHubInstallID = repo.GitHub.InstallationID.Value()
			normalized.GitHubPrivateKey = repo.GitHub.PrivateKey.Value()
			normalized.GitGeneratePreviewLinks = repo.GitHub.GeneratePreviewLinks.Value()
		}

		// GitLab settings
		if repo.GitLab != nil {
			normalized.GitURL = repo.GitLab.URL.Value()
			normalized.GitBranch = repo.GitLab.Branch.Value()
			normalized.GitPath = repo.GitLab.Path.Value()
			normalized.GitToken = repo.GitLab.Token.Value()
			normalized.GitGeneratePreviewLinks = repo.GitLab.GeneratePreviewLinks.Value()
		}

		// Bitbucket settings
		if repo.Bitbucket != nil {
			normalized.GitURL = repo.Bitbucket.URL.Value()
			normalized.GitBranch = repo.Bitbucket.Branch.Value()
			normalized.GitPath = repo.Bitbucket.Path.Value()
			normalized.GitToken = repo.Bitbucket.Token.Value()
			normalized.GitGeneratePreviewLinks = repo.Bitbucket.GeneratePreviewLinks.Value()
		}

		// Generic Git settings
		if repo.Git != nil {
			normalized.GitURL = repo.Git.URL.Value()
			normalized.GitBranch = repo.Git.Branch.Value()
			normalized.GitPath = repo.Git.Path.Value()
			normalized.GitToken = repo.Git.Token.Value()
			normalized.GitGeneratePreviewLinks = repo.Git.GeneratePreviewLinks.Value()
		}

		// Workflow settings
		if repo.Workflow != nil {
			normalized.WorkflowWriteDirect = repo.Workflow.WriteDirect.Value()
			normalized.WorkflowBranch = repo.Workflow.Branch.Value()
			normalized.WorkflowCommitMessage = repo.Workflow.CommitMessage.Value()
			normalized.WorkflowPRLabels = repo.Workflow.PRLabels
			normalized.WorkflowPRAssignees = repo.Workflow.PRAssignees
		}

		r.Repositories = append(r.Repositories, normalized)
	}

	for _, repo := range cfg.DeleteRepositories {
		r.DeleteRepositories = append(r.DeleteRepositories, &deleteRepositoryConfig{
			Name: repo.Name.Value(),
		})
	}

	return r
}
