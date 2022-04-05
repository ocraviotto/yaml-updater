# yaml-updater

This is a small tool for updating YAML files, either by updating a key value (with the option to create the yaml file if it does not exist), or removing a key (or a whole file).
The goal is to enable automating commits (and optionally PR creation) in a GitOps flow.

The difference between this and the forked [image-update](https://github.com/gitops-tools/image-updater) repository can be summed up as:

1. Augmenting the scope of actions to be a yaml key/file updater, rather than simply limiting the tool to updating an image only (hence the name change)
2. Modified libraries to allow for the above changes, and to fully support BitBucket
3. Removal of the http and pubsub commands (limited cloud support for the pubsub, security concerns with the webhook). If you need/want that, make sure to check upstream

## Command-line tool

The original code used to support additional commands, but as mentioned above, they were removed. However, the update functionality is still located under the update command and additional commands may be added in the future.

To see all existing options:

```shell
$ ./yaml-updater update --help
```

### Update a yaml via flags only

> **NOTE:** For more complex (or multiple) targets, it is recommended to use a "repositories" configuration file ([see below](#use-yaml-configuration)).

```shell
$ export GIT_USERNAME=my-user GIT_AUTH_TOKEN=my_token
$ ./yaml-updater update --file-path service-a/deployment.yaml --change-source-name my-docker-code-repo --source-repo my-org/my-change-target-repo --source-branch master --update-key spec.template.spec.containers.0.image --branch-generate-name gitops- --new-value quay.io/myorg/my-image:v1.1.0
```

This would create a new branch `gitops-[random]` from current master HEAD in a GitHub repository `my-org/my-change-target-repo`, and update the file `service-a/deployment.yaml` by changing the `spec.template.spec.containers.0.image` key in the file to `quay.io/myorg/my-image:v1.1.0`, creating a PR against master that would include a message `Automatic update from my-docker-code-repo`.
If you want to do the same for some of the other supported git services, set GIT_DRIVER: `GIT_DRIVER=bitbucketcloud` or `GIT_DRIVER=gitlab`

If you need to access to private GitLab or GitHub installation, you can provide the `--api-endpoint` flag (or use the corresponding `GIT_API_ENDPOINT` env variable)

To disable PR creation and commit directly to the `--source-branch` value, simply pass `--disable-pr-creation` (and make sure the source branch can be committed directly to).
For additional details, see below [Important: Updating the sourceBranch directly](#important-updating-the-sourcebranch-directly).

### Use yaml configuration

The repositories config allows one to simplify calling the cli, or to apply the same value change to multiple files, branches or repositories. For example, for CI/CD pipelines it's simpler to write most of the update command flags as configuration, and provide it as a list of repository details to target changes, which as mentioned, also supports targeting multiple repositories or files (if the repository details are the same).
For example, the above section cli command could be simplified, so that the configuration would look like

```yaml
repositories:
  my-repository:
    name: my-docker-code-repo
    sourceRepo: my-org/my-change-target-repo
    sourceBranch: master
    filePath: service-a/deployment.yaml
    updateKey: spec.template.spec.containers.0.image
    branchGenerateName: gitops-
```
and the command could be as simple as

```shell
$ export GIT_USERNAME=my-user GIT_AUTH_TOKEN=my_token
$ ./yaml-updater update --new-value quay.io/myorg/my-image:v1.1.0
```

The repositories yaml configuration itself can be a relative or absolute path and be set with the `--config-path` flag or the `GIT_CONFIG_PATH` env var, and defaults to '.yaml-updater.yaml' when not set. 

> Do note that, when using the configuration yaml, neither `--username`, `--driver` or `--auth-token` have a corresponding configuration field, as these are globally set and not particular to any repository.
> Likewise, every root or update command flag can be set as an env var of the form `GIT_` + the flag name with the dash replaced by an underscore and in uppercase. In the case of these flags, they were passed as env vars as they'll most likely be reused, but if not, call the updater and set them as flags instead.
> No need to set the driver if using `github`, which is the default.

In the above example, the `--new-value` `quay.io/myorg/my-image:v1.1.0` would be applied just as in the CLI example above. The main difference with this approach is that a project could configure multiple targets (i.e. different repositories, branches, files or even yaml keys) updated with the same value in one go (as long as the git service and the credentials are common!). 

For example, a config file could look like:

```yaml
repositories:
  prod:
    disabled: true
    name: my-docker-code-repo
    sourceRepo: my-org/my-change-target-repo
    sourceBranch: master
    filePath: service-a/deployment.yaml
    updateKey: spec.template.spec.containers.0.image
    branchGenerateName: gitops-
  dev:
    name: my-docker-code-repo
    sourceRepo: my-org/my-change-target-repo
    sourceBranch: dev
    filePath: service-a/deployment.yaml
    updateKey: spec.template.spec.containers.0.image
    disablePRCreation: true
```

And then call the updater regularly like:

```shell
$ export GIT_USERNAME=my-user GIT_AUTH_TOKEN=my_token
$ ./yaml-updater update --new-value [my-new-image-value]
```

but when targeting production, something like

```shell
$ export GIT_USERNAME=my-user GIT_AUTH_TOKEN=my_token
$ ./yaml-updater update --new-value [my-new-image-value] --only prod
```

This would ensure that development changes are automatically committed, not creating PRs to master (first command) while creating a PR for master and not modifying dev if executing the second.

> A third variation could be 
> ```shell
> $ export GIT_USERNAME=my-user GIT_AUTH_TOKEN=my_token
> $ ./yaml-updater update --new-value [my-new-image-value] --override-repositories "prod,dev" --disabled=false
> ```
> which would cause both to be enabled (overrides the `disable` key in both so effectively enabling them) and so it'd commit directly to dev and create a PR for master from branch `gitops-[random]`. 


For more options and uses, see the section below on [Yaml configuration overrides](#yaml-configuration-overrides).


### Yaml configuration overrides

If the config file exists and has repositories, command line flags can operate as overrides provided that the action is explicitly enabled (applied to **ALL** targeted repositories, though only effective on enabled repositories, unless the override is the `--disabled=false` flag). yaml-updater provides 3 flags for that:

1. `--override-all` (Boolean): As indicated by the name, if passes, any repository config passed via CLI will override every configured repository defined in config with the given value. 
2. `--override-repositories` (String): This is a comma separated list of matching repository config keys so that repository config passed via CLI will override every matched configured repository with the given value.
3. `--only` (String): This is also a comma separated list of matching repository config keys so that repository config passed via CLI will override every matched configured repository with the given value AND every matching repository will be enabled plus any NON matching key will be removed and not used at all.

> :warning: Using `only` has a slightly different semantics than the other 2 flags in that it will disable (remove) any repository not given in the `--only` flag.

> See `yaml-updater update --help` for additional details. 


### Important: Updating the sourceBranch directly

By default, changes are not committed directly, by via a PR with a branch whose name is prefixed with the value of `branchGenerateName` (`gitops-` by default).
To commit directly, make sure to either set `disablePRCreation` to `true` in the spec, or pass the corresponding `--disable-pr-creation` flag to the command.
*NOTE:* This is meant to work with `--override-repository [repo-config-key]`. If no such flag is passed, and `yaml-updater` detects more than one repository being updated, the command will fail, unless the `-override-all` (Boolean) flag is passed.

If no value is provided for `branchGenerateName`, then the `sourceBranch` will be updated directly, this means that if you use `master`, there will not be a PR created, the commit will be applied directly, and the token needs to authorize access to push a change directly to `master`.

## Building

A `Dockerfile` is provided for building a container, but otherwise:

```shell
$ go build ./cmd/yaml-updater
```

## Docker images

Images are available at `oscarc/yaml-updater` and there should soon be additional tags matching this repository tags.

## Testing

```shell
$ go test ./...
```
