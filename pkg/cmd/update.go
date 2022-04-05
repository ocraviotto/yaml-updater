package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ocraviotto/pkg/client"
	"github.com/ocraviotto/yaml-updater/pkg/applier"
	"github.com/ocraviotto/yaml-updater/pkg/config"
)

func makeUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update a repository configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			var repositories, pRepositories *config.RepoConfiguration
			logger, _ := zap.NewProduction()
			l := zapr.NewLogger(logger)
			defer func() {
				_ = logger.Sync() // flushes buffer, if any
			}()
			scmClient, err := createClientFromViper()
			if err != nil {
				return fmt.Errorf("failed to create a git driver: %s", err)
			}
			c := viper.GetString("config-path")
			if c != "" {
				repositories, err = config.Load(c)
				if err != nil {
					l.Info("Error trying to read repositories yaml config from file", "content", repositories, "error", err)
				}
			}

			if repositories == nil {
				l.Info("No repositories from config. Using flags or env")
				applier := applier.New(l, client.New(scmClient), nil)
				return applier.UpdateRepository(context.Background(), configFromFlags(), viper.GetString("new-value"))
			}

			pRepositories, err = processConfigsAndOverrides(repositories)
			if err != nil {
				return fmt.Errorf("failing to update to to error: %s", err)
			} else if pRepositories == nil {
				return fmt.Errorf("failing to update as processing repositories config returned an empty list")
			}
			applier := applier.New(l, client.New(scmClient), pRepositories)
			return applier.UpdateRepositories(context.Background(), viper.GetString("new-value"))
		},
	}

	cmd.Flags().String(
		"new-value",
		"",
		"The value to set for the yaml key, e.g. org/repo that is being updated for a docker image. If empty, the value will be set to an empty string. "+
			"Applies to ALL enabled and set repositories",
	)
	logIfError(viper.BindPFlag("new-value", cmd.Flags().Lookup("new-value")))

	cmd.Flags().String(
		"config-path",
		".yaml-updater.yaml",
		"List of repository configurations to apply update to. If file exists, it takes precedence over repository details passed via flags",
	)
	logIfError(viper.BindPFlag("config-path", cmd.Flags().Lookup("config-path")))

	addConfigFlags(cmd)

	return cmd
}

func addConfigFlags(cmd *cobra.Command) {
	cmd.Flags().String(
		"change-source-name",
		"",
		"The value to set for the yaml key, e.g. org/repo that is being updated for a docker image. Required either via flag, env or yaml. "+
			"When set, either via flag or env, it overrides all repository configs from yaml",
	)
	logIfError(viper.BindPFlag("change-source-name", cmd.Flags().Lookup("change-source-name")))

	// allow both the above to be accessed either way
	viper.RegisterAlias("change-source-name", "image-repo")

	cmd.Flags().Bool(
		"disabled",
		false,
		"Used as an override with repositories from configs to disable and enable repositories via cli. Requires that the repository be declard in config",
	)
	logIfError(viper.BindPFlag("disabled", cmd.Flags().Lookup("disabled")))

	cmd.Flags().String(
		"only",
		"",
		"A single or comman separated list of keys of the repositories defined in configuration to update and use. By default all are enabled "+
			"unless they are explicitely disabled. If given, any repository not in the only list will be disabled and if in the list, enabled. "+
			"This is why it takes precedence over the repositories 'disabled' field. "+
			"NOTE: This is different than the override keys in that it will disable any repository not in the list",
	)
	logIfError(viper.BindPFlag("only", cmd.Flags().Lookup("only")))

	cmd.Flags().String(
		"override-repositories",
		"",
		"A single or comman separated list of keys of the repositories to override via cli parameters. If not given and 'repositories' has more than one repository, "+
			"the update command will fail, unless --override-all is passed. If the keys do not match one or more repositories, the command will fail (safe).",
	)
	logIfError(viper.BindPFlag("override-repositories", cmd.Flags().Lookup("override-repositories")))

	cmd.Flags().Bool(
		"override-all",
		false,
		"If there are more than a single repository, either --override-repositories OR --only is passed to apply the overrides to a particular repository, "+
			"or this bool needs to be set, else the update command will fail. If all are given, "+
			"--override-repositories takes precedence over --override-all, and --only over --override-repositories. "+
			"When set, any key passed via cli will set the value for all configured repositories.",
	)
	logIfError(viper.BindPFlag("override-all", cmd.Flags().Lookup("override-all")))

	cmd.Flags().Bool(
		"disable-pr-creation",
		false,
		"If set, PR creation will be disabled and the commit will be directly done to the branh especified by --source-branch or sourceBranch "+
			"(defaults to master). This flag works as any other Repository spec flag, allowing a user to set it in config, via cli override, "+
			"or applied to all repositories via cli override along --override-all",
	)
	logIfError(viper.BindPFlag("disable-pr-creation", cmd.Flags().Lookup("disable-pr-creation")))

	cmd.Flags().String(
		"source-repo",
		"",
		"Git repository to update e.g. org/repo. Required either via flag, env or yaml",
	)
	logIfError(viper.BindPFlag("source-repo", cmd.Flags().Lookup("source-repo")))

	cmd.Flags().String(
		"source-branch",
		"master",
		"Branch to fetch for updating",
	)
	logIfError(viper.BindPFlag("source-branch", cmd.Flags().Lookup("source-branch")))

	cmd.Flags().String(
		"file-path",
		"",
		"Path within the source-repo to update. Required either via flag, env or yaml. "+
			"When set, either via flag or env, it overrides all repository configs from yaml",
	)
	logIfError(viper.BindPFlag("file-path", cmd.Flags().Lookup("file-path")))

	cmd.Flags().String(
		"update-key",
		"",
		"JSON path within the file-path to update e.g. spec.template.spec.containers.0.image. Required either via flag, env or yaml",
	)
	logIfError(viper.BindPFlag("update-key", cmd.Flags().Lookup("update-key")))

	cmd.Flags().String(
		"branch-generate-name",
		"gitops-",
		"Prefix for naming automatically generated branches. A PR will be created by default with a branch prefixed with this value. "+
			"To disable PRs, pass the --disable-pr-creation flag or set disablePRCreation",
	)
	logIfError(viper.BindPFlag("branch-generate-name", cmd.Flags().Lookup("branch-generate-name")))

	cmd.Flags().Bool(
		"create-missing",
		true,
		"If source file does not exist, create it. The yaml key (and the full objects/arrays if nested) will be always created if missing. "+
			"When set, either via flag or env, it overrides all repository configs from yaml",
	)
	logIfError(viper.BindPFlag("create-missing", cmd.Flags().Lookup("create-missing")))

	cmd.Flags().Bool(
		"remove-key",
		false,
		"If set, the update-key needs to be removed instead of updated. Note, this only affects the final key or item key that would otherwise be updated. "+
			"When set, either via flag or env, it overrides all repository configs from yaml",
	)
	logIfError(viper.BindPFlag("remove-key", cmd.Flags().Lookup("remove-key")))

	cmd.Flags().Bool(
		"remove-file",
		false,
		"If set, instead of an update or add operation, this will execute a removal of target of file-path if it exists. "+
			"When set, either via flag or env, it overrides all repository configs from yaml",
	)
	logIfError(viper.BindPFlag("remove-file", cmd.Flags().Lookup("remove-file")))

	cmd.Flags().String(
		"committer-name",
		"",
		"The name of the commit message author. Required for bitbucket",
	)
	logIfError(viper.BindPFlag("committer-name", cmd.Flags().Lookup("committer-name")))

	cmd.Flags().String(
		"committer-email",
		"",
		"The email of the commit message author. Required for bitbucket",
	)
	logIfError(viper.BindPFlag("committer-email", cmd.Flags().Lookup("committer-email")))

	cmd.Flags().String(
		"commit-msg",
		"",
		"The message to use when creating the commit to change/create the source branch file. "+
			"When set, either via flag or env, it overrides all repository configs from yaml. "+
			"Defaults to \"Automatic update from [change-source-name]\"",
	)
	logIfError(viper.BindPFlag("commit-msg", cmd.Flags().Lookup("commit-msg")))
}

// omitting disabled, as it makes no sense here
func configFromFlags() *config.Repository {
	return &config.Repository{
		Name:               viper.GetString("change-source-name"),
		SourceRepo:         viper.GetString("source-repo"),
		SourceBranch:       viper.GetString("source-branch"),
		FilePath:           viper.GetString("file-path"),
		UpdateKey:          viper.GetString("update-key"),
		BranchGenerateName: viper.GetString("branch-generate-name"),
		RemoveKey:          viper.GetBool("remove-key"),
		RemoveFile:         viper.GetBool("remove-file"),
		CreateMissing:      viper.GetBool("create-missing"),
		CommitMsg:          viper.GetString("commit-msg"),
		DisablePRCreation:  viper.GetBool("disable-pr-creation"),
		Signature: &config.Signature{
			Name:  viper.GetString("committer-name"),
			Email: viper.GetString("committer-email"),
		},
	}
}

// processConfigsAndOverrides is used to set cli or env overrides over configuration
// from files
func processConfigsAndOverrides(configs *config.RepoConfiguration) (*config.RepoConfiguration, error) {
	var reposToOverride, overrideRepos, only []string

	repoKeys := configs.Keys()

	if viper.GetBool("override-all") {
		reposToOverride = repoKeys
	}

	if overrideReposVal := viper.GetString("override-repositories"); overrideReposVal != "" {
		overrideRepos = strings.Split(overrideReposVal, ",")
		reposToOverride = overrideRepos
	}

	if onlyVal := viper.GetString("only"); onlyVal != "" {
		only = strings.Split(onlyVal, ",")
		reposToOverride = only
		// Process all repositories when only is given
		// to remove any repo not in only
		for _, repo := range repoKeys {
			// If --only is set and repo not in only, it will be deleted
			// regardless of "disabled" value. Likewise, if disabled,
			// it will be enabled and used.
			var inOnly bool
			for _, r := range only {
				if repo == r {
					inOnly = true
				}
			}
			if len(only) > 0 && inOnly {
				configs.Repositories[repo].Disabled = false
			} else if len(only) > 0 {
				delete(configs.Repositories, repo)
			}
		}
	}

	for _, repo := range reposToOverride {

		// Check the keys exist in the map
		if _, ok := configs.Repositories[repo]; !ok {
			// Fail as the user intended something else
			return nil, fmt.Errorf("user given repository: %s does not exist in the current repositories config - failsafe exit\nAre you passing --only too?", repo)
		}

		// Allow user to override disabled value when not using '--only'
		if viper.IsSet("disabled") && len(only) == 0 {
			configs.Repositories[repo].Disabled = viper.GetBool("disabled")
		}

		// Prevent processing disabled repos
		if configs.Repositories[repo].Disabled {
			continue
		}

		// Overrrides if set
		if viper.IsSet("change-source-name") || viper.IsSet("image-repo") {
			configs.Repositories[repo].Name = viper.GetString("change-source-name")
		}
		if viper.IsSet("source-repo") {
			configs.Repositories[repo].SourceRepo = viper.GetString("source-repo")
		}
		if viper.IsSet("source-branch") {
			configs.Repositories[repo].SourceBranch = viper.GetString("source-branch")
		}
		if viper.IsSet("file-path") {
			configs.Repositories[repo].FilePath = viper.GetString("file-path")
		}
		if viper.IsSet("update-key") {
			configs.Repositories[repo].UpdateKey = viper.GetString("update-key")
		}
		if viper.IsSet("branch-generate-name") {
			configs.Repositories[repo].BranchGenerateName = viper.GetString("branch-generate-name")
		}
		if viper.IsSet("remove-key") {
			configs.Repositories[repo].RemoveKey = viper.GetBool("remove-key")
		}
		if viper.IsSet("remove-file") {
			configs.Repositories[repo].RemoveFile = viper.GetBool("remove-file")
		}
		if viper.IsSet("create-missing") {
			configs.Repositories[repo].CreateMissing = viper.GetBool("create-missing")
		}
		if viper.IsSet("commit-msg") {
			configs.Repositories[repo].CommitMsg = viper.GetString("commit-msg")
		}
		if viper.IsSet("disable-pr-creation") {
			configs.Repositories[repo].DisablePRCreation = viper.GetBool("disable-pr-creation")
		}
		if viper.IsSet("committer-name") {
			configs.Repositories[repo].Signature.Name = viper.GetString("committer-name")
		}
		if viper.IsSet("committer-email") {
			configs.Repositories[repo].Signature.Email = viper.GetString("committer-email")
		}
	}

	// Remove any other disabled repository not already removed
	for k, r := range configs.Repositories {
		if r.Disabled {
			delete(configs.Repositories, k)
		}
	}

	return configs, nil
}
