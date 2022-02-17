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

```shell
$ export GIT_USERNAME=my-user GIT_AUTH_TOKEN=my_token
$ ./yaml-updater update --file-path service-a/deployment.yaml --change-source-name my-docker-code-repo --source-repo my-org/my-change-target-repo --source-branch master --new-value quay.io/myorg/my-image:v1.1.0 --update-key spec.template.spec.containers.0.image --branch-generate-name gitops- 
```

This would update a file `service-a/deployment.yaml` in a GitHub repository at `my-org/my-change-target-repo`, changing the `spec.template.spec.containers.0.image` key in the file to `quay.io/myorg/my-image:v1.1.0`, the PR will indicate that this is an update from `my-docker-code-repo`.
If you want to do the same for some of the other supported git services, set GIT_DRIVER: `GIT_DRIVER=bitbucketcloud` or `GIT_DRIVER=gitlab`

If you need to access to private GitLab or GitHub installation, you can provide the `--api-endpoint` flag (or use the corresponding `GIT_API_ENDPOINT` env variable)

### Use yaml configuration

This allows one to simplify calling the cli or to apply the same value change to multiple files or repositories. For example, for CI/CD pipelines it's simpler to write most of the update command flags as configuration, and provide it as a list of repository details to target changes, which as mentioned also supports targeting multiple repositories or files (if the repository details are the same).
For example, the above section cli command could be simplified, so that the configuration would look like

```yaml
repositories:
  - name: my-docker-code-repo
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

The yaml configuration can be a relative or absolute path and be set with the `--config-path` flag or the `GIT_CONFIG_PATH` env var, and defaults to '.yaml-updater.yaml' when not set.

Do note that, when using the configuration yaml, neither `--username`, `--driver` or `--auth-token` have a corresponding configuration field, as these are globally set and not particular to any repository.
Likewise, every root or update command flag can be set as an env var of the form `GIT_` + the flag name with the dash replaced by an underscore and in uppercase. In the case of these flags, they were passed as env vars as they'll most likely be reused, but if not, call the updater and set them as flags instead.
No need to set the driver if using `github`, which is the default.


### Important: Updating the sourceBranch directly

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
