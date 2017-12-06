package main

import (
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
	jsonTags := getCommandOutput("gcloud", fmt.Sprintf("container images list-tags --format=json eu.gcr.io/mycujoo-development/mycujoo-thumbs"))// %s", repoConfig.ImagePath))
	decodedTags := []gcloudDockerTag{}

	if err := json.Unmarshal([]byte(jsonTags), &decodedTags); err != nil {
		panic(err)
	}
	// prettyPrint, _ := json.MarshalIndent(decodedTags, "", "\t")
	// fmt.Println(string(prettyPrint))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	for _, tag := range decodedTags {
		// fmt.Println(tag)
		fmt.Fprintln(w, fmt.Sprintf("%s  \t  %s", tag.Tags, tag.Timestamp))
	}
	w.Flush()
}