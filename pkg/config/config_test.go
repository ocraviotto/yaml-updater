package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRepoConfigurationFind(t *testing.T) {
	findTests := []struct {
		name string
		want *Repository
	}{
		{"testing", &Repository{Name: "testing"}},
		{"unknown", nil},
	}

	cfgs := RepoConfiguration{
		Repositories: []*Repository{
			{Name: "testing"},
			{Name: "another"},
		},
	}

	for _, tt := range findTests {
		if diff := cmp.Diff(tt.want, cfgs.Find(tt.name)); diff != "" {
			t.Errorf("Find(%s) failed:\n %s", tt.name, diff)
		}
	}
}

func TestParse(t *testing.T) {
	parseTests := []struct {
		filename string
		want     *RepoConfiguration
	}{
		{
			"testdata/config.yaml", &RepoConfiguration{
				Repositories: []*Repository{
					{
						Name:               "testing/repo-image",
						SourceRepo:         "example/example-source",
						SourceBranch:       "main",
						FilePath:           "test/file.yaml",
						UpdateKey:          "person.name",
						BranchGenerateName: "repo-imager-",
					},
				},
			},
		},
	}

	for _, tt := range parseTests {
		t.Run(fmt.Sprintf("parsing %s", tt.filename), func(rt *testing.T) {
			f, err := os.Open(tt.filename)
			if err != nil {
				rt.Errorf("failed to open %v: %s", tt.filename, err)
			}
			defer f.Close()

			got, err := Parse(f)
			if err != nil {
				rt.Errorf("failed to parse %v: %s", tt.filename, err)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				rt.Errorf("Parse(%s) failed diff\n%s", tt.filename, diff)
			}
		})
	}
}

func TestConfigLoad(t *testing.T) {
	s := &Signature{
		Name:  "John Doe",
		Email: "john.doe@example.com",
	}
	parseTests := []struct {
		filename string
		want     *RepoConfiguration
	}{
		{
			"testdata/.yaml-updater.yaml", &RepoConfiguration{
				Repositories: []*Repository{
					{
						Name:               "testing/repo-image",
						SourceRepo:         "my-org/my-project",
						SourceBranch:       "main",
						FilePath:           "service-a/deployment.yaml",
						UpdateKey:          "spec.template.spec.containers.0.image",
						BranchGenerateName: "repo-imager-",
						CreateMissing:      true,
						Signature:          s,
					},
					{
						Name:               "testing/repo-image",
						SourceRepo:         "my-org/my-other-project",
						SourceBranch:       "master",
						FilePath:           "service-a/deployment.yaml",
						UpdateKey:          "spec.template.spec.containers.0.image",
						BranchGenerateName: "repo-imager-",
						CreateMissing:      true,
						Signature:          s,
					},
				},
			},
		},
		{
			"testdata/config.yaml", &RepoConfiguration{
				Repositories: []*Repository{
					{
						Name:               "testing/repo-image",
						SourceRepo:         "example/example-source",
						SourceBranch:       "main",
						FilePath:           "test/file.yaml",
						UpdateKey:          "person.name",
						BranchGenerateName: "repo-imager-",
					},
				},
			},
		},
	}

	for _, tt := range parseTests {
		t.Run(fmt.Sprintf("parsing %s", tt.filename), func(rt *testing.T) {
			got, err := Load(tt.filename)
			if err != nil {
				rt.Errorf("failed to load %v: %s", tt.filename, err)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				rt.Errorf("Loading(%s) failed diff\n%s", tt.filename, diff)
			}
		})
	}
}
