package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"
)

// Returns a list of the filenames of the filled-out templates
func kubeMakeTemplates() []string {
	os.MkdirAll(repoConfig.PWD+"/.kubedeploy-temp", 0755)

	templateFiles, err := ioutil.ReadDir(repoConfig.Application.PathToKubernetesFiles)
	if err != nil {
		fmt.Println("=> Unable to get list of kubernetes files.")
		os.Exit(1)
	}

	var filePaths []string
	for _, filePointer := range templateFiles {
		filename := filePointer.Name()
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
	if runFlags.Bool("keep-kubernetes-template-files") {
		fmt.Println("=> Leaving the templated files, like you asked.")
	} else {
		os.RemoveAll(repoConfig.PWD + "/.kubedeploy-temp")
	}
}

func runConsulTemplate(filename string) string {
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr != "" {
		vaultAddr = fmt.Sprintf("--vault-renew-token=false --vault-retry=false --vault-addr %s", vaultAddr)
		os.Setenv("SECRETS_LOCATION", repoConfig.Namespace)
	}
	consulTemplateArgs := fmt.Sprintf("%s -template %s -once -dry", vaultAddr, filename)

	// the map which will contain all environment variables to be set before running consul-template
	envMap := make(map[string]string)

	// Include the template freebie variables
	envMap["KD_RELEASE_NAME"] = repoConfig.ReleaseName
	envMap["KD_APP_NAME"] = repoConfig.Application.Name + "-" + repoConfig.GitBranch
	envMap["KD_KUBERNETES_NAMESPACE"] = repoConfig.Namespace
	envMap["KD_GIT_BRANCH"] = repoConfig.GitBranch
	envMap["KD_GIT_SHA"] = repoConfig.GitSHA
	envMap["KD_IMAGE_FULL_PATH"] = repoConfig.ImageFullPath
	envMap["KD_IMAGE_TAG"] = repoConfig.ImageTag

	environmentToBranchMappings := map[string][]string{
		"production":  []string{"production"},
		"staging":     []string{"master", "staging"},
		"development": []string{"else", "dev"},
		"acceptance":  []string{"acceptance"},
	}

	headingToLookFor := environmentToBranchMappings[repoConfig.Namespace]
	branchNameHeadings := repoConfig.Application.KubernetesTemplate.BranchVariables
	re := regexp.MustCompile(fmt.Sprintf("(%s),?", strings.Join(headingToLookFor, "|")))

	// Parse and add the global env vars
	for _, envVar := range repoConfig.Application.KubernetesTemplate.GlobalVariables {
		split := strings.Split(envVar, "=")
		envMap[split[0]] = split[1]
	}

	if runFlags.Bool("debug") {
		fmt.Println("=> Here's the regex I'm going to use for matching branches (templating process): ", re.String())
	}
	// Loop over the branch names we would match with
	// loop over the un-split headings
	for heading := range branchNameHeadings {
		// splitBranches := strings.Split(heading, ",")
		if re.MatchString(heading) {
			for _, envVar := range branchNameHeadings[heading] {
				split := strings.Split(envVar, "=")
				envMap[split[0]] = split[1]
			}
		}
	}

	if runFlags.Bool("debug") {
		fmt.Println(envMap)
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

	if runFlags.Bool("debug") {
		for _, i := range os.Environ() {
			fmt.Println(i)
		}
	}

	output, exitCode := getCommandOutputAndExitCode("consul-template", consulTemplateArgs)
	if exitCode != 0 {
		fmt.Println("=> Oh no, looks like consul-template failed!")
		os.Exit(1)
	}

	return strings.Join(strings.Split(output, "\n")[1:], "\n")
}
