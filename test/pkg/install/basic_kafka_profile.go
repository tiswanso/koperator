package install

import (
	"fmt"
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
	installDepends := NewInstallDependencies(config.ChartDir, config.ManifestDir,
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
		koperatorStatus.Error = fmt.Errorf("failed to install kafka operator: %v", err)
	}
	status = append(status, koperatorStatus)
	if koperatorStatus.Error != nil && failOnError {
		return status, nil
	}
	// TODO: wait and check for koperator up
	time.Sleep(30 * time.Second)

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
	installDepends := NewInstallDependencies(config.ChartDir, config.ManifestDir,
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
