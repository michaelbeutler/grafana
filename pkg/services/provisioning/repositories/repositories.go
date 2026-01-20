package repositories

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/apiserver"
)

var repositoryGVR = schema.GroupVersionResource{
	Group:    "provisioning.grafana.app",
	Version:  "v0alpha1",
	Resource: "repositories",
}

// Provisioner provisions GitSync repositories from configuration files
type Provisioner interface {
	Provision(ctx context.Context) error
}

// ProvisionerConfig contains the configuration for the repository provisioner
type ProvisionerConfig struct {
	Path               string
	RestConfigProvider apiserver.RestConfigProvider
}

type provisioner struct {
	log                log.Logger
	path               string
	restConfigProvider apiserver.RestConfigProvider
	configReader       ConfigReader
}

// NewProvisioner creates a new repository provisioner
func NewProvisioner(cfg ProvisionerConfig) Provisioner {
	logger := log.New("provisioning.repositories")
	return &provisioner{
		log:                logger,
		path:               cfg.Path,
		restConfigProvider: cfg.RestConfigProvider,
		configReader:       NewConfigReader(logger),
	}
}

// Provision reads configuration files and creates/updates/deletes repositories
func (p *provisioner) Provision(ctx context.Context) error {
	p.log.Info("Starting repository provisioning", "path", p.path)

	configs, err := p.configReader.ReadConfig(p.path)
	if err != nil {
		return fmt.Errorf("failed to read repository configs: %w", err)
	}

	if len(configs) == 0 {
		p.log.Debug("No repository provisioning configs found")
		return nil
	}

	// Get dynamic client
	restConfig, err := p.restConfigProvider.GetRestConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get REST config for repository provisioning: %w", err)
	}
	if restConfig == nil {
		return fmt.Errorf("REST config is nil for repository provisioning")
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	repoClient := dynamicClient.Resource(repositoryGVR).Namespace("default")

	// Process all configs
	for _, cfg := range configs {
		// Process deletions first
		for _, del := range cfg.DeleteRepositories {
			p.log.Info("Deleting provisioned repository", "name", del.Name)
			err := repoClient.Delete(ctx, del.Name, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				p.log.Error("Failed to delete repository", "name", del.Name, "error", err)
				return fmt.Errorf("failed to delete repository '%s': %w", del.Name, err)
			}
		}

		// Process creates/updates
		for _, repo := range cfg.Repositories {
			if err := p.provisionRepository(ctx, repoClient, repo); err != nil {
				return err
			}
		}
	}

	p.log.Info("Repository provisioning completed successfully")
	return nil
}

func (p *provisioner) provisionRepository(ctx context.Context, client dynamic.ResourceInterface, repo *repositoryFromConfig) error {
	p.log.Info("Provisioning repository", "name", repo.Name, "type", repo.Type)

	// Build the repository object
	obj := p.buildRepositoryObject(repo)

	// Check if repository exists
	existing, err := client.Get(ctx, repo.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check existing repository '%s': %w", repo.Name, err)
	}

	if existing != nil {
		// Update existing repository
		obj.SetResourceVersion(existing.GetResourceVersion())
		_, err = client.Update(ctx, obj, metav1.UpdateOptions{})
		if err != nil {
			p.log.Error("Failed to update repository", "name", repo.Name, "error", err)
			return fmt.Errorf("failed to update repository '%s': %w", repo.Name, err)
		}
		p.log.Info("Updated repository", "name", repo.Name)
	} else {
		// Create new repository
		_, err = client.Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			p.log.Error("Failed to create repository", "name", repo.Name, "error", err)
			return fmt.Errorf("failed to create repository '%s': %w", repo.Name, err)
		}
		p.log.Info("Created repository", "name", repo.Name)
	}

	return nil
}

func (p *provisioner) buildRepositoryObject(repo *repositoryFromConfig) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "provisioning.grafana.app/v0alpha1",
			"kind":       "Repository",
			"metadata": map[string]interface{}{
				"name":      repo.Name,
				"namespace": "default",
				"annotations": map[string]interface{}{
					"grafana.app/provenance": "file",
				},
			},
			"spec": map[string]interface{}{
				"type":        repo.Type,
				"title":       repo.Title,
				"description": repo.Description,
			},
		},
	}

	spec := obj.Object["spec"].(map[string]interface{})

	// Add sync configuration
	if repo.SyncEnabled || repo.SyncTarget != "" || repo.SyncIntervalSeconds > 0 || repo.SyncAllowsEdits {
		syncConfig := map[string]interface{}{
			"enabled": repo.SyncEnabled,
		}
		if repo.SyncTarget != "" {
			syncConfig["target"] = repo.SyncTarget
		}
		if repo.SyncIntervalSeconds > 0 {
			syncConfig["intervalSeconds"] = repo.SyncIntervalSeconds
		}
		if repo.SyncAllowsEdits {
			syncConfig["allowsEdits"] = repo.SyncAllowsEdits
		}
		spec["sync"] = syncConfig
	}

	// Add type-specific configuration
	switch repo.Type {
	case "local":
		spec["local"] = map[string]interface{}{
			"path": repo.LocalPath,
		}

	case "github":
		githubConfig := map[string]interface{}{}
		if repo.GitURL != "" {
			githubConfig["url"] = repo.GitURL
		}
		if repo.GitHubOwner != "" {
			githubConfig["owner"] = repo.GitHubOwner
		}
		if repo.GitHubRepository != "" {
			githubConfig["repository"] = repo.GitHubRepository
		}
		if repo.GitBranch != "" {
			githubConfig["branch"] = repo.GitBranch
		}
		if repo.GitPath != "" {
			githubConfig["path"] = repo.GitPath
		}
		if repo.GitToken != "" {
			githubConfig["token"] = map[string]interface{}{
				"value": repo.GitToken,
			}
		}
		if repo.GitHubAppID > 0 {
			githubConfig["appId"] = repo.GitHubAppID
		}
		if repo.GitHubInstallID > 0 {
			githubConfig["installationId"] = repo.GitHubInstallID
		}
		if repo.GitHubPrivateKey != "" {
			githubConfig["privateKey"] = map[string]interface{}{
				"value": repo.GitHubPrivateKey,
			}
		}
		if repo.GitGeneratePreviewLinks {
			githubConfig["generatePreviewLinks"] = repo.GitGeneratePreviewLinks
		}
		spec["github"] = githubConfig

	case "gitlab":
		gitlabConfig := map[string]interface{}{}
		if repo.GitURL != "" {
			gitlabConfig["url"] = repo.GitURL
		}
		if repo.GitBranch != "" {
			gitlabConfig["branch"] = repo.GitBranch
		}
		if repo.GitPath != "" {
			gitlabConfig["path"] = repo.GitPath
		}
		if repo.GitToken != "" {
			gitlabConfig["token"] = map[string]interface{}{
				"value": repo.GitToken,
			}
		}
		if repo.GitGeneratePreviewLinks {
			gitlabConfig["generatePreviewLinks"] = repo.GitGeneratePreviewLinks
		}
		spec["gitlab"] = gitlabConfig

	case "bitbucket":
		bitbucketConfig := map[string]interface{}{}
		if repo.GitURL != "" {
			bitbucketConfig["url"] = repo.GitURL
		}
		if repo.GitBranch != "" {
			bitbucketConfig["branch"] = repo.GitBranch
		}
		if repo.GitPath != "" {
			bitbucketConfig["path"] = repo.GitPath
		}
		if repo.GitToken != "" {
			bitbucketConfig["token"] = map[string]interface{}{
				"value": repo.GitToken,
			}
		}
		if repo.GitGeneratePreviewLinks {
			bitbucketConfig["generatePreviewLinks"] = repo.GitGeneratePreviewLinks
		}
		spec["bitbucket"] = bitbucketConfig

	case "git":
		gitConfig := map[string]interface{}{}
		if repo.GitURL != "" {
			gitConfig["url"] = repo.GitURL
		}
		if repo.GitBranch != "" {
			gitConfig["branch"] = repo.GitBranch
		}
		if repo.GitPath != "" {
			gitConfig["path"] = repo.GitPath
		}
		if repo.GitToken != "" {
			gitConfig["token"] = map[string]interface{}{
				"value": repo.GitToken,
			}
		}
		if repo.GitGeneratePreviewLinks {
			gitConfig["generatePreviewLinks"] = repo.GitGeneratePreviewLinks
		}
		spec["git"] = gitConfig
	}

	// Add workflow configuration
	if repo.WorkflowWriteDirect || repo.WorkflowBranch != "" || len(repo.WorkflowPRLabels) > 0 || len(repo.WorkflowPRAssignees) > 0 {
		workflow := map[string]interface{}{}
		if repo.WorkflowWriteDirect {
			workflow["write"] = "direct"
		} else {
			workflow["write"] = "branch"
		}
		if repo.WorkflowBranch != "" {
			workflow["branch"] = repo.WorkflowBranch
		}
		if len(repo.WorkflowPRLabels) > 0 {
			workflow["prLabels"] = repo.WorkflowPRLabels
		}
		if len(repo.WorkflowPRAssignees) > 0 {
			workflow["prAssignees"] = repo.WorkflowPRAssignees
		}
		if repo.WorkflowCommitMessage != "" {
			workflow["commitMessage"] = repo.WorkflowCommitMessage
		}
		spec["workflow"] = workflow
	}

	return obj
}

// Provision is the entry point function that matches the pattern used by other provisioners
func Provision(ctx context.Context, cfg ProvisionerConfig) error {
	p := NewProvisioner(cfg)
	return p.Provision(ctx)
}
