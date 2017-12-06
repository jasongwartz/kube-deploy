package main

import (
	"fmt"
	"time"
	"os"
	"strings"
//	"os/exec"
)

// Returns 'true' for clean, 'false' for dirty
func checkWorkingDirectory () (bool) {
	if *flags["dirtyWorkingDirectoryOverride"] {
		fmt.Println("=> Respecting your wishes to override the dirty working directory and build anyway.")
		return true
	}

	dirtyWorkingDirectory := []int{
		getCommandExitCode("git", "diff-index --quiet HEAD --"), // checks for modified files
		getCommandExitCode("test", "-z \"$(git ls-files --others)\"")} // checks for untracked files
	for _, code := range dirtyWorkingDirectory {
		if code != 0 {
			return false
		}
	}
	return true
}

func makeBuild() {

	if repoConfig.EnvironmentName == "production" {
		if ! checkWorkingDirectory() {
			fmt.Println("=> You have uncommited changes in the working tree. Please commit or stash before deploying to production.")
			os.Exit(1)
		}
	}

	fmt.Println("=> Okay, let's start the build process!")
	fmt.Printf("=> First, let's build the image with tag: %s\n\n", repoConfig.ImageName)
	time.Sleep(1 * time.Second)

	// Run docker build
	streamAndGetCommandOutput("docker", fmt.Sprintf("build -t %s %s", repoConfig.ImageName, repoConfig.PWD))

	// Start container and run tests
	tests := repoConfig.Tests
	for _, testSet := range tests {
		fmt.Printf("\n\n=> Setting up test set: %s\n", testSet.Name)
		fmt.Printf("=> Starting docker image %s...\n", repoConfig.ImageName)
		containerName, exitCode := getCommandOutputAndExitCode(
			"docker",
			fmt.Sprintf("run --rm -d %s %s", testSet.DockerArgs, repoConfig.ImageName))
		if exitCode != 0 {
			teardownAndExit(containerName)
		}
		for _, testCommand := range testSet.Commands {
			fmt.Printf("=> Executing test command: %s\n", testCommand)
			commandSplit := strings.SplitN(testCommand, " ", 2)
			if _, exitCode := streamAndGetCommandOutputAndExitCode(commandSplit[0], commandSplit[1]); exitCode != 0 {
				teardownAndExit(containerName)
				break
			}
		}
		teardownAndExit(containerName)
	}

	fmt.Printf("=> Tagging the image short name %s with the image full path:\n\t%s.\n\n", repoConfig.ImageName, repoConfig.ImageFullPath)

	fmt.Print("=> Do you want to push this to gcloud now? Press 'y' to push, anything else to exit.\n>>> ") // TODO - make this pluggable
	confirm, _ := reader.ReadString('\n')
	if confirm != "y\n" && confirm != "Y" {
		fmt.Println("=> Thanks for building, Bob!")
		os.Exit(0)
	} else {
		streamAndGetCommandOutput("gcloud", fmt.Sprintf("docker -- push %s", repoConfig.ImageFullPath))		
	}
}

func teardownAndExit(containerName string) {
	fmt.Println("=> Tearing down test container.")
	getCommandOutput("docker", fmt.Sprintf("stop %s", containerName))
	os.Exit(1)
}
