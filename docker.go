package main

import (
	"strings"
	"os"
	"text/tabwriter"
	"fmt"
	"encoding/json"
)
type gcloudDockerTag struct {
	Digest string `json:"digest"`
	Tags []string
	Timestamp struct {
		Datetime string
	}
}

func dockerListTags() {
	if ! strings.Contains(repoConfig.DockerRepository.RegistryRoot, "gcr.io") {
		fmt.Println("=> Sorry, the 'list-tags' feature only works with Google Cloud Registry.")
		os.Exit(1)
	}

	jsonTags := getCommandOutput("gcloud", fmt.Sprintf("container images list-tags --format=json eu.gcr.io/mycujoo-development/mycujoo-thumbs"))// %s", repoConfig.ImagePath))
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

func dockerImageExists() (bool) {
	exitCode := getCommandExitCode("docker", fmt.Sprintf("inspect %s", repoConfig.ImageFullPath))

	if exitCode != 0 {
		return false
	}
	return true
}