package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
)

// func initUserConfig(configFilePath string) userConfigMap {

// 	configFile, err := ioutil.ReadFile(configFilePath)
// 	if err != nil {
// 		fmt.Fprintln(os.Stderr, "Failed reading user config file:", err)
// 		os.Exit(1)
// 	}

// 	userConfig := userConfigMap{}
// 	err = yaml.Unmarshal(configFile, &userConfig)
// 	if err != nil {
// 		fmt.Fprintln(os.Stderr, "Failed parsing YAML user config file:", err)
// 		os.Exit(1)
// 	}

// 	return userConfig
// }

// // TODO : find someway to de-duplicate these two functions

// repoConfigMap : hash of the YAML data from project's deploy.yaml
type repoConfigMap struct {
	DockerRepository struct {
		DevelopmentRepositoryName string `yaml:"developmentRepositoryName"`
		ProductionRepositoryName  string `yaml:"productionRepositoryName"`
		RegistryRoot              string `yaml:"registryRoot"`
	} `yaml:"dockerRepository"`
	Application struct {
		Name                  string `yaml:"name"`
		Version               string `yaml:"version"`
		PathToKubernetesFiles string `yaml:"pathToKubernetesFiles"`
		KubernetesTemplate    struct {
			GlobalVariables []string            `yaml:"globalVariables"`
			BranchVariables map[string][]string `yaml:"branchVariables"`
		} `yaml:"kubernetesTemplate"`
	} `yaml:"application"`
	DockerRepositoryName string
	ClusterName          string // 'production' or 'development' - 'staging' should use the production cluster
	Namespace            string
	GitBranch            string
	BuildID              string
	ImageTag             string
	ImageName            string
	ImageFullPath        string `yaml:"imageFullPath"`
	PWD                  string
	ReleaseName          string
	KubeAPIClientSet     *kubernetes.Clientset
	Tests                []testConfigMap `yaml:"tests"`
}

// testConfigMap : layout of the details for running a single test step (during build)
type testConfigMap struct {
	Name          string   `yaml:"name"`
	DockerArgs    string   `yaml:"dockerArgs"`
	DockerCommand string   `yaml:"dockerCommand"`
	Type          string   `yaml:"type"`
	Commands      []string `yaml:"commands"`
}

func initRepoConfig(configFilePath string) repoConfigMap {

	configFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed reading repo config file:", err)
		os.Exit(1)
	}

	repoConfig := repoConfigMap{}
	err = yaml.Unmarshal(configFile, &repoConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed parsing YAML repo config file:", err)
		os.Exit(1)
	}

	repoConfig.GitBranch = strings.TrimSuffix(getCommandOutput("git", "rev-parse --abbrev-ref HEAD"), "\n")
	invalidDockertagCharRegex := regexp.MustCompile(`([^a-z|A-Z|0-9|\-|_|\.])`)
	repoConfig.GitBranch = invalidDockertagCharRegex.ReplaceAllString(repoConfig.GitBranch, "-")
	repoConfig.BuildID = strings.TrimSuffix(getCommandOutput("git", "rev-parse --verify --short HEAD"), "\n")

	switch branch := repoConfig.GitBranch; branch {
	case "production":
		repoConfig.DockerRepositoryName = repoConfig.DockerRepository.ProductionRepositoryName
		repoConfig.ClusterName = "production"
		if repoConfig.Namespace == "" {
			repoConfig.Namespace = "production"
		}
	case "master":
		repoConfig.DockerRepositoryName = repoConfig.DockerRepository.ProductionRepositoryName
		repoConfig.ClusterName = "production" // deploy to production cluster
		if repoConfig.Namespace == "" {
			repoConfig.Namespace = "staging"
		}
	default:
		repoConfig.DockerRepositoryName = repoConfig.DockerRepository.DevelopmentRepositoryName
		repoConfig.ClusterName = "development"
		if repoConfig.Namespace == "" {
			repoConfig.Namespace = "development"
		}
	}

	repoConfig.ImageTag = fmt.Sprintf("%s-%s-%s",
		repoConfig.Application.Version,
		repoConfig.GitBranch,
		repoConfig.BuildID)

	if repoConfig.ImageFullPath == "" { // if the path was not already provided in the deploy.yaml
		repoConfig.ImageName = fmt.Sprintf("%s/%s:%s", repoConfig.DockerRepositoryName, repoConfig.Application.Name, repoConfig.ImageTag)
		if repoConfig.DockerRepository.RegistryRoot != "" {
			repoConfig.ImageFullPath = fmt.Sprintf("%s/%s", repoConfig.DockerRepository.RegistryRoot, repoConfig.ImageName)
		} else { // For DockerHub images, no RegistryRoot is needed
			repoConfig.ImageFullPath = repoConfig.ImageName
		}
	}

	repoConfig.ReleaseName = fmt.Sprintf("%s-%s", repoConfig.Application.Name, repoConfig.ImageTag)
	repoConfig.PWD, err = os.Getwd()

	repoConfig.KubeAPIClientSet = setupKubeAPI()

	return repoConfig
}
