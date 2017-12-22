package main

import (
	"bytes"
	"os"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"
)


// Returns a list of the filenames of the filled-out templates
func kubeMakeTemplates() []string {
	os.MkdirAll(repoConfig.PWD+"/.kubedeploy-temp", 0755)

	var filePaths []string
	for _, filename := range getKubeTemplateFiles() {
		fmt.Printf("=> Generating YAML from template for %s\n", filename)
		kubeFileTemplated := runConsulTemplate(repoConfig.Application.PathToKubernetesFiles + "/" + filename)

		tempFilePath := repoConfig.PWD + "/.kubedeploy-temp/" + filename
		err := ioutil.WriteFile(tempFilePath, []byte(kubeFileTemplated), 0644)
		if err != nil {
			fmt.Println(err)
		}
		filePaths = append(filePaths, tempFilePath)
	}
	return filePaths
}

func kubeRemoveTemplates() {
	os.RemoveAll(repoConfig.PWD + "/.kubedeploy-temp")
}

func getKubeTemplateFiles() ([]string) {
	return strings.Split(getCommandOutput("ls", repoConfig.Application.PathToKubernetesFiles), "\n")
}

func runConsulTemplate(filename string) (string) {
	vaultAddr := os.Getenv("VAULT_ADDR");
	if  vaultAddr != "" {
		vaultAddr = fmt.Sprintf("--vault-renew-token=false --vault-retry=false --vault-addr %s", vaultAddr)
		os.Setenv("SECRETS_LOCATION", "production")// repoConfig.EnvironmentName)
	}
	consulTemplateArgs := fmt.Sprintf("%s -template %s -once -dry", vaultAddr, filename)

	// the map which will contain all environment variables to be set before running consul-template
	envMap := make(map[string]string)

	// Include the template freebie variables
	envMap["KD_RELEASE_NAME"] = repoConfig.ReleaseName
	envMap["KD_APP_NAME"] = repoConfig.Application.Name + "-" + repoConfig.GitBranch
	envMap["KD_KUBERNETES_NAMESPACE"] = repoConfig.Namespace
	envMap["KD_GIT_BRANCH"] = repoConfig.GitBranch
	envMap["KD_ENVIRONMENT_NAME"] = repoConfig.EnvironmentName
	envMap["KD_IMAGE_FULL_PATH"] = repoConfig.ImageFullPath
	envMap["KD_IMAGE_TAG"] = repoConfig.ImageTag

	var branchName string
	if _, ok := repoConfig.Application.KubernetesTemplate.BranchVariables[repoConfig.GitBranch]; ok {
		branchName = repoConfig.GitBranch
	} else {
		branchName = "else"
	}
	// Parse and add the branch-specific env vars first
	for _, envVar := range repoConfig.Application.KubernetesTemplate.BranchVariables[branchName] {
		split := strings.Split(envVar, "=")
		envMap[split[0]] = split[1]
	}

	// Parse and add the rest of the env vars
	for _, envVar := range repoConfig.Application.KubernetesTemplate.GlobalVariables {
		split := strings.Split(envVar, "=")
		envMap[split[0]] = split[1]
	}

	// Add the variables to the environment, doing any inline substitutions
	for key, value := range envMap {
		var envVarBuf bytes.Buffer
		tmplVar, err := template.New("EnvVar: " + key).Parse(value)
		err = tmplVar.Execute(&envVarBuf, envMap)
		if err != nil {
			fmt.Println("=> Uh oh, failed to do a substitution in one of your template variables.")
			fmt.Println(err)
			os.Exit(1)
		}
		os.Setenv(key, envVarBuf.String())
	}

	output, exitCode := getCommandOutputAndExitCode("consul-template", consulTemplateArgs)
	if exitCode != 0 {
		fmt.Println("=> Oh no, looks like consul-template failed!")
		os.Exit(1)
	}

	return strings.Join(strings.Split(output, "\n")[1:], "\n")
}
