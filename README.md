# kube-deploy

Welcome to kube-deploy, a deployment tool for Kubernetes!

## Commands 

### Building
    - 'build'               Builds a Docker image, runs the build tests, and pushes the image to the remote repository.
    - 'make'                An alias for 'build'.
    - 'test'                Makes a build and runs the build tests, but does not push the build.

### Rolling Out
    - 'list-deployments'    Prints a list of recent Kubernetes deployments for the current branch of this project.
    - 'list-tags'           Prints a list of available docker tags in the remote repository that match the current git branch (Google Cloud Registry only).
    - 'lock'                Writes the lockfile (prevents others from starting a deployment) for this project without starting a deployment.
    - 'lock-all'            Writes the lockfile (prevents others from starting a deployment) for ALL projects.
    - 'rollback'            With no args, immediately rolls back to the previous release. A Docker tag may optionally be provided, in which case the deployment will be rolled back to the specified tag (with no canary points).
    - 'start-rollout'       Starts a new rollout.
    - 'unlock'              Removes the lockfile, if it was created from the 'lock' command.

### Creating and Removing Resources
    - 'create'              Will create the Kubernetes resources located in the `kubernetes/config` directory in the project, with the namespace and domain as specified in [Branch Name Mappings](#Branch-Name-Mappings).
    - 'teardown'            Will remove all specified resources for the current branch from Kubernetes. Only possible for the 'development' namespace (ie. not possible for staging and production).

### Kubernetes commands
    - 'rolling-restart'     Will create a new ReplicaSet of the same image, to gradually restart all pods for the Deployment.
    - 'scale'               Scales the current deployment for this project and branch to the provided number of pods.

## Building and Pushing

### Running Tests

The test sets are defined in the format:

    tests:
    - name: <testSet name>
      dockerArgs: <the arguments that will be passed to docker run. `-d` is very often useful>
      commands:
      - array of
      - shell commands that
      - will be executed after the
      - test container has started

There's even a `deploy.yaml` for `kube-deploy`, which tests that the source code can build and run.

An example test set configuration looks like this:

    tests:
    - name: Test container can start
      dockerArgs: -d
      commands:
      - docker ps | grep 'test'
    - name: Test that container can respond to ping
      dockerArgs: -d -p 3000:3000 -e ENVIRONMENT=development
      commands:
      - curl --quiet localhost:3000
    - name: Run the test scripts
      dockerArgs: -p 3000:3000
      commands:
      - sh test.sh
      - npm test

The commands can be anything that is runnable in your local shell, including `docker-compose` to set up a more complex test environment.

### Pushing to Remote

If you're logged in to your Docker remote repository, you don't need to set up any additional configuration to push your Docker image.

If you're using Google Container Registry for `kube-deploy` images, a few short steps can log you into the Docker remote. If `kube-deploy` is being run locally, it will prompt you to run the following commands to authenticate the container registry (If your local machine is a Mac, you might have to disable "Securely store docker logins in macOS keychain" to make `docker-credential-gcr` work properly):

    gcloud components install docker-credential-gcr
    docker-credential-gcr configure-docker

If running on a machine inside Google Cloud, `kube-deploy` will prompt you to run the following command to log in:

    docker login -u oauth2accesstoken -p "$(gcloud auth application-default print-access-token)" https://gcr.io

## Kubernetes Configuration

`kube-deploy` utilises [`consul-template`](https://github.com/hashicorp/consul-template) to interpolate variables into Kubernetes YAML configuration files.

Environment variables can be specificed in the `kube-deploy` configuration file. All environment variables are declared in bash-like environment variable statements (in the format `ENV_KEY=value`), and will be added to the environment before templating the file with consul-template. These template variables can reference each other using Go string formatting - for example, `DOMAIN={{.APP_NAME}}.mycujoo.tv`.
There are two sets of variables that can be declared inside `application->kubernetesTemplate` (see example configuration for more information):
- `branchVariables` is a map of arrays of environment variable statements, specified per git branch.
```
branchVariables:
    production:
    - DOMAIN=thumbs.mycujoo.tv
    master:
    - DOMAIN=thumbs.staging.mycujoo.tv
    else:
    - DOMAIN={{.KD_GIT_BRANCH}}.thumbs.dev.mycujoo.tv
```
- `globalVariables` is an array of environment variables that will be consistent across all git branches.
```
globalVariables:
  - APP_NAME=thumbs
  - REPLICAS=4
```

Some freebie variables are included by `kube-deploy` for you to use in your Kubernetes YAML files, prepended with "KD". These can be used in the exact same way as the other template variables, both in the Kubernetes file using the `consul-template` syntax and inside other environment variables (like `DOMAIN={{.KD_GIT_BRANCH}}.{{.KD_ENVIRONMENT_NAME}}.mycujoo.tv`).

The "KD" freebie variables are:
- `KD_GIT_BRANCH` - the current git branch
- `KD_ENVIRONMENT_NAME` - ('production' for master and production branch names, 'development' otherwise)
- `KD_IMAGE_FULL_PATH` - the full tag of the Docker image, including repository URL

The branch-speciifc variables are parsed first, which means that the `globalVariables` can reference values from `branchVariables`, but not the other way around. Both `globalVariables` and `branchVariables` can reference the "KD" freebie variables.

### Usage

To use these in your Kubernetes config file, use the `consul-template` syntax for environment variable interpolation. In practice, this might look like:

    metadata:
        name: {{ env "APP_NAME" }}
        namespace: {{ env "NAMESPACE" }}





## Doing a Rollout

For a normal rollout, first check out the repository to the branch you wish to deplot, and start the process by running `kube-deploy start-rollout`. If you have already made and pushed a build for the current HEAD, `kube-deploy` will begin the deployment process immediately; if you have not made and pushed a build for the current HEAD, `kube-deploy` will prompt you to do so now.

`kube-deploy` will create a lockfile on the deployment server during deployments to staging and production, to prevent two people from deploying at the same time.

## Branch Name Mappings

    Branch Name     Kubernetes Namespace    Domain

    master          staging                 <project-subdomain>.staging.<domain-tld>
    production      production              <project-subdomain>.<domain-tld>
    *any other*     development             <git-SHA1>.<your-username>.<project-subdomain>.dev.<domain-tld>

For example, if I (Jason) deploy the current branch at SHA `8086b67` for the project `awesome-website` and the company 'mycujoo.tv', the Kubernetes resources will be created in the `development` namespace and I will be able to access any Ingresses at `8086b67.jason.awesome-website.dev.mycujoo.tv`.

