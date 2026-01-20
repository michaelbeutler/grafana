package repositories

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/grafana/grafana/pkg/infra/log"
)

// ConfigReader reads repository provisioning configuration files
type ConfigReader interface {
	ReadConfig(path string) ([]*configs, error)
}

type configReader struct {
	log log.Logger
}

// NewConfigReader creates a new repository config reader
func NewConfigReader(log log.Logger) ConfigReader {
	return &configReader{
		log: log,
	}
}

func (cr *configReader) ReadConfig(path string) ([]*configs, error) {
	var repositories []*configs

	files, err := os.ReadDir(path)
	if err != nil {
		// If the directory doesn't exist, that's fine - just return empty
		if os.IsNotExist(err) {
			cr.log.Debug("Repository provisioning directory does not exist", "path", path)
			return repositories, nil
		}
		cr.log.Error("Can't read repository provisioning files from directory", "path", path, "error", err)
		return repositories, nil
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml") {
			repo, err := cr.parseRepositoryConfig(path, file)
			if err != nil {
				return nil, err
			}

			if repo != nil {
				repositories = append(repositories, repo)
			}
		}
	}

	err = cr.validateConfigs(repositories)
	if err != nil {
		return nil, err
	}

	return repositories, nil
}

func (cr *configReader) parseRepositoryConfig(path string, file fs.DirEntry) (*configs, error) {
	filename, _ := filepath.Abs(filepath.Join(path, file.Name()))

	// nolint:gosec
	// We can ignore the gosec G304 warning on this one because `filename` comes from cfg.ProvisioningPath
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var apiVersion *configVersion
	err = yaml.Unmarshal(yamlFile, &apiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository config %s: %w", filename, err)
	}

	if apiVersion == nil {
		apiVersion = &configVersion{APIVersion: 0}
	}

	// Only v1 is supported for repositories
	if apiVersion.APIVersion < 1 {
		return nil, fmt.Errorf("repository provisioning config %s requires apiVersion: 1 or higher", filename)
	}

	v1 := &configsV1{}
	err = yaml.Unmarshal(yamlFile, v1)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository config %s: %w", filename, err)
	}

	return v1.mapToRepositoryFromConfig(apiVersion.APIVersion), nil
}

func (cr *configReader) validateConfigs(configs []*configs) error {
	names := make(map[string]bool)

	for _, cfg := range configs {
		for _, repo := range cfg.Repositories {
			if repo.Name == "" {
				return fmt.Errorf("repository provisioning: name is required")
			}

			if names[repo.Name] {
				return fmt.Errorf("repository provisioning: duplicate repository name '%s'", repo.Name)
			}
			names[repo.Name] = true

			// Validate type
			validTypes := map[string]bool{
				"local":     true,
				"github":    true,
				"gitlab":    true,
				"bitbucket": true,
				"git":       true,
			}
			if !validTypes[repo.Type] {
				return fmt.Errorf("repository provisioning: invalid type '%s' for repository '%s', must be one of: local, github, gitlab, bitbucket, git", repo.Type, repo.Name)
			}

			// Validate sync target
			if repo.SyncTarget != "" {
				validTargets := map[string]bool{
					"folder":   true,
					"instance": true,
				}
				if !validTargets[repo.SyncTarget] {
					return fmt.Errorf("repository provisioning: invalid sync target '%s' for repository '%s', must be one of: folder, instance", repo.SyncTarget, repo.Name)
				}
			}

			// Validate type-specific requirements
			switch repo.Type {
			case "local":
				if repo.LocalPath == "" {
					return fmt.Errorf("repository provisioning: local.path is required for local repository '%s'", repo.Name)
				}
			case "github":
				if repo.GitHubOwner == "" || repo.GitHubRepository == "" {
					if repo.GitURL == "" {
						return fmt.Errorf("repository provisioning: github.owner and github.repository (or github.url) are required for github repository '%s'", repo.Name)
					}
				}
			case "gitlab", "bitbucket", "git":
				if repo.GitURL == "" {
					return fmt.Errorf("repository provisioning: url is required for %s repository '%s'", repo.Type, repo.Name)
				}
			}
		}

		for _, repo := range cfg.DeleteRepositories {
			if repo.Name == "" {
				return fmt.Errorf("repository provisioning: name is required for delete")
			}
		}
	}

	return nil
}
