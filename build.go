package main

import (
	"fmt"
	"os"
	"strings"
	"time"
	//	"os/exec"
)

func makeAndPushBuild() {
	makeAndTestBuild()
	pushDockerImage()
}
func makeAndTestBuild() {
	makeBuild()
	runBuildTests()
	tagDockerImage()
}

func checkWorkingDirectory() bool {
	// Returns 'true' for clean, 'false' for dirty
	if runFlags.Bool("override-dirty-workdir") {
		fmt.Println("=> Respecting your wishes to override the dirty working directory and build anyway.")
		return true
	}

	dirtyWorkingDirectory := []int{
		getCommandExitCode("git", "diff-index --quiet HEAD --"),       // checks for modified files
		getCommandExitCode("test", "-z \"$(git ls-files --others)\"")} // checks for untracked files
	for _, code := range dirtyWorkingDirectory {
		if code != 0 {
			return false
		}
	}
	return true
}

func makeBuild() {
	// Builds the docker image and tags it with the image short-name (ie. without the registry path)
	if repoConfig.EnvironmentName == "production" {
		if !checkWorkingDirectory() {
			fmt.Println("=> Oh no! You have uncommited changes in the working tree. Please commit or stash before deploying to production.")
			os.Exit(1)
		}
	}

	fmt.Println("=> Okay, let's start the build process!")
	fmt.Printf("=> First, let's build the image with tag: %s\n\n", repoConfig.ImageName)
	time.Sleep(1 * time.Second)

	// Run docker build
	streamAndGetCommandOutput("docker", fmt.Sprintf("build -t %s %s", repoConfig.ImageName, repoConfig.PWD))
}

func runBuildTests() {
	// Start container and run tests
	tests := repoConfig.Tests
	for _, testSet := range tests {
		fmt.Printf("\n\n=> Setting up test set: %s\n", testSet.Name)
		fmt.Printf("=> Starting docker image %s...\n", repoConfig.ImageName)

		var dockerRunCommand string
		if testSet.DockerArgs != "" {
			dockerRunCommand = fmt.Sprintf("%s %s", testSet.DockerArgs, repoConfig.ImageName)
		} else {
			dockerRunCommand = repoConfig.ImageName
		}

		// Start the test container
		containerName, exitCode := getCommandOutputAndExitCode("docker",
			strings.Join([]string{"run -d", dockerRunCommand}, " "))
		if exitCode != 0 {
			teardownTest(containerName, true)
		}

		// Wait two seconds for it to come alive
		time.Sleep(2 * time.Second)

		// Run all tests
		for _, testCommand := range testSet.Commands {
			// Wait two seconds for it to come alive
			time.Sleep(2 * time.Second)
			fmt.Printf("=> Executing test command: %s\n", testCommand)
			commandSplit := strings.SplitN(testCommand, " ", 2)
			if _, exitCode := streamAndGetCommandOutputAndExitCode(commandSplit[0], commandSplit[1]); exitCode != 0 {
				teardownTest(containerName, true)
				break
			}
		}
		teardownTest(containerName, false)
	}
}

func teardownTest(containerName string, exit bool) {
	fmt.Println("=> Tearing down test container.")
	getCommandOutput("docker", fmt.Sprintf("stop %s", containerName))
	if exit { os.Exit(1) }
}

func tagDockerImage() {
	fmt.Printf("=> Tagging the image short name %s with the image full path:\n\t%s.\n\n", repoConfig.ImageName, repoConfig.ImageFullPath)
	getCommandOutput("docker", fmt.Sprintf("tag %s %s", repoConfig.ImageName, repoConfig.ImageFullPath))
}

func pushDockerImage() {
	fmt.Print("=> Yay, all the tests passed! Would you like to push this to the remote now?\n=> Press 'y' to push, anything else to exit.\n>>> ") // TODO - make this pluggable
	confirm, _ := reader.ReadString('\n')
	if confirm != "y\n" && confirm != "Y" {
		fmt.Println("=> Thanks for building, Bob!")
		os.Exit(0)
	} else {
		streamAndGetCommandOutput("gcloud", fmt.Sprintf("docker -- push %s", repoConfig.ImageFullPath))
	}
}
