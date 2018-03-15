package kubeapi

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func ParseKubeFile(fileContents []byte) runtime.Object {

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(fileContents, nil, nil)

	if err != nil {
		fmt.Println(fmt.Sprintf("=> Error while decoding YAML into kube object. Err was: %s", err))
		return nil
	}

	return obj
}
