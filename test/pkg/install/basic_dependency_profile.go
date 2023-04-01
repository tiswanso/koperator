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

import "fmt"

type BasicDependencies string

// BasicDependenciesInstaller
// This implements the ProfileInstaller interface
// it installs the koperator dependencies described in
// https://banzaicloud.com/docs/supertubes/kafka-operator/install-kafka-operator/#install-cert-manager
var BasicDependenciesInstaller BasicDependencies = "Basic Dependencies Profile"

func (b BasicDependencies) Install(config InstallConfig, failOnError bool) ([]PackageInstallStatus, error) {
	installDepends := NewInstallComponents(config.ChartDir, config.ManifestDir,
		config.KubeConfig, config.extendedClient)
	if installDepends == nil {
		return nil, fmt.Errorf("failed to create Install Dependencies object")
	}
	var status []PackageInstallStatus
	certManagerStatus := PackageInstallStatus{
		Name:      "cert-manager",
		Namespace: "cert-manager",
		Error:     nil,
	}
	if err := installDepends.CreateNs("cert-manager"); err != nil {
		certManagerStatus.Error = fmt.Errorf("failed to create kafka namespace: %v", err)
	} else if err := installDepends.InstallCertManager("cert-manager"); err != nil {
		certManagerStatus.Error = fmt.Errorf("failed to install cert-manager: %v", err)
	}
	status = append(status, certManagerStatus)
	if certManagerStatus.Error != nil && failOnError {
		return status, nil
	}
	zookeeperOpStatus := PackageInstallStatus{
		Name:      "zookeeper-operator",
		Namespace: "zookeeper",
		Error:     nil,
	}
	if err := installDepends.CreateNs("zookeeper"); err != nil {
		zookeeperOpStatus.Error = fmt.Errorf("failed to create kafka namespace: %v", err)
	} else if err := installDepends.InstallZookeeperOperator("zookeeper"); err != nil {
		zookeeperOpStatus.Error = fmt.Errorf("failed to install zookeeper operator: %v", err)
	}
	status = append(status, zookeeperOpStatus)
	if zookeeperOpStatus.Error != nil && failOnError {
		return status, nil
	}
	zookeeperClusterStatus := PackageInstallStatus{
		Name:      "ZookeeperCluster",
		Namespace: "zookeeper",
		Error:     nil,
	}
	if err := installDepends.InstallZookeeperCluster("zookeeper"); err != nil {
		zookeeperClusterStatus.Error = fmt.Errorf("failed to install zookeeper cluster: %v", err)
	}
	status = append(status, zookeeperClusterStatus)
	if zookeeperClusterStatus.Error != nil && failOnError {
		return status, nil
	}
	prometheusOperatorStatus := PackageInstallStatus{
		Name:      "PrometheusOperator",
		Namespace: "default",
		Error:     nil,
	}
	if err := installDepends.InstallPrometheusOperator("default"); err != nil {
		prometheusOperatorStatus.Error = fmt.Errorf("failed to install prometheus operator: %v", err)
	}
	status = append(status, prometheusOperatorStatus)
	if prometheusOperatorStatus.Error != nil && failOnError {
		return status, nil
	}
	return nil, nil
}
func (b BasicDependencies) Uninstall(config InstallConfig, packages []PackageInstallStatus) error {
	// uninstall everything for now... continue on error
	installDepends := NewInstallComponents(config.ChartDir, config.ManifestDir,
		config.KubeConfig, config.extendedClient)
	if installDepends == nil {
		return fmt.Errorf("failed to create Install Dependencies object")
	}

	var lastError error = nil
	if err := installDepends.UninstallPrometheusOperator("default"); err != nil {
		lastError = fmt.Errorf("failed to delete prometheus operator: %v", err)
	}
	if err := installDepends.UninstallZookeeperCluster("zookeeper"); err != nil {
		lastError = fmt.Errorf("failed to uninstall zookeeper cluster: %v", err)
	}
	if err := installDepends.UninstallZookeeperOperator("zookeeper"); err != nil {
		lastError = fmt.Errorf("failed to uninstall zookeeper operator: %v", err)
	}
	if err := installDepends.UninstallCertManager("cert-manager"); err != nil {
		lastError = fmt.Errorf("failed to uninstall cert-manager: %v", err)
	}

	return lastError
}
