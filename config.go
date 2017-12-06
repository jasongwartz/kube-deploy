package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
	"regexp"
)

// userConfigMap : hash of the YAML data from the Users ~/.deploy.conf
type userConfigMap struct {
	GoogleCloud struct {
		DevelopmentProjectName string `yaml:"developmentProjectName"`
		ProductionProjectName string `yaml:"productionProjectName"`
		RegistryRoot string `yaml:"registryRoot"`
	} `yaml:"googleCloud"`
}

func initUserConfig(configFilePath string) (userConfigMap) {

	configFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed reading user config file:", err)
		os.Exit(1)
	}

	userConfig := userConfigMap{}
	err = yaml.Unmarshal(configFile, &userConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed parsing YAML user config file:", err)
		os.Exit(1)
	}

	return userConfig
}

// TODO : find someway to de-duplicate these two functions

// repoConfigMap : hash of the YAML data from project's deploy.yaml
type repoConfigMap struct {
	Application struct {
		Name string `yaml:"name"`
		Version string `yaml:"version"`
		RemoteType string `yaml:"remoteType"`
	} `yaml:"application"`
	GoogleCloudProjectName string
	EnvironmentName string // 'production' or 'development'
	GitBranch string
	BuildID string
	ImageTag string
	ImageName string
	ImageFullPath string
	PWD string
	Tests []testConfigMap `yaml:"tests"`
}

// testConfigMap : layout of the details for running a single test step (during build)
type testConfigMap struct {
	Name string `yaml:"name"`
	DockerArgs string `yaml:"dockerArgs"`
	Commands []string `yaml:"commands"`
}
func initRepoConfig(configFilePath string) (repoConfigMap) {

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

	if repoConfig.GitBranch == "master" || repoConfig.GitBranch == "production" {
		repoConfig.GoogleCloudProjectName = userConfig.GoogleCloud.ProductionProjectName
		repoConfig.EnvironmentName = "production"
	} else {
		repoConfig.GoogleCloudProjectName = userConfig.GoogleCloud.DevelopmentProjectName
		repoConfig.EnvironmentName = "development"
	}

	repoConfig.ImageTag = fmt.Sprintf("%s-%s-%s",
		repoConfig.Application.Version,
		repoConfig.GitBranch,
		repoConfig.BuildID)

	repoConfig.ImageName = fmt.Sprintf("%s/%s:%s", repoConfig.GoogleCloudProjectName, repoConfig.Application.Name, repoConfig.ImageTag)
	repoConfig.ImageFullPath = fmt.Sprintf("%s/%s", userConfig.GoogleCloud.RegistryRoot, repoConfig.ImageName)
	repoConfig.PWD, err = os.Getwd()

	return repoConfig
}