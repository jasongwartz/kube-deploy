package main

import (
	"fmt"
	"io/ioutil"
	"k8s.io/api/extensions/v1beta1"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"
)

func kubeStartRollout() {

	lockBeforeRollout()
	if !dockerImageExists() {
		makeAndPushBuild()
	}

	existingDeployment := kubeAPIGetSingleDeployment(repoConfig.ReleaseName)
	if existingDeployment.Name != "" {
		fmt.Println("=> Looks like there is an existing deployment by this name, so we'll just update/replace it.\n")
	}

	previousReleases := kubeAPIListDeployments(map[string]string{
		"app": repoConfig.Application.Name + "-" + repoConfig.GitBranch})
	sort.Slice(previousReleases.Items, func(i, j int) bool {
		return previousReleases.Items[i].CreationTimestamp.Time.Sub(previousReleases.Items[j].CreationTimestamp.Time) > 0
	})
	// Find the most recent previous release that doesn't have the same release name (i.e. is not a duplicate of this release)
	var mostRecentRelease v1beta1.Deployment
	for _, r := range previousReleases.Items {
		if r.Name != repoConfig.ReleaseName {
			mostRecentRelease = r
			break
		}
	}

	rolloutStartTime := time.Now()
	// Make the template files, tag deployment with release ID
	for _, f := range kubeMakeTemplates() {
		streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("apply -f %s", f))
	}
	// Find the just-created deployment
	thisDeployment := kubeAPIGetSingleDeployment(repoConfig.ReleaseName)
	desiredPods := *thisDeployment.Spec.Replicas

	// Add the 'kubedeploy-releasetime' label (which will force the deployment to recreate pods if it already existed)
	kubeAPIAddDeploymentLabel(thisDeployment, "kubedeploy-releasetime", strconv.FormatInt(rolloutStartTime.Unix(), 10))
	thisDeployment = kubeAPIUpdateDeployment(thisDeployment)

	if runFlags.Bool("no-canary") || runFlags.Bool("force") {
		streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("rollout status --namespace=%s deployment/%s", repoConfig.Namespace, repoConfig.ReleaseName))
	} else {
		// Quickly scale to only one pod
		firstCanaryPods := int32(1)
		fmt.Printf("=> Scaling to first canary point: %d pod(s)\n", firstCanaryPods)
		thisDeployment.Spec.Replicas = &firstCanaryPods
		thisDeployment = kubeAPIUpdateDeployment(thisDeployment)

		// Make sure first pod gets started
		streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("rollout status --namespace=%s deployment/%s", repoConfig.Namespace, repoConfig.ReleaseName))

		// Pause to watch monitors and make sure that the 1 pod deploy was successful
		fmt.Println("\n=> Wait for at least one minute to make sure the new pod(s) started okay, and is getting some traffic.")
		canaryHoldAndWait(60)

		if desiredPods > firstCanaryPods { // Skip second canary point if no new pods are needed
			// Scale up to desired number of pods in new canary release
			fmt.Printf("=> Scaling to next canary point: %d pod(s)\n=> This should give the new pods roughly 50%% of traffic (if the old deployment was the same size).\n", desiredPods)

			thisDeployment = kubeAPIGetSingleDeployment(repoConfig.ReleaseName)
			thisDeployment.Spec.Replicas = &desiredPods
			thisDeployment = kubeAPIUpdateDeployment(thisDeployment)
			//			streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("scale --namespace=%s deployment/%s --replicas=%d", repoConfig.Namespace, repoConfig.ReleaseName, desiredPods))
			streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("rollout status --namespace=%s deployment/%s", repoConfig.Namespace, repoConfig.ReleaseName))

			fmt.Println("\n=> Now, let's wait for 5 minutes, watch the monitors, and let everything simmer to make sure it looks good.")
			canaryHoldAndWait(300)
		}

		// Scale down pods in old release
		if mostRecentRelease.Name != "" {
			fmt.Println("=> Scaling down old deployment, leaving only new deployment pods.")
			mostRecentRelease.Spec.Replicas = new(int32) // new() returns default value, which is 0 for int32
			mostRecentRelease = *kubeAPIUpdateDeployment(&mostRecentRelease)
			fmt.Println("=> Now, let's wait for another 5 minutes, watch the monitors again, amd make sure we're confident with the new deployment.")

			canaryHoldAndWait(300)
		}
	}

	// Need to retrieve the deployment again after any kube configs
	thisDeployment = kubeAPIGetSingleDeployment(repoConfig.ReleaseName)
	// Tag the new release with 'is-live'
	fmt.Println("=> Tagging the new release with the tag 'kubedeploy-is-live'.")
	kubeAPIAddDeploymentLabel(thisDeployment, "kubedeploy-is-live", "true")
	thisDeployment = kubeAPIUpdateDeployment(thisDeployment)

	// Tag older release with 'instant-rollback-target'
	if mostRecentRelease.Name != "" {
		fmt.Printf("=> Tagging release %s with tag 'instant-rollback-target'.\n=> You can rollback to this in one command with `kube-deploy rollback`.\n", mostRecentRelease.Name)
		// kubeAPIAddDeploymentLabel(mostRecentRelease, "kubedeploy-rollback-target", "true")
		mostRecentRelease.Labels["kubedeploy-rollback-target"] = "true"
		delete(mostRecentRelease.Labels, "kubedeploy-is-live")
		kubeAPIUpdateDeployment(&mostRecentRelease)
	} else {
		fmt.Println("=> Since there are no previous deployments, no 'kubedeploy-rollback-target' will be assigned.")
	}

	// Clean up any older release deployments - leave current new and older
	for _, r := range previousReleases.Items {
		if r.Name != thisDeployment.Name && r.Name != mostRecentRelease.Name {
			fmt.Printf("=> Cleaning up older deployment: %s.\n", r.Name)
			kubeAPIDeleteDeployment(&r)
		}
	}

	// Clean up workdir and remove lockfile
	kubeRemoveTemplates()
	unlockAfterRollout()
}

func kubeRollingRestart() {
	isLiveDeployments := kubeAPIListDeployments(map[string]string{"app": repoConfig.Application.Name + "-" + repoConfig.GitBranch, "kubedeploy-is-live": "true"})

	if len(isLiveDeployments.Items) != 1 {
		fmt.Println("=> Whoah, there's either more or less than one 'is_live' deployment. You should fix that first.")
		for _, i := range isLiveDeployments.Items {
			fmt.Printf("\t%s\n", i.Name)
		}
		os.Exit(1)
	}

	isLive := isLiveDeployments.Items[0]
	isLive.Spec.Template.Labels["kubedeploy-last-rolling-restart"] = strconv.FormatInt(time.Now().Unix(), 10)
	kubeAPIUpdateDeployment(&isLive)
	streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("rollout status --namespace=%s deployment/%s", repoConfig.Namespace, isLive.Name))

	fmt.Printf("\n=> All pods have been recreated.\n\n")
}

func kubeInstantRollback() {
	// Find deployment with label 'instant-rollback-target'
	rollbackTargets := kubeAPIListDeployments(map[string]string{"app": repoConfig.Application.Name + "-" + repoConfig.GitBranch, "kubedeploy-rollback-target": "true"})
	isLiveDeployments := kubeAPIListDeployments(map[string]string{"app": repoConfig.Application.Name + "-" + repoConfig.GitBranch, "kubedeploy-is-live": "true"})

	if len(rollbackTargets.Items) != 1 || len(isLiveDeployments.Items) != 1 {
		fmt.Println("=> Whoah, there's either more or less than one 'is_live' deployment or 'rollback-target' deployment. You should fix that first.")
		for _, i := range isLiveDeployments.Items {
			fmt.Printf("\t%s\n", i.Name)
		}
		for _, i := range rollbackTargets.Items {
			fmt.Printf("\t%s\n", i.Name)
		}
		os.Exit(1)
	}

	isLive := isLiveDeployments.Items[0]
	replicas := isLive.Spec.Replicas
	if int(*replicas) >= 0 {
		oneReplica := int32(1)
		replicas = &oneReplica
	}

	rollbackTarget := rollbackTargets.Items[0]
	rollbackTarget.Spec.Replicas = replicas
	fmt.Printf("=> Rolling back to %s, pod count %d.\n", rollbackTarget.Name, *replicas)
	rollbackTarget.Labels["kubedeploy-is-live"] = "true"
	delete(rollbackTarget.Labels, "kubedeploy-rollback-target")
	kubeAPIUpdateDeployment(&rollbackTarget)
	streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("rollout status --namespace=%s deployment/%s", repoConfig.Namespace, rollbackTarget.Name))

	fmt.Println("=> Wait for one minute to make sure that the old pods came up correctly.")
	canaryHoldAndWait(60)

	// Scale old pods down to zero
	isLive.Spec.Replicas = new(int32)
	isLive.Labels["kubedeploy-rollback-target"] = "true"
	delete(isLive.Labels, "kubedeploy-is-live")
	kubeAPIUpdateDeployment(&isLive)
	fmt.Println("=> Wait for the old pods to scale down to 0.")
	streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("rollout status --namespace=%s deployment/%s", repoConfig.Namespace, rollbackTarget.Name))

	fmt.Printf("=> The deployment has been successfully rolled back to: %s.\n", rollbackTarget.Name)
}

func kubeScaleDeployment(replicas int32) {
	deployments := kubeAPIListDeployments(map[string]string{"app": repoConfig.Application.Name + "-" + repoConfig.GitBranch, "kubedeploy-is-live": "true"})
	if len(deployments.Items) == 1 {
		fmt.Printf("=> Starting to scale to %d replica(s).\n", replicas)
		liveDeployment := deployments.Items[0]
		liveDeployment.Spec.Replicas = &replicas
		kubeAPIUpdateDeployment(&liveDeployment)
		streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("rollout status --namespace=%s deployment/%s", repoConfig.Namespace, repoConfig.ReleaseName))
		fmt.Printf("=> Finished scaling to %d replica(s).\n", replicas)
	} else {
		fmt.Println("=> Whoah, there's more than one 'is_live' deployment. You should fix that first.")
		for _, i := range deployments.Items {
			fmt.Printf("\t%s\n", i.Name)
		}
		os.Exit(1)
	}
}

func kubeListDeployments() {
	deployments := kubeAPIListDeployments(map[string]string{"app": repoConfig.Application.Name + "-" + repoConfig.GitBranch})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, fmt.Sprintf("%s  \t  %s", "Active Deployments", "Date Created"))
	fmt.Fprintln(w, fmt.Sprintf("%s  \t  %s", "----------", "----------"))
	for _, d := range deployments.Items {
		fmt.Fprintln(w, fmt.Sprintf("%s  \t  %s", d.Name, d.CreationTimestamp))
	}
	w.Flush()
}

func kubeRemoveTemplates() {
	os.RemoveAll(repoConfig.PWD + "/.kubedeploy-temp")
}

// Returns a list of the filenames of the filled-out templates
func kubeMakeTemplates() []string {
	os.MkdirAll(repoConfig.PWD+"/.kubedeploy-temp", 0755)

	var filePaths []string
	for _, filename := range getKubeTemplateFiles() {
		fmt.Printf("=> Generating YAML from template for %s\n", filename)
		kubeFileTemplated := runConsulTemplate(repoConfig.Application.PathToKubernetesFiles + "/" + filename)

		tempFilePath := repoConfig.PWD + "/.kubedeploy-temp/" + filename
		err := ioutil.WriteFile(tempFilePath, []byte(kubeFileTemplated), 0644)
		if err != nil {
			fmt.Println(err)
		}
		filePaths = append(filePaths, tempFilePath)
	}
	return filePaths
}

func canaryHoldAndWait(waitTimeSeconds int) bool {
	firstPromptTime := time.Now()
	printablePromptTime := firstPromptTime.Format("Jan _2 15:04:05")
	askToProceed(fmt.Sprintf("%s: You are at a canary point.", printablePromptTime), "Well, you bailed out. I'm leaving things in an unclean state, so you'll have to clean up yourself.")
	elasped := int(time.Since(firstPromptTime).Seconds())

	if elasped < waitTimeSeconds {
		askToProceed("=> Bad behaviour - you're back too quickly. Honestly, are you really sure?", "Okay, I guess you weren't sure. Make sure to clean up your half-progress.")
	}
	return true
}