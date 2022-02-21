package cmd

import (
	"context"
	"fmt"

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
			var repositories *config.RepoConfiguration
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

			if repositories, err = config.Load(c); err != nil {
				l.Info("No repositories from config", "content", repositories, "error", err)
				applier := applier.New(l, client.New(scmClient), nil)
				return applier.UpdateRepository(context.Background(), configFromFlags(), viper.GetString("new-value"))
			}
			repositories = globalConfigsOverrides(repositories)
			applier := applier.New(l, client.New(scmClient), repositories)
			return applier.UpdateRepositories(context.Background(), viper.GetString("new-value"))
		},
	}

	cmd.Flags().String(
		"new-value",
		"",
		"The value to set for the yaml key, e.g. org/repo that is being updated for a docker image. If empty, the value will be set to an empty string",
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
		"image-repo",
		"",
		"The source code or image repository to identify from where the update originated. Deprecated, use \"change-source-name\"",
	)
	logIfError(viper.BindPFlag("image-repo", cmd.Flags().Lookup("image-repo")))

	cmd.Flags().String(
		"change-source-name",
		"",
		"The value to set for the yaml key, e.g. org/repo that is being updated for a docker image. Required either via flag, env or yaml",
	)
	logIfError(viper.BindPFlag("change-source-name", cmd.Flags().Lookup("change-source-name")))

	cmd.Flags().String(
		"source-repo",
		"",
		"Git repository to update e.g. org/repo. Required either via flag, env or yaml",
	)
	logIfError(viper.BindPFlag("source-repo", cmd.Flags().Lookup("source-repo")))

	cmd.Flags().String(
		"source-branch",
		"master",
		"Branch to fetch for updating. When set, either via flag or env, it overrides config from yaml",
	)
	logIfError(viper.BindPFlag("source-branch", cmd.Flags().Lookup("source-branch")))

	cmd.Flags().String(
		"file-path",
		"",
		"Path within the source-repo to update. Required either via flag, env or yaml. When set, either via flag or env, it overrides config from yaml",
	)
	logIfError(viper.BindPFlag("file-path", cmd.Flags().Lookup("file-path")))

	cmd.Flags().String(
		"update-key",
		"",
		"JSON path within the file-path to update e.g. spec.template.spec.containers.0.image. Required either via flag, env or yaml. When set, either via flag or env, it overrides config from yaml",
	)
	logIfError(viper.BindPFlag("update-key", cmd.Flags().Lookup("update-key")))

	cmd.Flags().String(
		"branch-generate-name",
		"gitops-",
		"Prefix for naming automatically generated branches. If empty, the source-branch will be directly committed changes to",
	)
	logIfError(viper.BindPFlag("branch-generate-name", cmd.Flags().Lookup("branch-generate-name")))

	cmd.Flags().Bool(
		"create-missing",
		true,
		"If source file does not exist, create it. The yaml key (and the full objects/arrays if nested) will be always created if missing. When set, either via flag or env, it overrides config from yaml",
	)
	logIfError(viper.BindPFlag("create-missing", cmd.Flags().Lookup("create-missing")))

	cmd.Flags().Bool(
		"remove-key",
		false,
		"If set, the update-key needs to be removed instead of updated. Note, this only affects the final key or item key that would otherwise be updated. When set, either via flag or env, it overrides config from yaml",
	)
	logIfError(viper.BindPFlag("remove-key", cmd.Flags().Lookup("remove-key")))

	cmd.Flags().Bool(
		"remove-file",
		false,
		"If set, instead of an update or add operation, this will execute a removal of target of file-path if it exists. When set, either via flag or env, it overrides config from yaml",
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
		"Automatic update because of GitOps yaml update/removal",
		"The message to use when creating the commit to change/create the source branch file",
	)
	logIfError(viper.BindPFlag("commit-msg", cmd.Flags().Lookup("commit-msg")))
}

// Deprecate image-repo
func setUpdateName() string {
	v := viper.GetString("change-source-name")
	if v != "" {
		return v
	}
	return viper.GetString("image-repo")
}

func configFromFlags() *config.Repository {
	return &config.Repository{
		Name:               setUpdateName(),
		SourceRepo:         viper.GetString("source-repo"),
		SourceBranch:       viper.GetString("source-branch"),
		FilePath:           viper.GetString("file-path"),
		UpdateKey:          viper.GetString("update-key"),
		BranchGenerateName: viper.GetString("branch-generate-name"),
		RemoveKey:          viper.GetBool("remove-key"),
		RemoveFile:         viper.GetBool("remove-file"),
		CreateMissing:      viper.GetBool("create-missing"),
		CommitMsg:          viper.GetString("commit-msg"),
		Signature: &config.Signature{
			Name:  viper.GetString("committer-name"),
			Email: viper.GetString("committer-email"),
		},
	}
}

// globalConfigsOverrides is used to set global cli or env overrides over configuration
// from files
func globalConfigsOverrides(configs *config.RepoConfiguration) *config.RepoConfiguration {
	for _, repo := range configs.Repositories {
		if viper.IsSet("remove-key") {
			repo.RemoveKey = viper.GetBool("remove-key")
		}
		if viper.IsSet("remove-file") {
			repo.RemoveFile = viper.GetBool("remove-file")
		}
		if viper.IsSet("create-missing") {
			repo.CreateMissing = viper.GetBool("create-missing")
		}
		if viper.IsSet("file-path") {
			repo.FilePath = viper.GetString("file-path")
		}
		if viper.IsSet("update-key") {
			repo.FilePath = viper.GetString("update-key")
		}
		if viper.IsSet("source-branch") {
			repo.FilePath = viper.GetString("source-branch")
		}
	}
	return configs
}
