package main

import (
	"time"
	"os"
	"fmt"
	// "encoding/json"
	"io/ioutil"
	"text/tabwriter"
	// "strings"
)

func lockFirst() {
	// os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0666) // or exit

	// callback()
	//unlock
}

// type kubeFunc func()

// func lockingKubeFunc(f kubeFunc) kubeFunc {
//     return func() {
//         lockFirst()
// 		return f()
// 		unlockLast()
//     }
// }

type kubeDeploymentReleaseSet struct {
	// Items []struct {
	// 	Kind string `json:"kind"`
	// 	APIVersion string
	// } `json:"items"` //[]kubeDeploymentReleaseItem //`json:"items"`
	Kind string
	APIVersion string
	Items []interface{} `json:"items"` // []test
}

func kubeStartRollout() {
	
	// previousReleases := getCommandOutput("kubectl",
	// 	fmt.Sprintf("get deployments --namespace=%s -l app=%s --output=json",
	// 		repoConfig.Namespace,
	// 		repoConfig.Application.Name + "-" + repoConfig.GitBranch))
	// decodedReleases := kubeDeploymentReleaseSet{}

	// if err := json.Unmarshal([]byte(previousReleases), &decodedReleases); err != nil {
	// 	panic(err)
	// }

	// data := map[string]interface{}{}
	// dec := json.NewDecoder(strings.NewReader(previousReleases))
	// dec.Decode(&data)

	// fmt.Println(decodedReleases)
	// fmt.Println(data)

	
	// Make the template files, tag deployment with release ID
	for _, f := range kubeMakeTemplates() {
		streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("apply -f %s", f))
	}
	desiredPods := kubeAPIGetPodSpecReplicaCount()
	// Quickly scale to only one pod
	fmt.Println("=> Scaling to first canary point: one pod")
	streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("scale --namespace=%s deployment/%s --replicas=1", repoConfig.Namespace, repoConfig.ReleaseName))	
	// Make sure first pod gets started
	streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("rollout status --namespace=%s deployment/%s", repoConfig.Namespace, repoConfig.ReleaseName))

	// Pause to watch monitors and make sure that the 1 pod deploy was successful
	canarySimmer(60)

	// Scale up to desired number of pods in new canary release
	fmt.Printf("=> Scaling to next canary point: %d pod(s)\n=> This should be roughly 50/50.\n", desiredPods)
	streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("scale --namespace=%s deployment/%s --replicas=%d", repoConfig.Namespace, repoConfig.ReleaseName, desiredPods))
	streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("rollout status --namespace=%s deployment/%s", repoConfig.Namespace, repoConfig.ReleaseName))
	
	fmt.Println("=> Now, let's wait for 5 minutes, watch the monitors, and let everything simmer to make sure it looks good.")
	canarySimmer(300)

	// Scale down pods in old release
	fmt.Println("=> Scaling down old deployment, leaving only new deployment pods.")
	fmt.Println("=> Now, let's wait for another 5 minutes, watch the monitors again, amd make sure we're confident with the new deployment.")
	
	canarySimmer(300)

	// Tag older release with 'instant-rollback-target'
	fmt.Printf("=> Tagging release %s with tag 'instant-rollback-target'.\n=> You can rollback to this in one command with `kube-deploy rollback`.\n", "OLDER")

	// Clean up any older release deployments - leave current new and older
	fmt.Println("=> Cleaning up any older deployments for this project and branch.")
	
	
}

func canarySimmer(waitTimeSeconds int) {
	firstPromptTime := time.Now()
	askToProceed("You are at a canary point.", "Well, you bailed out. I'm leaving things in an unclean state, so you'll have to clean up yourself.")
	elasped := int(time.Since(firstPromptTime).Seconds())

	if elasped < waitTimeSeconds {
		fmt.Println("=> Bad behaviour - you're back too quickly. Honestly, are you really sure?")
		canarySimmer(waitTimeSeconds - elasped)
	}
	return
}

func kubeInstantRollback() {
	// Find deployment with label 'instant-rollback-target'

	kubeFinishRollback()
	
}

func kubeListDeployments() {
	deployments := kubeAPIListDeployments()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, fmt.Sprintf("%s  \t  %s", "Active Deployments", "Date Created"))
	fmt.Fprintln(w, fmt.Sprintf("%s  \t  %s", "----------", "----------"))
	for _, d := range deployments.Items {
		fmt.Fprintln(w, fmt.Sprintf("%s  \t  %s", d.Name, d.CreationTimestamp))
	}
	w.Flush()
}

func kubeTargetRollback() {
	// Create deployment config for given Docker tag

	kubeFinishRollback()
}

func kubeFinishRollback() {
	// Wait for pods to scale up in rollback target

	// Canary point

	// Scale down newer release pods OR delete newer deployment
}

func kubeDeleteResources() {

}

func kubeRemoveTemplates() {
	os.RemoveAll(repoConfig.PWD + "/.kubedeploy-temp")
}

// Returns a list of the filenames of the filled-out templates
func kubeMakeTemplates() ([]string) {
	os.MkdirAll(repoConfig.PWD + "/.kubedeploy-temp", 0755)

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