package main

import (
	"fmt"
)

func kubeListDeployments() {
	
}
	

func kubeStartRollout() {
	for _, f := range getKubeTemplateFiles() {
		kubeFileTemplated := runConsulTemplate(repoConfig.Application.PathToKubernetesFiles + "/" + f)
		fmt.Println(kubeFileTemplated)
	}
}