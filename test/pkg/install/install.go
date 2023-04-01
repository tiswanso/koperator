// Copyright © 2023 Cisco Systems, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package install

import (
	"fmt"
	"github.com/banzaicloud/koperator/test/pkg/kube"
	"io/ioutil"
	istio "istio.io/istio/pkg/kube"
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
	extendedClient istio.ExtendedClient
	KubeClient     kube.KubeClient
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
	kubeClient, _ := kube.NewKubeClient()
	return &Install{
		Config: InstallConfig{
			ChartDir:       chartDir,
			ManifestDir:    manifestDir,
			KubeConfig:     kubeConfig,
			extendedClient: nil,
			KubeClient:     kubeClient,
		},
	}
}

func (i *Install) GetKubeClient() (kube.KubeClient, error) {
	return i.Config.KubeClient, nil
}

func (i *Install) GetExtendedClient() (istio.ExtendedClient, error) {
	if i.Config.extendedClient == nil {
		kubeClientConfig, err := getClientCfgFromKubeconfigFile(i.Config.KubeConfig)
		if err != nil {
			return nil, err
		}
		i.Config.extendedClient, err = istio.NewExtendedClient(kubeClientConfig, "")
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
