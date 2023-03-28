package main

import (
	"flag"
	"fmt"
	"github.com/banzaicloud/koperator/test/pkg/install"
	"io/ioutil"
	"istio.io/istio/pkg/kube"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"testing"
	"time"
)

var (
	kubeConfigEnvName         = "KUBECONFIG"
	kubeConfigDefaultFilename = "~/.kube/config"
	kubeConfig                string
	cleanup                   bool
	kopNoPrereqs              bool
	kafkaClusterManifestDir   = "../../config/samples"
)

type kubeClient struct {
	KubeConfigFile string
	Config         *rest.Config
	Getter         genericclioptions.RESTClientGetter
	Clientset      *kubernetes.Clientset
	ExtendedClient kube.ExtendedClient
}

var kClient kubeClient

func init() {
	flag.StringVar(&kubeConfig, "kubeconfig", "", "The kubeconfig to for the cluster to test")
	flag.BoolVar(&cleanup, "cleanup", false, "cleanup install after test.")
	flag.BoolVar(&kopNoPrereqs, "noprereqs", false, "install koperator prerequisites")
}

func getKubeConfigFileName() string {
	if kubeConfig != "" {
		return kubeConfig
	}
	kubeConfigFilename := os.Getenv(kubeConfigEnvName)
	// Fallback to default kubeconfig file location if no env variable set
	if kubeConfigFilename == "" {
		kubeConfigFilename = kubeConfigDefaultFilename
	}
	return kubeConfigFilename
}

// getKubeConfig is a wrapper function to get *rest.Config and set proper QPS and Burst time.
func getKubeRestConfig(kubeconfigPath string) (*rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	// Set higher QPS and Burst for to avoid throttling messages
	config.QPS = 200
	config.Burst = 200

	return config, nil
}

func getClientCfgFromKubeconfigFile(kubeconfigPath string) (clientcmd.ClientConfig, error) {
	kconfContents, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		fmt.Printf("Error reading kubeconfig file: %v", err)
		return nil, err
	}
	return clientcmd.NewClientConfigFromBytes(kconfContents)
}

func TestMain(m *testing.M) {
	var err error
	flag.Parse()
	kClient.KubeConfigFile = getKubeConfigFileName()
	kClient.Config, err = getKubeRestConfig(kClient.KubeConfigFile)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	kClient.Clientset, err = kubernetes.NewForConfig(kClient.Config)
	if err != nil {
		os.Exit(1)
	}
	kubeClientConfig, err := getClientCfgFromKubeconfigFile(kClient.KubeConfigFile)
	if err != nil {
		os.Exit(1)
	}
	kClient.ExtendedClient, err = kube.NewExtendedClient(kubeClientConfig, "")
	if err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// Perform the koperator installation steps documented in the koperator installation guide
// https://banzaicloud.com/docs/supertubes/kafka-operator/install-kafka-operator/#install-cert-manager
// Prereqs:
//   - kubeconfig for cluster in normal dir/env places or passed via `--kubeconfig` param
//   - koperator CRDs loaded into the k8s cluster
//   - this method in install_dependencies.go fails due to unable to turn off validation in the yaml apply API used
func TestInstall(t *testing.T) {

	if !kopNoPrereqs {
		installObj := install.NewInstall("charts", "manifests", kClient.KubeConfigFile)
		if installObj == nil {
			t.Fatalf("Failed to initialize installer")
		}
		// setup dependencies -- don't fail on error
		status, err := installObj.InstallProfile(install.BasicDependenciesProfile, false)
		if err != nil {
			t.Logf("error returned for dependencies install: %v", err)
		}
		if status != nil {
			for _, packageStatus := range status.PackageStatus {
				statusStr := "success"
				if packageStatus.Error != nil {
					statusStr = fmt.Sprintf("error: %v", packageStatus.Error)
				}
				t.Logf("package: %s install status: %s", packageStatus.Name, statusStr)
			}
		}
		if cleanup {
			defer InstallCleanup(t, installObj, status)
		}
	}
	kinstallObj := install.NewInstall("charts", kafkaClusterManifestDir, kClient.KubeConfigFile)
	if kinstallObj == nil {
		t.Fatalf("Failed to initialize installer")
	}
	// setup koperator and kafkaCluster -- fail on error
	kstatus, err := kinstallObj.InstallProfile(install.BasicKafkaProfile, true)
	if err != nil {
		t.Logf("error returned for dependencies install: %v", err)
	}
	if kstatus != nil {
		for _, packageStatus := range kstatus.PackageStatus {
			statusStr := "success"
			if packageStatus.Error != nil {
				statusStr = fmt.Sprintf("error: %v", packageStatus.Error)
				t.Logf("package: %s install failed: %s", packageStatus.Name, statusStr)
				t.Fail()
			}
			t.Logf("package: %s install status: %s", packageStatus.Name, statusStr)
		}
	}
	if cleanup {
		defer InstallCleanup(t, kinstallObj, kstatus)
	}
	time.Sleep(100 * time.Second)
	// TODO:  Do stuff to check installation

	/* OLD method
		installDepends := install.NewInstallDependencies("charts", "manifests", kClient.KubeConfigFile)
		if cleanup {
			defer InstallCleanup(t, installDepends)
		}
		t.Logf("Installing cert-manager")
		if err := installDepends.InstallCertManager("cert-manager"); err != nil {
			t.Errorf("Failed to install cert-manager: %v", err)
		}
		t.Logf("Installing zookeeper operator")
		if err := installDepends.InstallZookeeperOperator("zookeeper"); err != nil {
			t.Errorf("Failed to install zookeeper operator: %v", err)
		}
		t.Logf("Installing zookeeper cluster")
		if err := kClient.ExtendedClient.ApplyYAMLFiles("zookeeper", zookeeperClusterManifestFile); err != nil {
			t.Errorf("Failed to install zookeeper cluster: %v", err)
		}
		t.Logf("Installing prometheus operator")
		if err := kClient.ExtendedClient.ApplyYAMLFiles("default", prometheusOperatorManifestFile); err != nil {
			t.Errorf("Failed to install prometheus operator: %v", err)
		}

	t.Logf("Installing kafka operator")
	if err := installDepends.InstallKafkaOperator("kafka"); err != nil {
		t.Errorf("Failed to install kafka operator: %v", err)
	}
	t.Logf("Installing kafka cluster")
	if err := kClient.ExtendedClient.ApplyYAMLFiles("kafka", kafkaClusterManifestFile); err != nil {
		t.Errorf("Failed to install kafkaCluster: %v", err)
	}

	*/

}

func TestUninstall(t *testing.T) {
	kinstallObj := install.NewInstall("charts", kafkaClusterManifestDir, kClient.KubeConfigFile)
	if kinstallObj == nil {
		t.Fatalf("Failed to initialize installer")
	}
	kstatus := install.InstallStatus{
		Profile: install.BasicKafkaProfile,
	}
	if err := kinstallObj.Uninstall(&kstatus); err != nil {
		t.Logf("Failed to uninstall kafka profile: %v", err)
	}

	if !kopNoPrereqs {
		installObj := install.NewInstall("charts", "manifests", kClient.KubeConfigFile)
		if installObj == nil {
			t.Fatalf("Failed to initialize installer")
		}
		status := install.InstallStatus{
			Profile: install.BasicDependenciesProfile,
		}
		if err := kinstallObj.Uninstall(&status); err != nil {
			t.Logf("Failed to uninstall kafka profile: %v", err)
		}
	}
}

func InstallCleanup(t *testing.T, installObj *install.Install, status *install.InstallStatus) {
	t.Logf("Cleaning up installation for profile %d", status.Profile)
	if err := installObj.Uninstall(status); err != nil {
		t.Logf("Uninstall failed for profile %d: %v", status.Profile, err)
	}
}
