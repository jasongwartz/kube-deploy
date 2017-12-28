package main

import (
	"io/ioutil"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

type gcloudDockerTag struct {
	Digest    string `json:"digest"`
	Tags      []string
	Timestamp struct {
		Datetime string
	}
}

func dockerListTags() {
	if !strings.Contains(repoConfig.DockerRepository.RegistryRoot, "gcr.io") {
		fmt.Println("=> Sorry, the 'list-tags' feature only works with Google Cloud Registry.")
		os.Exit(1)
	}

	jsonTags := getCommandOutput("gcloud", fmt.Sprintf("container images list-tags --format=json eu.gcr.io/mycujoo-development/mycujoo-thumbs")) // %s", repoConfig.ImagePath))
	decodedTags := []gcloudDockerTag{}

	if err := json.Unmarshal([]byte(jsonTags), &decodedTags); err != nil {
		panic(err)
	}
	// prettyPrint, _ := json.MarshalIndent(decodedTags, "", "\t")
	// fmt.Println(string(prettyPrint))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, fmt.Sprintf("%s  \t  %s", "List of Tags", "Date Tagged"))
	fmt.Fprintln(w, fmt.Sprintf("%s  \t  %s", "----------", "----------"))
	for _, tag := range decodedTags {
		// fmt.Println(tag)
		fmt.Fprintln(w, fmt.Sprintf("%s  \t  %s", strings.Join(tag.Tags, ", "), tag.Timestamp.Datetime))
	}
	w.Flush()
}

func dockerImageExistsLocal() bool {
	exitCode := getCommandExitCode("docker", fmt.Sprintf("inspect %s", repoConfig.ImageFullPath))

	if exitCode != 0 {
		return false
	}
	return true
}

func dockerImageExistsRemote() bool {
	exitCode := getCommandExitCode("docker", fmt.Sprintf("pull %s", repoConfig.ImageFullPath))

	if exitCode != 0 {
		return false
	}
	return true
}

func dockerAmLoggedIn() bool {

	dockerAuthFile, err := ioutil.ReadFile(os.Getenv("HOME") + "/.docker/config.json")
	if err != nil {
		fmt.Println("=> There was a problem reading your docker config file, so I don't know if you're logged in!")
		panic(err.Error())
	}

	var dockerAuthData map[string]interface{}
	json.Unmarshal(dockerAuthFile, &dockerAuthData)

	auths := dockerAuthData["auths"].(map[string]interface{})

	// If no RegistryRoot is specified, look for dockerhub details
	var authToLookFor string
	if repoConfig.DockerRepository.RegistryRoot == "" {
		authToLookFor = "docker"
	} else {
		authToLookFor = repoConfig.DockerRepository.RegistryRoot
	}

	for remoteName := range auths {
		if remoteName == authToLookFor || remoteName == "https://" + authToLookFor {
			return true
		}
	}

	return false
}