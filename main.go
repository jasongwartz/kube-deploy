package main

import (
	"strconv"
	"bufio"
	"io/ioutil"
	// "flag"
	"fmt"
	"os"
	// "os/user"
	"github.com/simonleung8/flags"
)

// var userConfig userConfigMap
var repoConfig repoConfigMap
var reader *bufio.Reader

func main() {
	parseFlags()
	pwd, _ := os.Getwd()
	// user, _ := user.Current()
	// userHome := user.HomeDir
	reader = bufio.NewReader(os.Stdin)

	fmt.Println("\n=> Welcome to kube-deploy.\n\n")

	if ! runFlags.Bool("test-only") {
		fmt.Println("=> First, I'm going to read the repo configuration file.")
		repoConfig = initRepoConfig(fmt.Sprintf("%s/deploy.yaml", pwd))
		fmt.Printf(`=> I found the following data:
			Repository name: %s
			Current branch: %s
			HEAD hash: %s
			
=> That means we're dealing with the image tag:
	%s
	`, repoConfig.Application.Name, repoConfig.GitBranch, repoConfig.BuildID, repoConfig.ImageFullPath)

		if exitCode := getCommandExitCode("curl", "-s --connect-timeout 3 https://ifconfig.co"); exitCode != 0 {
			fmt.Println("=> Uh oh, looks like you're not connected to the internet (or maybe it's just too slow).")
			os.Exit(1)
		}	
	}
	// args has to have at least length 2, since the first element is the executable name
	if len(args) >= 2 {
		fmt.Printf("\n=> You've chosen the action '%s'. Proceeding...\n\n", args[1])

		switch c := args[1]; c {
		case "install":
			installDependencies()
		case "build":
			makeAndPushBuild()
		case "make":
			makeAndPushBuild()
		case "test":
			makeAndTestBuild()
		case "testonly":
			runBuildTests()

		case "start-rollout":
			kubeStartRollout()
		case "scale":
			replicas, _ := strconv.ParseInt(args[2], 0, 32)
			kubeScaleDeployment(int32(replicas))
		case "rollback":
			kubeInstantRollback()
		case "rolling-restart":
			kubeRollingRestart()

		case "active-deployments":
			kubeListDeployments()
		case "list-tags":
			dockerListTags()

		case "lock": writeLockFile(repoConfig.Application.Name, "manually blocked rollouts for " + repoConfig.Application.Name)
		case "unlock": deleteLockFile(repoConfig.Application.Name)
		case "lock-all": writeLockFile("all", "manually blocked all rollouts")
		case "unlock-all": deleteLockFile("all")
		default:
			{
				fmt.Println("=> Uh oh - that command isn't recongised. Please enter a valid command. Do you need some help?")
				fmt.Print("=> Press 'y' to show the help menu, anything else to exit.\n>>>  ")
				pleaseHelpMe, _ := reader.ReadString('\n')
				if pleaseHelpMe != "y\n" && pleaseHelpMe != "Y\n" {
					fmt.Println("Better luck next time.")
					os.Exit(0)
				}
				showHelp()
			}
		}
	} else {
		fmt.Println("You'll need to add a command.")
		os.Exit(0)
	}
}

func askToProceed(promptMessage string, exitMessage string) {
	fmt.Printf("=> %s\n=> Press 'y' to proceed, anything else to exit.\n>>> ", promptMessage)
	if proceed, _ := reader.ReadString('\n'); proceed != "y\n" && proceed != "Y\n" {
		fmt.Println("=> " + exitMessage)
		os.Exit(0)
	}
}

func installDependencies() {
	// Installs:
		// git
		// vault
		// consul-template
		// kubectl
	
}

func showHelp() {
	helpData, err := ioutil.ReadFile("README.md") // make this part of the application bundle
	if err != nil {
		fmt.Println("=> Oh no, we couldn't even read the help file!")
		panic(err)
	}
	fmt.Print(string(helpData))
}

var args []string
var runFlags flags.FlagContext

func parseFlags() {

	runFlags = flags.New()
	runFlags.NewBoolFlag("debug", "", "Print extra-fun information.")
	runFlags.NewBoolFlag("override-dirty-workdir", "", "Forces a build even if the git working directory is dirty (only needed for 'production' and 'master' branches).")
	runFlags.NewBoolFlag("force", "", "Unwisely bypasses the sanity checks, which you really need. Even you.")
	runFlags.NewBoolFlag("force-push-image", "", "Automatically push the built Docker image if the tests pass (useful for CI/CD).")
	runFlags.NewBoolFlag("keep-test-container", "", "Don't clean up (docker rm) the test containers (Default false).")
	runFlags.NewBoolFlag("no-canary", "", "Bypass the canary release points (useful for CI/CD).")
	runFlags.NewBoolFlag("test-only", "", "Skips the run configuration and only tests that the binary can start.")

	if err := runFlags.Parse(os.Args...); err != nil {
		fmt.Println("\n=> Oh no, I don't know what to do with those command line flags. Sorry...\n")
		fmt.Println(runFlags.ShowUsage(4))
		os.Exit(1)
	}
	args = runFlags.Args()
}
