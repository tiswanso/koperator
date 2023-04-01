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
	"path/filepath"
	"time"
)

type BasicKafka string

// BasicKafkaInstaller
// This implements the ProfileInstaller interface
// it installs the koperator as described in
// https://banzaicloud.com/docs/supertubes/kafka-operator/install-kafka-operator/#install-cert-manager
var BasicKafkaInstaller BasicKafka = "Basic Kafka Profile"

var kafkaClusterManifestFile = "simplekafkacluster.yaml"

func (b BasicKafka) Install(config InstallConfig, failOnError bool) ([]PackageInstallStatus, error) {
	installDepends := NewInstallComponents(config.ChartDir, config.ManifestDir,
		config.KubeConfig, config.extendedClient)
	if installDepends == nil {
		return nil, fmt.Errorf("failed to create Install Dependencies object")
	}

	var status []PackageInstallStatus
	koperatorStatus := PackageInstallStatus{
		Name:      "koperator",
		Namespace: "kafka",
		Error:     nil,
	}
	if err := installDepends.CreateNs("kafka"); err != nil {
		koperatorStatus.Error = fmt.Errorf("failed to create kafka namespace: %v", err)
	} else if err := installDepends.InstallKafkaOperator("kafka"); err != nil {
		koperatorStatus.Error = fmt.Errorf("failed to install kafka operator: %w", err)
	}

	// TODO: wait and check for koperator up
	timeout := 30 * time.Second
	if pods, err := kube.WaitForActivePodAny(config.KubeClient.Clientset,
		"kafka", timeout, "app.kubernetes.io/name=kafka-operator"); err != nil {
		koperatorStatus.Error = fmt.Errorf("kafka operator pods (%d) failed to come up: %w", err, len(pods.Items))
	}
	status = append(status, koperatorStatus)
	if koperatorStatus.Error != nil && failOnError {
		return status, nil
	}
	// Seem to still have to wait for the webhook handler to be ready in koperator
	// Likely koperator needs a readiness probe implementation to make this deterministic
	time.Sleep(10 * time.Second)

	kafkaClusterStatus := PackageInstallStatus{
		Name:      "KafkaCluster",
		Namespace: "kafka",
		Error:     nil,
	}
	if err := installDepends.InstallKafkaCluster("kafka",
		filepath.Join(config.ManifestDir, kafkaClusterManifestFile)); err != nil {
		kafkaClusterStatus.Error = fmt.Errorf("failed to install kafkaCluster: %v", err)
	}
	status = append(status, kafkaClusterStatus)
	if kafkaClusterStatus.Error != nil && failOnError {
		return status, nil
	}

	return nil, nil
}
func (b BasicKafka) Uninstall(config InstallConfig, packages []PackageInstallStatus) error {
	// uninstall everything for now... continue on error
	installDepends := NewInstallComponents(config.ChartDir, config.ManifestDir,
		config.KubeConfig, config.extendedClient)
	if installDepends == nil {
		return fmt.Errorf("failed to create Install Dependencies object")
	}

	// todo: determine if we can pack errors into one
	var lastError error = nil
	if err := installDepends.UninstallKafkaCluster("kafka",
		filepath.Join(config.ManifestDir, kafkaClusterManifestFile)); err != nil {
		lastError = fmt.Errorf("failed to delete kafkaCluster: %v", err)
	}
	// TODO: wait and check for KafkaCluster removed
	time.Sleep(30 * time.Second)
	if err := installDepends.UninstallKafkaOperator("kafka"); err != nil {
		lastError = fmt.Errorf("failed to uninstall kafka operator: %v", err)
	}

	return lastError
}
