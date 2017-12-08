package main

import (
	"bytes"
	"os"
	"fmt"
	"strings"
	"text/template"
)

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

	envMap := make(map[string]string)
	for _, e := range repoConfig.Application.KubernetesTemplateVariables {
		split := strings.Split(e, "=")
		envMap[split[0]] = split[1]
	}

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