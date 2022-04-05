package cmd

import (
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	driverFlag      = "driver"
	apiEndpointFlag = "api-endpoint"
	authTokenFlag   = "auth-token"
	usernameFlag    = "username"
	insecureFlag    = "insecure"
)

func init() {
	cobra.OnInitialize(initConfig)
}

func logIfError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func makeRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "yaml-updater",
		TraverseChildren: true,
		Short:            "Update YAML files in a Git service, with optional automated Pull Requests",
	}

	cmd.PersistentFlags().String(
		driverFlag,
		"github",
		"go-scm driver name to use e.g. github, gitlab, bitbucket, bitbucketcloud",
	)
	logIfError(viper.BindPFlag(driverFlag, cmd.PersistentFlags().Lookup(driverFlag)))

	cmd.PersistentFlags().String(
		authTokenFlag,
		"",
		"The token or password to authenticate requests to your Git service",
	)
	logIfError(viper.BindPFlag(authTokenFlag, cmd.PersistentFlags().Lookup(authTokenFlag)))
	cmd.PersistentFlags().String(
		usernameFlag,
		"",
		"The username to authenticate requests to your Git service. Must be given for bitbucketcloud and if using pass. This has nothing to do with the committer name",
	)
	logIfError(viper.BindPFlag(usernameFlag, cmd.PersistentFlags().Lookup(usernameFlag)))

	cmd.PersistentFlags().String(
		apiEndpointFlag,
		"",
		"The API endpoint to communicate with private GitLab/GitHub/BitBucket installations",
	)
	logIfError(viper.BindPFlag(apiEndpointFlag, cmd.PersistentFlags().Lookup(apiEndpointFlag)))

	cmd.PersistentFlags().Bool(
		insecureFlag,
		false,
		"Allow insecure server connections when using SSL",
	)
	logIfError(viper.BindPFlag(insecureFlag, cmd.PersistentFlags().Lookup(insecureFlag)))

	cmd.AddCommand(makeUpdateCmd())

	return cmd
}

func initConfig() {
	viper.SetEnvPrefix("git")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
}

// Execute is the main entry point into this component.
func Execute() {
	if err := makeRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
