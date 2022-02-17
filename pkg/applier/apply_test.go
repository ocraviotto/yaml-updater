package applier

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/zapr"
	"github.com/ocraviotto/go-scm/scm"
	pkgClient "github.com/ocraviotto/pkg/client"
	"github.com/ocraviotto/pkg/client/mock"
	"github.com/ocraviotto/pkg/updater"
	"github.com/ocraviotto/yaml-updater/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

const (
	testQuayRepo   = "mynamespace/repository"
	testGitHubRepo = "testorg/testrepo"
	testFilePath   = "environments/test/services/service-a/test.yaml"
)

func TestUpdaterWithSingleRepo(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, "master", []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, "master", testSHA)
	applier := makeApplier(t, m, createConfigs())
	newValue := "repo:production"

	err := applier.UpdateRepositories(context.Background(), newValue)
	if err != nil {
		t.Fatal(err)
	}

	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	want := fmt.Sprintf("test:\n  image: %s\n", newValue)
	if s := string(updated); s != want {
		t.Fatalf("update failed, got %#v, want %#v", s, want)
	}
	m.AssertBranchCreated(testGitHubRepo, "test-branch-a", testSHA)
	m.AssertPullRequestCreated(testGitHubRepo, &scm.PullRequestInput{
		Title:  "Automated image update",
		Body:   fmt.Sprintf("Automated update from %q", testQuayRepo),
		Source: "test-branch-a",
		Target: "master",
	})
}

func TestUpdaterWithSingleRepositoryMethod(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, "master", []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, "master", testSHA)
	configs := createConfigs()
	applier := makeApplier(t, m, configs)
	newValue := "repo:production"

	err := applier.UpdateRepository(context.Background(), configs.Repositories[0], newValue)
	if err != nil {
		t.Fatal(err)
	}

	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	want := fmt.Sprintf("test:\n  image: %s\n", newValue)
	if s := string(updated); s != want {
		t.Fatalf("update failed, got %#v, want %#v", s, want)
	}
	m.AssertBranchCreated(testGitHubRepo, "test-branch-a", testSHA)
	m.AssertPullRequestCreated(testGitHubRepo, &scm.PullRequestInput{
		Title:  "Automated image update",
		Body:   fmt.Sprintf("Automated update from %q", testQuayRepo),
		Source: "test-branch-a",
		Target: "master",
	})
}

func TestUpdaterWithMultiRepo(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	anotherTestSHA := "ab40b7377b39a4f876e7f49639b580a80b66e8ad"
	secondRepo := "testorg/anothertestrepo"
	secondRepoBranch := "main"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, "master", []byte("test:\n  image: old-image\n"))
	m.AddFileContents(secondRepo, testFilePath, secondRepoBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, "master", testSHA)
	m.AddBranchHead(secondRepo, secondRepoBranch, anotherTestSHA)
	configs := createConfigs()
	secondConfig := createConfigs().Repositories[0]
	secondConfig.SourceRepo = secondRepo
	secondConfig.SourceBranch = secondRepoBranch
	configs.Repositories = append(configs.Repositories, secondConfig)
	applier := makeApplier(t, m, configs)
	newValue := "repo:production"

	err := applier.UpdateRepositories(context.Background(), newValue)
	if err != nil {
		t.Fatal(err)
	}

	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	want := fmt.Sprintf("test:\n  image: %s\n", newValue)
	if s := string(updated); s != want {
		t.Fatalf("update failed, got %#v, want %#v", s, want)
	}
	updated2 := m.GetUpdatedContents(secondRepo, testFilePath, "test-branch-a")
	if s2 := string(updated2); s2 != want {
		t.Fatalf("update failed, got %#v, want %#v", s2, want)
	}
	m.AssertBranchCreated(testGitHubRepo, "test-branch-a", testSHA)
	m.AssertBranchCreated(secondRepo, "test-branch-a", anotherTestSHA)
	m.AssertPullRequestCreated(testGitHubRepo, &scm.PullRequestInput{
		Title:  "Automated image update",
		Body:   fmt.Sprintf("Automated update from %q", testQuayRepo),
		Source: "test-branch-a",
		Target: "master",
	})
	m.AssertPullRequestCreated(secondRepo, &scm.PullRequestInput{
		Title:  "Automated image update",
		Body:   fmt.Sprintf("Automated update from %q", testQuayRepo),
		Source: "test-branch-a",
		Target: secondRepoBranch,
	})
}

// With no name-generator, the change should be made to master directly, rather
// than going through a PullRequest.
func TestUpdaterWithNoNameGenerator(t *testing.T) {
	sourceBranch := "production"
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, sourceBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, sourceBranch, testSHA)
	configs := createConfigs()
	configs.Repositories[0].BranchGenerateName = ""
	configs.Repositories[0].SourceBranch = sourceBranch
	applier := makeApplier(t, m, configs)
	newValue := "repo:production"

	err := applier.UpdateRepositories(context.Background(), newValue)
	if err != nil {
		t.Fatal(err)
	}

	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, sourceBranch)
	want := fmt.Sprintf("test:\n  image: %s\n", newValue)
	if s := string(updated); s != want {
		t.Fatalf("update failed, got %#v, want %#v", s, want)
	}
	m.AssertNoBranchesCreated()
	m.AssertNoPullRequestsCreated()
}

func TestUpdaterWithMissingFile(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, "master", []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, "master", testSHA)
	applier := makeApplier(t, m, createConfigs())
	newValue := "repo:production"
	testErr := errors.New("missing file")
	m.GetFileErr = testErr

	err := applier.UpdateRepositories(context.Background(), newValue)

	if err != testErr {
		t.Fatalf("got %s, want %s", err, testErr)
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	if s := string(updated); s != "" {
		t.Fatalf("update failed, got %#v, want %#v", s, "")
	}
	m.AssertNoBranchesCreated()
	m.AssertNoPullRequestsCreated()
}

func TestUpdaterWithCreateMissingFile(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, "master", []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, "master", testSHA)
	configs := createConfigs()
	configs.Repositories[0].SourceBranch = "master"
	configs.Repositories[0].CreateMissing = true
	applier := makeApplier(t, m, configs)
	newValue := "repo:production"
	testErr := pkgClient.SCMError{
		Msg:    fmt.Sprintf("failed to get file %s from repo %s ref %s", testFilePath, testGitHubRepo, testSHA),
		Status: 404,
	}
	m.GetFileErr = testErr

	err := applier.UpdateRepositories(context.Background(), newValue)
	if err != nil {
		t.Fatal(err)
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	want := fmt.Sprintf("test:\n  image: %s\n", newValue)
	if s := string(updated); s != want {
		t.Fatalf("update failed, got %#v, want %#v", s, want)
	}
}

func TestUpdaterWithBranchCreationFailure(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, "master", []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, "master", testSHA)
	applier := makeApplier(t, m, createConfigs())
	newValue := "repo:production"
	testErr := errors.New("can't create branch")
	m.CreateBranchErr = testErr

	err := applier.UpdateRepositories(context.Background(), newValue)

	if err.Error() != "failed to create branch: can't create branch" {
		t.Fatalf("got %s, want %s", err, "failed to create branch: can't create branch")
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	if s := string(updated); s != "" {
		t.Fatalf("update failed, got %#v, want %#v", s, "")
	}
	m.AssertNoBranchesCreated()
	m.AssertNoPullRequestsCreated()
}

func TestUpdaterWithUpdateFileFailure(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, "master", []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, "master", testSHA)
	applier := makeApplier(t, m, createConfigs())
	newValue := "repo:production"
	testErr := errors.New("can't update file")
	m.UpdateFileErr = testErr

	err := applier.UpdateRepositories(context.Background(), newValue)

	if err.Error() != "failed to update file: can't update file" {
		t.Fatalf("got %s, want %s", err, "failed to update file: can't update file")
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	if s := string(updated); s != "" {
		t.Fatalf("update failed, got %#v, want %#v", s, "")
	}
	m.AssertBranchCreated(testGitHubRepo, "test-branch-a", testSHA)
	m.RefutePullRequestCreated(testGitHubRepo, &scm.PullRequestInput{
		Title:  fmt.Sprintf("Image %s updated", testQuayRepo),
		Body:   "Automated Image Update",
		Source: "test-branch-a",
		Target: "master",
	})
}

func TestUpdaterWithCreatePullRequestFailure(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, "master", []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, "master", testSHA)
	applier := makeApplier(t, m, createConfigs())
	newValue := "repo:production"
	testErr := errors.New("failure")
	m.CreatePullRequestErr = testErr

	err := applier.UpdateRepositories(context.Background(), newValue)

	if err.Error() != "failed to create pull request in repo testorg/testrepo: failed to create a pull request: failure" {
		t.Fatalf("got %s, want %s", err, "failed to create a pull request: can't create pull-request")
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	want := fmt.Sprintf("test:\n  image: %s\n", newValue)
	if s := string(updated); s != want {
		t.Fatalf("update failed, got %#v, want %#v", s, "")
	}
	m.AssertBranchCreated(testGitHubRepo, "test-branch-a", testSHA)
	m.RefutePullRequestCreated(testGitHubRepo, &scm.PullRequestInput{
		Title:  fmt.Sprintf("Image %s updated", testQuayRepo),
		Body:   "Automated Image Update",
		Source: "test-branch-a",
		Target: "master",
	})
}

func makeApplier(t *testing.T, m *mock.MockClient, cfgs *config.RepoConfiguration) *Applier {
	logger := zapr.NewLogger(zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel)))
	applier := New(logger, m, cfgs, updater.NameGenerator(stubNameGenerator{name: "a"}))
	return applier
}

func createConfigs() *config.RepoConfiguration {
	return &config.RepoConfiguration{
		Repositories: []*config.Repository{
			{
				Name:               testQuayRepo,
				SourceRepo:         testGitHubRepo,
				SourceBranch:       "master",
				FilePath:           testFilePath,
				UpdateKey:          "test.image",
				BranchGenerateName: "test-branch-",
			},
		},
	}
}

type stubNameGenerator struct {
	name string
}

func (s stubNameGenerator) PrefixedName(p string) string {
	return p + s.name
}
