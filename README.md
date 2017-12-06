# kube-deploy

Welcome to kube-deploy, a deployment tool for Kubernetes!

## Commands 

### Building
    - 'build'               Builds a Docker image, runs the build tests, and pushes the image to the remote repository.
    - 'make'                An alias for 'build'.
    - 'test'                Makes a build and runs the build tests, but does not push the build.

### Rolling Out
    - 'list-deployments'    Prints a list of recent Kubernetes deployments for the current branch of this project.
    - 'list-tags'           Prints a list of available docker tags in the remote repository that match the current git branch.
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

kubectl patch deployment thumbs --namespace=development -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"rolling-restart\":\"`date +'%s'`\"}}}}}"

kubectl patch deployment mycujoo-api --namespace=production -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"rolling-restart\":\"`date +'%s'`\"}}}}}" && kubectl rollout status --namespace=production deployment/mycujoo-api

## Building and Pushing

### Pushing to Remote

If you're logged in to your Docker remote repository, you don't need to set up any additional configuration to push your Docker image.

If you're using Google Container Registry for `kube-deploy` images, a few short steps can log you into the Docker remote. If `kube-deploy` is being run locally, it will prompt you to run the following commands to authenticate the container registry:

    gcloud components install docker-credential-gcr
    docker-credential-gcr configure-docker

If running on a machine inside Google Cloud, `kube-deploy` will prompt you to run the following command to log in:

    docker login -u oauth2accesstoken -p "$(gcloud auth application-default print-access-token)" https://gcr.io


## Doing a Rollout

For a normal rollout, first check out the repository to the branch you wish to deplot, and start the process by running `kube-deploy start-rollout`. If you have already made and pushed a build for the current HEAD, `kube-deploy` will begin the deployment process immediately; if you have not made and pushed a build for the current HEAD, `kube-deploy` will prompt you to do so now.

`kube-deploy` will create a lockfile on the deployment server during deployments to staging and production, to prevent two people from deploying at the same time.

## Branch Name Mappings

    Branch Name     Kubernetes Namespace    Domain

    master          staging                 <project-subdomain>.staging.<domain-tld>
    production      production              <project-subdomain>.<domain-tld>
    *any other*     development             <git-SHA1>.<your-username>.<project-subdomain>.dev.<domain-tld>

For example, if I (Jason) deploy the current branch at SHA `8086b67` for the project `awesome-website` and the company 'mycujoo.tv', the Kubernetes resources will be created in the `development` namespace and I will be able to access any Ingresses at `8086b67.jason.awesome-website.dev.mycujoo.tv`.