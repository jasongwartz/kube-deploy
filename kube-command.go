package main

import (
	"os"
	"fmt"
	"io/ioutil"
)

func kubeListDeployments() {
	
}
	

func kubeStartRollout() {
	os.MkdirAll(repoConfig.PWD + "/.kubedeploy-temp", 0755)

	for _, f := range getKubeTemplateFiles() {
		fmt.Printf("=> Generating YAML from template for %s\n", f)
		kubeFileTemplated := runConsulTemplate(repoConfig.Application.PathToKubernetesFiles + "/" + f)

		tempFilePath := repoConfig.PWD + "/.kubedeploy-temp/" + f
		err := ioutil.WriteFile(tempFilePath, []byte(kubeFileTemplated), 0644)
		if err != nil {
			fmt.Println(err)
		}

		streamAndGetCommandOutputAndExitCode("kubectl", fmt.Sprintf("apply -f %s", tempFilePath))
	}

	os.RemoveAll(repoConfig.PWD + "/.kubedeploy-temp")
}