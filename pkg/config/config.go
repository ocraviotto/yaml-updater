package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"sigs.k8s.io/yaml"
)

// Repository is the items that are required to update a specific file in a repo.
type Repository struct {
	Name               string     `json:"name"`
	SourceRepo         string     `json:"sourceRepo"`
	SourceBranch       string     `json:"sourceBranch"`
	FilePath           string     `json:"filePath"`
	UpdateKey          string     `json:"updateKey"`
	BranchGenerateName string     `json:"branchGenerateName"`
	RemoveKey          bool       `json:"removeKey,omitempty"`
	RemoveFile         bool       `json:"removeFile,omitempty"`
	CreateMissing      bool       `json:"createMissing,omitempty"`
	CommitMsg          string     `json:"commitMsg,omitempty"`
	Signature          *Signature `json:"signature,omitempty"`
}

// Signature represents a git commit creator by name and email
type Signature struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

func Load(path string) (*RepoConfiguration, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse reads and returns a configuration from Reader.
func Parse(in io.Reader) (*RepoConfiguration, error) {
	body, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML: %w", err)
	}
	rc := &RepoConfiguration{}
	err = yaml.Unmarshal(body, rc)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	return rc, nil
}

// RepoConfiguration is a slice of Repository values.
type RepoConfiguration struct {
	Repositories []*Repository `json:"repositories"`
}

// Find looks up the repository by name.
func (c RepoConfiguration) Find(name string) *Repository {
	for _, cfg := range c.Repositories {
		if cfg.Name == name {
			return cfg
		}
	}
	return nil
}
