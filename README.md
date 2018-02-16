# kube-deploy

Welcome to kube-deploy, an opinionated but friendly deployment tool for Kubernetes!

## Overview

* Easy developer-focused command-line tool.
* Utilizes templating kubernetes consul-template and vault to create secure and complex deployments that fit almost any scenerio.
* Canary deployments, easy rollback and scaling
* Support for multiple clusters
* Works for teams of any size

## Commands 

### Context
    - 'name'                Prints the full path of the docker image that `kube-deploy` would currently build and roll out.
    - 'environment'         Prints the current environment/namespace being considered - one of 'production', 'staging', or 'development' - unless overridden.
    - 'cluster'             Prints the name of the cluster to be rolled out to - 'production' for the 'production' and 'staging' environments, 'development' otherwise.

### Building
    - 'build'               Builds a Docker image, runs the build tests, and pushes the image to the remote repository.
    - 'make'                An alias for 'build'.
    - 'test'                Makes a build and runs the build tests, but does not push the build.
    - 'testonly'            Runs the tests without making a build - only use if you're certain you haven't changed anything since the last build.
    - 'list-tags'           Prints a list of available docker tags in the remote repository that match the current git branch (Google Cloud Registry only).

### Rolling Out
    - 'lock'                Writes the lockfile (prevents others from starting a deployment) for this project without starting a deployment.
    - 'lock-all'            Writes the lockfile (prevents others from starting a deployment) for ALL projects.
    - 'rollback'            Immediately rolls back to the previous release.
    - 'start-rollout'       Starts a new rollout.
    - 'status'              Checks the lockfile to see if anyone is currently rolling out from this machine.
    - 'unlock'              Removes the lockfile, if it was created from the 'lock' command.

### Kubernetes commands
    - 'active-deployments'  Lists the Deployments currently associated with this project and branch, as well as their replica count and creation date.
    - 'rolling-restart'     Will create a new ReplicaSet of the same image, to gradually restart all pods for the Deployment.
    - 'scale'               Scales the current deployment for this project and branch to the provided number of pods.

## Workflow

The primary workflow of `kube-deploy` involves the following steps:
- Parsing the git information and `deploy.yaml` file for configuration data
- Build and test:
    - Building a docker image
    - Running tests against the newly-built docker image
    - Pushing the docker image to remote
- Kubernetes rollout:
    - Templating yaml files using consul-template
    - Creating new objects in Kubernetes from these yaml files
    - Starting 1 pod of the new deployment, and wait for a go-ahead (canary point)
    - Starts the desired number of new pods alongside the old pods, thus the new code receiving roughly 50% of traffic (second canary point)
    - Scale down the old deployment to zero pods, giving the new code 100% of traffic (last canary point)

## Opinions

`kube-deploy` is pretty opinionated about it's environment, but the rules are simple.

- Git and Docker:
    - The `master` branch is for the `staging` environment; the `acceptance` branch is for the `acceptance` environment; the `production` branch is for the `production` environment; all other branches are for the `development` environment
- Kubernetes:
    - The `development` cluster lives on its own, and has a Namespace called `development`
    - The `staging` environment is part of the `production` Kubernetes cluster (but lives in a Namespace called `staging`)
    - The `acceptance` environment is part of the `production` Kubernetes cluster (but lives in a Namespace called `acceptance`)
    - Only `Deployment` types are supported currently (not `StatefulSet`, `DaemonSet`, `Jobs`, etc)

## Configuration

`kube-deploy` depends on a `deploy.yaml` file in the root directory of your project. The rough structure of this `deploy.yaml` file is:

    dockerRepository:
        developmentRepositoryName:  ""
        productionRepositoryName: ""
        registryRoot: ""
    application:
        name: ""
        version: ""
        packageJSON: bool (uses a 'package.json' file to override name and version)
        kubernetesTemplate: (see below for details)
            branchVariables: { branchName: [] }
            globalVariables: []
    tests:
        - name: ""
          type: ""
          dockerArgs: ""
          dockerCommand: ""
          commands: []

Most of the details of this configuration is explained elsewhere in this README.

## Docker Naming Conventions

`kube-deploy` names its docker images in the following format:

    registryRoot/repositoryName/applicationName:version-branch-gitSHA

So, for example, assuming the following variables in a `deploy.yaml`:

    dockerRepository:
        registryRoot: private-docker.company.com
        developmentRepositoryName: dev-builds
        productionRepositoryName: prod-builds
    application:
        name: great-api
        version: 1.0.3

A build on the `feature/amazing-new-idea` branch at commit SHA '7b7f73' would yield the docker image named:

    private-docker.company.com/dev-builds/great-api:1.0.3-feature-amazing-new-idea-7b7f73

When that feature gets merged to master, creating a new commit SHA of '198edc0', the resulting docker image would be called:

    private-docker.company.com/prod-builds/great-api:1.0.3-master-198edc0

Thus, every docker image can be tied to an exact point in time in the git history. This also means that one can rollback or roll forward to any arbitrary point in the git history.

The combination of the application name, the branch name, and the commit hash, is referred to by `kube-deploy` as the 'Release Name', and is also used in the Kubernetes configuration. In this example, the release name would be:

    great-api-1.0.3-master-198edc0


## Building and Pushing

### Running Tests

The test sets are defined in the format:

    tests:
    - name: testSet name
      dockerArgs: the arguments that will be passed to docker run. `-d` is very often useful
      dockerCommand: Optional - an override command passed to `docker run`
      type: One of [ `in-external-container` (default), `in-test-container`, `on-host`, `host-only` ]
      commands:
      - array of commands (eg. `curl localhost:3000`, or `cat start.log` or `bash -c "curl localhost:3000 | grep 'teststring'"`)

Test types:
- `in-external-container` (default): Starts the test container, then starts another container to run the tests in the same network as the test container. Runs `docker run --rm --network container:<TEST_CONTAINER> <IMAGE> <COMMAND>`, thus starting a new container for each command. 
- `in-test-container`: Starts the test container, then runs the commands inside that container via `docker exec`.
- `on-host`: Starts the test container, then runs the commands on the host.
- `host-only`: Runs the commands on the host without starting a test container. Useful for things like `docker-compose up -d` to start up all dependencies and leave them up for the other testsets, then use another `host-only` testset at the end for `docker-compose down`.


An example test set configuration looks like this:

    tests:
    - name: Test container can start
      dockerArgs: -d
      type: on-host # Will start the container, then run the following commands on host
      commands:
      - bash -c "docker ps | grep 'test'"
    - name: Test that container can respond to ping
      dockerArgs: -d -p 3000:3000 -e ENVIRONMENT=development
      type: in-external-container
      commands:
      - curl --quiet localhost:3000
    - name: Run the test scripts
      dockerArgs: -p 3000:3000
      commands:
      - sh test.sh
      - npm test

There's even a `deploy.yaml` for `kube-deploy`, which tests that the source code for this project can build and run.

### Pushing to Remote

You must be authenticated to your remote container registry (docker repository) in order to push the images you build.

For Docker Hub, use `docker login`.

For Google Container Registry, a few short steps can log you into the Docker remote. If running locally, you can run the following commands to authenticate the container registry (If your local machine is a Mac, you might have to disable "Securely store docker logins in macOS keychain" to make `docker-credential-gcr` work properly):

    gcloud components install docker-credential-gcr
    docker-credential-gcr configure-docker

If running on a machine inside Google Cloud, you can also run the following command to log in (replace "https://gcr.io" if using another region):

    docker login -u oauth2accesstoken -p "$(gcloud auth application-default print-access-token)" https://gcr.io

## Kubernetes Configuration

`kube-deploy` utilises [`consul-template`](https://github.com/hashicorp/consul-template) to interpolate variables into Kubernetes YAML configuration files.

Environment variables can be specificed in the `kube-deploy` configuration file. All environment variables are declared in bash-like environment variable statements (in the format `ENV_KEY=value`), and will be added to the environment before templating the file with consul-template.

These template variables can reference each other using Go string formatting - for example, `DOMAIN={{.APP_NAME}}.mycujoo.tv`.

There are two sets of variables that can be declared inside `application->kubernetesTemplate` (see example configuration for more information):

`branchVariables` is a map of arrays of environment variable statements, specified per git branch. It is possible to list multiple comma-seperated branch names.
```
branchVariables:
    production:
    - DOMAIN=thumbs.mycujoo.tv
    - INGRESS_CLASS=nginx-production
    master:
    - DOMAIN=thumbs.staging.mycujoo.tv
    else:
    - DOMAIN={{.KD_GIT_BRANCH}}.thumbs.dev.mycujoo.tv
    master,else:
    - INGRESS_CLASS=nginx-internal

```

`globalVariables` is an array of environment variables that will be consistent across all git branches.
```
globalVariables:
  - APP_NAME=thumbs
  - REPLICAS=4
```

Some freebie variables are included by `kube-deploy` for you to use in your Kubernetes YAML files, prepended with "KD". These can be used in the exact same way as the other template variables, both in the Kubernetes file using the `consul-template` syntax (like `{{ env "VAR_NAME" }}`) and inside other environment variables using Go templating syntax (like `DOMAIN={{.KD_GIT_BRANCH}}.{{.KD_KUBERNETES_NAMESPACE}}.mycujoo.tv`).

The "KD" freebie variables are:
- `KD_RELEASE_NAME` - the Release Name (see Docker Naming Conventions above) - made of the application name plus the Image Tag
- `KD_APP_NAME` - the application name plus the branch name; used as the selector for Services, etc so that pods from multiple Deployments can share a Service (but be discrete from Services associated with other git branches)
- `KD_KUBERNETES_NAMESPACE` - the Kubernetes namespace - either 'production', 'staging', or 'development' (unless overridden)
- `KD_GIT_BRANCH` - the current git branch
- `KD_IMAGE_FULL_PATH` - the full tag of the Docker image, including repository URL
- `KD_IMAGE_TAG` - Of the format: `version-gitbranch-gitSHA`

The branch-speciifc variables are parsed first, which means that the `globalVariables` can reference values from `branchVariables`, but not the other way around. Both `globalVariables` and `branchVariables` can reference the "KD" freebie variables.

### Usage

To use these in your Kubernetes config file, use the `consul-template` syntax for environment variable interpolation. In practice, this might look like:

    metadata:
        name: {{ env "APP_NAME" }}
        namespace: {{ env "NAMESPACE" }}

### Vault

An advantage of using `consul-template` over any other Go-style string templating is the ability to interpolate secrets from Vault (usually into a Kubernetes Secrets YAML file).

Example:
```
data:
  ACCESS_KEY_ID: "{{ with secret "secret/access_key_id" }}{{ .Data.value | base64Encode }}{{ end }}"
```

This relies on a valid `$VAULT_ADDR` and `$VAULT_TOKEN` being set in the user environment, outside of `kube-deploy`.

## Doing a Rollout

For a normal rollout, first check out the repository to the branch you wish to deplot, and start the process by running `kube-deploy start-rollout`. If you have already made and pushed a build for the current HEAD, `kube-deploy` will begin the deployment process immediately; if you have not made and pushed a build for the current HEAD, `kube-deploy` will prompt you to do so now.

`kube-deploy` will create a lockfile on the deployment server during deployments to staging and production, to prevent two people from deploying at the same time.

## Rollbacks

To do an instant rollback, run `kube-deploy rollback`. This will start up pods in the old Deployment, labelled `kubedeploy-rollback-target`. There will be one canary point, when the reverting pods come up (and should have roughly 50% of traffic) to check that the problem is resolving. If you proceed at the canary, the reverted Deployment will scale to zero.

The Deployment that was reverted will be left in place, marked with `kubedeploy-rollback-target`, so that running `kube-deploy rollback` will swap back to the "newer" Deployment. In case the rollback was uncessary and the issue was somewhere else, re-rolling back will make the most recent Deployment live again.

<!-- 
## Branch Name Mappings

    Branch Name     Kubernetes Namespace    Domain

    master          staging                 <project-subdomain>.staging.<domain-tld>
    production      production              <project-subdomain>.<domain-tld>
    *any other*     development             <git-SHA1>.<your-username>.<project-subdomain>.dev.<domain-tld>

For example, if I (Jason) deploy the current branch at SHA `8086b67` for the project `awesome-website` and the company 'mycujoo.tv', the Kubernetes resources will be created in the `development` namespace and I will be able to access any Ingresses at `8086b67.jason.awesome-website.dev.mycujoo.tv`. -->
