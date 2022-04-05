package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ocraviotto/yaml-updater/pkg/config"
	"github.com/spf13/viper"
)

type testFlags map[string]interface{}

func TestOverrides(t *testing.T) {
	s := &config.Signature{
		Name:  "John Doe",
		Email: "john.doe@example.com",
	}
	parseTests := []struct {
		testName string
		filename string
		flags    *testFlags
		want     *config.RepoConfiguration
	}{
		{
			"testOverrideRepositories",
			"testdata/base-repositories.yaml",
			&testFlags{
				"override-repositories": "testRepo1,testRepo2",
				"disabled":              true,
			},
			&config.RepoConfiguration{
				Repositories: map[string]*config.Repository{},
			},
		},
		{
			"testOverridesAll",
			"testdata/base-repositories.yaml",
			&testFlags{
				"override-all": true,
				"disabled":     true,
			},
			&config.RepoConfiguration{
				Repositories: map[string]*config.Repository{},
			},
		},
		{
			"testOnlyWithOverrides",
			"testdata/base-repositories.yaml",
			&testFlags{
				"only":                 "testRepo3",
				"disabled":             true,
				"source-branch":        "branch3",
				"create-missing":       false,
				"branch-generate-name": "",
				"committer-name":       "Doe, John",
			},
			&config.RepoConfiguration{
				Repositories: map[string]*config.Repository{
					"testRepo3": {
						Disabled:           false,
						Name:               "testing/another-repo",
						SourceRepo:         "my-org/my-other-project",
						SourceBranch:       "branch3",
						FilePath:           "argocd/application.yaml",
						UpdateKey:          "spec.source.targetRevision",
						BranchGenerateName: "",
						Signature: &config.Signature{
							Name:  "Doe, John",
							Email: "john.doe@example.com",
						},
					},
				},
			},
		},
		{
			"testNoOverridesSingleRepoInConfig",
			"testdata/single-repository.yaml",
			&testFlags{
				"source-branch":       "direct",
				"disable-pr-creation": true,
			},
			&config.RepoConfiguration{
				Repositories: map[string]*config.Repository{
					"testRepo1": {
						Name:               "testing/repo-image",
						SourceRepo:         "my-org/my-project",
						SourceBranch:       "main",
						FilePath:           "service-a/deployment.yaml",
						UpdateKey:          "spec.template.spec.containers.0.image",
						BranchGenerateName: "repo-imager-",
						DisablePRCreation:  false,
						CreateMissing:      true,
						Signature:          s,
					},
				},
			},
		},
		{
			"testOverridesAll",
			"testdata/base-repositories.yaml",
			&testFlags{
				"override-all": true,
				"disabled":     false,
			},
			&config.RepoConfiguration{
				Repositories: map[string]*config.Repository{
					"testRepo1": {
						Name:               "testing/repo-image",
						SourceRepo:         "my-org/my-project",
						SourceBranch:       "main",
						FilePath:           "service-a/deployment.yaml",
						UpdateKey:          "spec.template.spec.containers.0.image",
						BranchGenerateName: "repo-imager-",
						CreateMissing:      true,
						Signature:          s,
					},
					"testRepo2": {
						Name:               "testing/repo-image2",
						SourceRepo:         "my-org/my-other-project",
						SourceBranch:       "master",
						FilePath:           "service-b/pod.yaml",
						UpdateKey:          "spec.containers.0.image",
						BranchGenerateName: "",
						CreateMissing:      true,
						Signature:          s,
					},
					"testRepo3": {
						Name:               "testing/another-repo",
						SourceRepo:         "my-org/my-other-project",
						SourceBranch:       "master",
						FilePath:           "argocd/application.yaml",
						UpdateKey:          "spec.source.targetRevision",
						BranchGenerateName: "",
						CreateMissing:      true,
						Signature:          s,
					},
				},
			},
		},
		{
			"testOverridesGen",
			"testdata/base-repositories.yaml",
			&testFlags{
				"override-repositories": "testRepo2,testRepo3",
				"source-branch":         "direct",
				"disable-pr-creation":   true,
			},
			&config.RepoConfiguration{
				Repositories: map[string]*config.Repository{
					"testRepo1": {
						Name:               "testing/repo-image",
						SourceRepo:         "my-org/my-project",
						SourceBranch:       "main",
						FilePath:           "service-a/deployment.yaml",
						UpdateKey:          "spec.template.spec.containers.0.image",
						BranchGenerateName: "repo-imager-",
						CreateMissing:      true,
						Signature:          s,
					},
					"testRepo2": {
						Name:               "testing/repo-image2",
						SourceRepo:         "my-org/my-other-project",
						SourceBranch:       "direct",
						FilePath:           "service-b/pod.yaml",
						UpdateKey:          "spec.containers.0.image",
						BranchGenerateName: "",
						DisablePRCreation:  true,
						CreateMissing:      true,
						Signature:          s,
					},
				},
			},
		},
	}

	for _, tt := range parseTests {
		t.Run(fmt.Sprintf("parsing %s", tt.filename), func(rt *testing.T) {
			initViper()
			makeUpdateCmd()
			repositories := loadRepositoriesFromFile(tt.filename, rt)

			setViperFromTestFlags(tt.flags)

			got, err := processConfigsAndOverrides(repositories)
			if err != nil {
				t.Fatalf("error with applyConfigsOverrides %#v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				rt.Errorf("Test %s (loading %s) failed diff\n%s", tt.testName, tt.filename, diff)
			}
		})
	}
}

func TestConfigFromFlags(t *testing.T) {
	s := &config.Signature{
		Name:  "John Doe",
		Email: "john.doe@example.com",
	}
	parseTests := []struct {
		testName string
		flags    *testFlags
		want     *config.Repository
	}{
		{
			"testDisablePR",
			&testFlags{
				"image-repo":     "testing/another-repo",
				"source-repo":    "my-org/my-other-project",
				"source-branch":  "branch3",
				"file-path":      "argocd/application.yaml",
				"update-key":     "spec.source.targetRevision",
				"create-missing": false,
				"disabled":       true,
			},
			&config.Repository{
				Disabled:           false,
				Name:               "testing/another-repo",
				SourceRepo:         "my-org/my-other-project",
				SourceBranch:       "branch3",
				BranchGenerateName: "gitops-",
				FilePath:           "argocd/application.yaml",
				UpdateKey:          "spec.source.targetRevision",
				Signature:          &config.Signature{},
			},
		}, {
			"testWithOverrides",
			&testFlags{
				"change-source-name":  "testing/another-repo",
				"source-repo":         "my-org/my-other-project",
				"source-branch":       "branch3",
				"file-path":           "argocd/application.yaml",
				"update-key":          "spec.source.targetRevision",
				"committer-name":      "John Doe",
				"committer-email":     "john.doe@example.com",
				"commit-msg":          "hello from my PR",
				"disable-pr-creation": true,
			},
			&config.Repository{
				Disabled:           false,
				Name:               "testing/another-repo",
				SourceRepo:         "my-org/my-other-project",
				SourceBranch:       "branch3",
				BranchGenerateName: "gitops-",
				FilePath:           "argocd/application.yaml",
				UpdateKey:          "spec.source.targetRevision",
				DisablePRCreation:  true,
				CommitMsg:          "hello from my PR",
				CreateMissing:      true,
				Signature:          s,
			},
		},
	}

	for _, tt := range parseTests {
		t.Run(fmt.Sprintf("parsing test: %s", tt.testName), func(rt *testing.T) {
			initViper()
			makeUpdateCmd()
			setViperFromTestFlags(tt.flags)

			got := configFromFlags()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				rt.Errorf("Test %s failed diff\n%s", tt.testName, diff)
			}
		})
	}
}

func loadRepositoriesFromFile(fileName string, rt *testing.T) (repositories *config.RepoConfiguration) {
	repositories, err := config.Load(fileName)
	if err != nil {
		rt.Errorf("failed to parse testFile %v: %s", fileName, err)
		return nil
	}
	return repositories

}

func setViperFromTestFlags(flags *testFlags) {
	for f, v := range *flags {
		viper.Set(f, v)
	}
}

func initViper() {
	viper.Reset()
	viper.SetEnvPrefix("git")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
}
