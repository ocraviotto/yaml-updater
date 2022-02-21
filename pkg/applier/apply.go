package applier

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ocraviotto/go-scm/scm"
	"github.com/ocraviotto/pkg/client"
	"github.com/ocraviotto/pkg/updater"
	"github.com/ocraviotto/yaml-updater/pkg/config"
)

// New creates and returns a new Applier.
func New(l logr.Logger, c client.GitClient, cfgs *config.RepoConfiguration, opts ...updater.UpdaterFunc) *Applier {
	return &Applier{configs: cfgs, log: l, updater: updater.New(l, c, opts...)}
}

// Applier can update a Git repo with an updated version of a file based on a
// RepositoryPushHook.
type Applier struct {
	configs *config.RepoConfiguration
	log     logr.Logger
	updater *updater.Updater
}

// UpdateRepositories takes a list of repositories (e.g. from config)
// and for each it calls UpdateRepository, returning on any detected error.
func (u *Applier) UpdateRepositories(ctx context.Context, newValue string) error {
	var result error
	for _, repo := range u.configs.Repositories {
		if res := u.UpdateRepository(ctx, repo, newValue); res != nil {
			u.log.Error(result, "Failed to update repository file", "repository", repo.SourceRepo, "file", repo.FilePath)
			result = res
		}
	}
	return result
}

// UpdateRepository does the job of fetching the existing file, optionally creating it if it does not exist,
// updating it, and then optionally creating a PR. It also supports file removal.
func (u *Applier) UpdateRepository(ctx context.Context, cfg *config.Repository, newValue string) error {
	signature := scm.Signature{}
	cuFunc := updater.UpdateYAML(cfg.UpdateKey, newValue)
	if cfg.RemoveKey {
		cuFunc = updater.RemoveYAMLKey(cfg.UpdateKey)
	}
	cs := cfg.Signature
	if cs != nil && cs.Name != "" && cs.Email != "" {
		signature.Name = cs.Name
		signature.Email = cs.Email
	}
	commitMsg := cfg.CommitMsg
	if commitMsg == "" {
		commitMsg = "Automatic update because of GitOps yaml update/removal"
	}
	ci := updater.CommitInput{
		Repo:               cfg.SourceRepo,
		Filename:           cfg.FilePath,
		Branch:             cfg.SourceBranch,
		BranchGenerateName: cfg.BranchGenerateName,
		CreateMissing:      cfg.CreateMissing,
		RemoveFile:         cfg.RemoveFile,
		CommitMessage:      commitMsg,
		Signature:          signature,
	}
	newBranch, err := u.updater.ApplyUpdateToFile(ctx, ci, cuFunc)
	if err != nil {
		u.log.Error(err, "failed to get file from repo")
		return err
	}
	u.log.Info("updated branch with value", "value", newValue, "branch", newBranch)

	// If we modified the original branch...
	if newBranch == cfg.SourceBranch {
		return nil
	}

	pullRequestInput := updater.PullRequestInput{
		Title:        "Automated image update",
		Body:         fmt.Sprintf("Automated update from %q", cfg.Name),
		Repo:         cfg.SourceRepo,
		NewBranch:    newBranch,
		SourceBranch: cfg.SourceBranch,
	}

	pr, err := u.updater.CreatePR(ctx, pullRequestInput)
	if err != nil {
		return fmt.Errorf("failed to create pull request in repo %s: %w", cfg.SourceRepo, err)
	}
	u.log.Info("created PullRequest", "link", pr.Link)
	return nil
}
