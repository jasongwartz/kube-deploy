package build

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mycujoo/kube-deploy/cli"
	"github.com/mycujoo/kube-deploy/config"
)

const testCommandImage = "mycujoo/gcloud-docker"

var repoConfig config.RepoConfigMap

func MakeAndPushBuild(forcePush bool, dirtyWorkDirOverride bool, keepTestContainer bool, repoConfigParam config.RepoConfigMap) {
	MakeAndTestBuild(dirtyWorkDirOverride, keepTestContainer, repoConfigParam)
	var pushExitCode int
	if forcePush {
		pushExitCode = forcePushDockerImage()
	} else {
		pushExitCode = askPushDockerImage()
	}
	if pushExitCode != 0 {
		os.Exit(1)
	}
}
func MakeAndTestBuild(dirtyWorkDirOverride bool, keepTestContainer bool, repoConfigParam config.RepoConfigMap) {
	if !DockerAmLoggedIn() {
		fmt.Println("=> Uh oh, you're not logged into the configured docker remote for this repo. You won't be able to push!")
		os.Exit(1)
	}

	repoConfig = repoConfigParam

	// Builds the docker image and tags it with the image short-name (ie. without the registry path)
	if repoConfig.ClusterName == "production" && !workingDirectoryIsClean() {
		if dirtyWorkDirOverride {
			fmt.Println("=> Respecting your wishes to override the dirty working directory and build anyway.")
		} else {
			fmt.Println("=> Oh no! You have uncommited changes in the working tree. Please commit or stash before deploying to production.")
			fmt.Println("=> If you're really, really sure, you can override this warning with the '--override-dirty-workdir' flag.")
			os.Exit(1)
		}
	}

	makeBuild()
	runBuildTests(keepTestContainer)
}

func workingDirectoryIsClean() bool {

	cleanWorkDirChecks := []bool{
		cli.GetCommandExitCode("git", "diff-index --quiet HEAD --") == 0, // checks for modified files
		cli.GetCommandOutput("git", "ls-files --others") == "",           // checks for untracked files
	}
	for _, clean := range cleanWorkDirChecks {
		if !clean {
			return false
		}
	}
	return true
}

func makeBuild() {

	fmt.Println("=> Okay, let's start the build process!")
	fmt.Printf("=> First, let's build the image with tag: %s\n\n", repoConfig.ImageFullPath)
	time.Sleep(1 * time.Second)

	// Run docker build
	if exitCode := cli.StreamAndGetCommandExitCode(
		"docker",
		fmt.Sprintf("build -t %s %s", repoConfig.ImageFullPath, repoConfig.PWD),
	); exitCode != 0 {
		os.Exit(1)
	}
}

func runBuildTests(keepTestContainer bool) {
	// Start container and run tests
	tests := repoConfig.Tests
	for _, testSet := range tests {
		fmt.Printf("\n\n=> Setting up test set: %s\n", testSet.Name)

		// Start the test container
		var (
			containerName string
			exitCode      int
		)
		if testSet.Type != "host-only" { // 'host-only' skips running the test docker container (for env setup)
			fmt.Printf("=> Starting docker image: %s\n", repoConfig.ImageFullPath)

			var dockerRunCommand string
			if testSet.DockerArgs != "" {
				dockerRunCommand = fmt.Sprintf("%s %s", testSet.DockerArgs, repoConfig.ImageFullPath)
			} else {
				dockerRunCommand = repoConfig.ImageFullPath
			}
			if testSet.DockerCommand != "" {
				dockerRunCommand = dockerRunCommand + " " + testSet.DockerCommand
			}

			containerName, exitCode = cli.StreamAndGetCommandOutputAndExitCode("docker",
				strings.Join([]string{"run", dockerRunCommand}, " "))
			if exitCode != 0 {
				teardownTest(containerName, true, keepTestContainer)
			}
		}

		// Wait two seconds for it to come alive
		time.Sleep(2 * time.Second)

		// Run all tests
		for _, testCommand := range testSet.Commands {
			// Wait two seconds for it to come alive
			time.Sleep(2 * time.Second)
			fmt.Printf("=> Executing test command: %s\n", testCommand)
			// commandSplit := strings.SplitN(testCommand, " ", 2)
			// Run the test command
			switch t := testSet.Type; t {
			case "on-host", "host-only":
				commandSplit := strings.SplitN(testCommand, " ", 2)
				if exitCode := cli.StreamAndGetCommandExitCode(commandSplit[0], commandSplit[1]); exitCode != 0 {
					teardownTest(containerName, true, keepTestContainer)
					break
				}
			case "in-test-container":
				if exitCode := cli.StreamAndGetCommandExitCode("docker", fmt.Sprintf("exec %s %s", containerName, testCommand)); exitCode != 0 {
					teardownTest(containerName, true, keepTestContainer)
					break
				}
			case "in-external-container":
				if exitCode := cli.StreamAndGetCommandExitCode("docker", fmt.Sprintf("run --rm --network container:%s %s %s", containerName, testCommandImage, testCommand)); exitCode != 0 {
					teardownTest(containerName, true, keepTestContainer)
					break
				}
			default:
				fmt.Printf("=> Since you didn't specify where to run test %s, I'll run it in an external container (attached to the same network).\n", testCommand)
				if exitCode := cli.StreamAndGetCommandExitCode("docker", fmt.Sprintf("run --rm --network container:%s %s %s", containerName, testCommandImage, testCommand)); exitCode != 0 {
					teardownTest(containerName, true, keepTestContainer)
				}
			}
		}
		teardownTest(containerName, false, keepTestContainer)
	}
}

func teardownTest(containerName string, exit bool, keepTestContainer bool) {
	if containerName != "" {
		fmt.Println("=> Stopping test container.")
		cli.GetCommandOutput("docker", fmt.Sprintf("stop %s", containerName))
		if keepTestContainer {
			fmt.Println("=> Leaving the test container without deleting, like you asked.\n")
		} else {
			fmt.Println("=> Removing test container.")
			cli.GetCommandOutput("docker", fmt.Sprintf("rm %s", containerName))
		}
	}
	if exit {
		os.Exit(1)
	}
}

func askPushDockerImage() int {
	fmt.Print("=> Yay, all the tests passed! Would you like to push this to the remote now?\n=> Press 'y' to push, anything else to exit.\n>>> ") // TODO - make this pluggable
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	if confirm != "y\n" && confirm != "Y" {
		fmt.Println("=> Thanks for building, Bob!")
		os.Exit(0)
	}
	return cli.StreamAndGetCommandExitCode("docker", fmt.Sprintf("push %s", repoConfig.ImageFullPath))

}

func forcePushDockerImage() int {
	return cli.StreamAndGetCommandExitCode("docker", fmt.Sprintf("push %s", repoConfig.ImageFullPath))
}
