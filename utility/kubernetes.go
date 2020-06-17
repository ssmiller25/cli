package utility

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// ObtainKubeConfig is the function to get the kubeconfig from the cluster
// and save to the file or merge with the existing one
func ObtainKubeConfig(KubeconfigFilename string, civoConfig string, merge bool, clusterName string) error {

	kubeConfig := []byte(civoConfig)

	if merge {
		var err error
		kubeConfig, err = mergeConfigs(KubeconfigFilename, kubeConfig, clusterName)
		if err != nil {
			return err
		}
	}

	if writeErr := writeConfig(KubeconfigFilename, kubeConfig, false, merge, clusterName); writeErr != nil {
		return writeErr
	}
	return nil
}

func mergeConfigs(localKubeconfigPath string, k3sconfig []byte, clusterName string) ([]byte, error) {
	// Create a temporary kubeconfig to store the config of the newly create k3s cluster
	file, err := ioutil.TempFile(os.TempDir(), "civo-temp-*")
	if err != nil {
		return nil, fmt.Errorf("could not generate a temporary file to store the kuebeconfig: %s", err)
	}
	defer file.Close()

	if writeErr := writeConfig(file.Name(), k3sconfig, true, true, clusterName); writeErr != nil {
		return nil, writeErr
	}

	fmt.Printf("Merged with main kubernetes config: %s\n", Green(localKubeconfigPath))

	// Append KUBECONFIGS in ENV Vars
	appendKubeConfigENV := fmt.Sprintf("KUBECONFIG=%s:%s", localKubeconfigPath, file.Name())

	// Merge the two kubeconfigs and read the output into 'data'
	cmd := exec.Command("kubectl", "config", "view", "--merge", "--flatten")
	cmd.Env = append(os.Environ(), appendKubeConfigENV)
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("could not merge kubeconfigs: %s", err)
	}

	// Remove the temporarily generated file
	err = os.Remove(file.Name())
	if err != nil {
		return nil, fmt.Errorf("could not remove temporary kubeconfig file: %s", file.Name())
	}

	return data, nil
}

// Generates config files give the path to file: string and the data: []byte
func writeConfig(path string, data []byte, suppressMessage bool, mergeConfigs bool, clusterName string) error {
	if !suppressMessage {
		fmt.Print("\nAccess your cluster with:\n")
		if mergeConfigs {
			fmt.Printf("kubectl config use-context %s\n", clusterName)
			fmt.Println("kubectl get node")
		} else {
			if strings.Contains(path, ".kube") {
				fmt.Print("kubectl get node\n")
			} else {
				fmt.Printf("KUBECONFIG=%s kubectl get node\n", path)
			}
		}
	}

	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		if err != nil {
			Error(err.Error())
		}
		defer file.Close()
	}

	writeErr := ioutil.WriteFile(path, data, 0600)
	if writeErr != nil {
		return writeErr
	}
	return nil
}
