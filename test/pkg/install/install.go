package install

import (
	"fmt"
	"io/ioutil"
	"istio.io/istio/pkg/kube"
	"k8s.io/client-go/tools/clientcmd"
)

type Profile int

const (
	BasicDependenciesProfile Profile = iota
	BasicKafkaProfile
)

type InstallConfig struct {
	ChartDir       string
	ManifestDir    string
	KubeConfig     string
	extendedClient kube.ExtendedClient
}

type Install struct {
	Config InstallConfig
}

type PackageInstallStatus struct {
	Name      string
	Namespace string
	Error     error
}

type InstallStatus struct {
	Profile       Profile
	failOnError   bool
	PackageStatus []PackageInstallStatus // ordered list of package install statuses
}

type ProfileInstaller interface {
	Install(config InstallConfig, failOnError bool) ([]PackageInstallStatus, error)
	Uninstall(InstallConfig, []PackageInstallStatus) error
}

var installProfiles = map[Profile]ProfileInstaller{
	BasicDependenciesProfile: BasicDependenciesInstaller,
	BasicKafkaProfile:        BasicKafkaInstaller,
}

func NewInstall(chartDir, manifestDir, kubeConfig string) *Install {
	return &Install{
		Config: InstallConfig{
			ChartDir:       chartDir,
			ManifestDir:    manifestDir,
			KubeConfig:     kubeConfig,
			extendedClient: nil,
		},
	}
}

func (i *Install) GetExtendedClient() (kube.ExtendedClient, error) {
	if i.Config.extendedClient == nil {
		kubeClientConfig, err := getClientCfgFromKubeconfigFile(i.Config.KubeConfig)
		if err != nil {
			return nil, err
		}
		i.Config.extendedClient, err = kube.NewExtendedClient(kubeClientConfig, "")
		if err != nil {
			return nil, err
		}
	}
	return i.Config.extendedClient, nil
}

func (i *Install) InstallProfile(profile Profile, failOnError bool) (*InstallStatus, error) {
	if installer, ok := installProfiles[profile]; ok {
		var err error
		if i.Config.extendedClient, err = i.GetExtendedClient(); err != nil {
			return nil, err
		}
		packageStatus, err := installer.Install(i.Config, failOnError)
		status := &InstallStatus{
			Profile:       profile,
			failOnError:   failOnError,
			PackageStatus: packageStatus,
		}
		return status, err
	}
	return nil, fmt.Errorf("Invalid Profile")
}

func (i *Install) Uninstall(installStatus *InstallStatus) error {
	if installStatus == nil {
		return fmt.Errorf("Invalid install state object")
	}
	if installer, ok := installProfiles[installStatus.Profile]; ok {
		var err error
		if i.Config.extendedClient, err = i.GetExtendedClient(); err != nil {
			return err
		}
		return installer.Uninstall(i.Config, installStatus.PackageStatus)
	}
	return fmt.Errorf("Install status contains invalid profile, unable to proceed")
}

func getClientCfgFromKubeconfigFile(kubeconfigPath string) (clientcmd.ClientConfig, error) {
	kconfContents, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		fmt.Printf("Error reading kubeconfig file: %v", err)
		return nil, err
	}
	return clientcmd.NewClientConfigFromBytes(kconfContents)
}
