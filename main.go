package main

import (
	"fmt"
	"os"
	"bufio"
	"os/user"
	"flag"
)

var userConfig userConfigMap
var repoConfig repoConfigMap
var reader *bufio.Reader

func main() {
	parseFlags()
	pwd, _ := os.Getwd()
	user, _ := user.Current()
	userHome := user.HomeDir
	reader = bufio.NewReader(os.Stdin)

	fmt.Println("\n=> Welcome to kube-deploy.\n\n")
	fmt.Println("=> First, I'm going to read your user configuration file.")
	userConfig = initUserConfig(fmt.Sprintf("%s/.kube-deploy.conf", userHome))
	fmt.Println("=> Now, I'm going to read the repo configuration file.")
	repoConfig = initRepoConfig(fmt.Sprintf("%s/deploy.yaml", pwd))
	fmt.Printf(`=> I found the following data:
		Repository name: %s
		Current branch: %s
		HEAD hash: %s
		
=> That means we're dealing with the image tag:
		%s
		`, repoConfig.Application.Name, repoConfig.GitBranch, repoConfig.BuildID, repoConfig.ImageFullPath)

	fmt.Print("\n\n=> Press 'y' if this is correct, anything else to exit.\n>>>  ")
	confirm, _ := reader.ReadString('\n')
	if confirm != "y\n" && confirm != "Y" {
		fmt.Println("I'm sorry to say goodbye, I thought we really had something.")
		os.Exit(1)
	}

	makeBuild()
}

var flags map[string]*bool
func parseFlags() {
	flags = make(map[string]*bool)
	dirtyWorkingDirectoryOverride := flag.Bool("override-dirty-workdir", false, "Forces a build even if the git working directory is dirty.")
	flag.Parse()
	flags["dirtyWorkingDirectoryOverride"] = dirtyWorkingDirectoryOverride
}