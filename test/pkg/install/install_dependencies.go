package install

import (
	"fmt"
	"github.com/banzaicloud/koperator/test/pkg/helm"
	"helm.sh/helm/v3/pkg/kube"
	istio "istio.io/istio/pkg/kube"
	"path/filepath"
)

var (
	CertManagerChartTgz            = "cert-manager-v1.6.2.tgz"
	ZookeeperOperatorChartTgz      = "zookeeper-operator-0.2.14.tgz"
	KoperatorChartDir              = "../../charts/kafka-operator"
	zookeeperClusterManifestFile   = "manifests/zookeeperCluster.yaml"
	prometheusOperatorManifestFile = "manifests/prometheus-operator-bundle.yaml"
)

type InstallDependencies struct {
	InstallConfig
	//Getter   genericclioptions.RESTClientGetter
}

func NewInstallDependencies(chartDir, manifestDir, kubeConfig string, extendedClient istio.ExtendedClient) *InstallDependencies {
	return &InstallDependencies{
		InstallConfig{
			ChartDir:       chartDir,
			ManifestDir:    manifestDir,
			KubeConfig:     kubeConfig,
			extendedClient: extendedClient,
		},
	}
}

func (id *InstallDependencies) InstallCertManager(namespace string) error {
	chartFile := filepath.Join(id.ChartDir, CertManagerChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.InstallChart("cert-manager",
		namespace, chartFile, getter)
}
func (id *InstallDependencies) UninstallCertManager(namespace string) error {
	chartFile := filepath.Join(id.ChartDir, CertManagerChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.UninstallChart("cert-manager",
		namespace, chartFile, getter)
}

func (id *InstallDependencies) InstallZookeeperOperator(namespace string) error {
	chartFile := filepath.Join(id.ChartDir, ZookeeperOperatorChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.InstallChart("zookeeper",
		namespace, chartFile, getter)
}
func (id *InstallDependencies) UninstallZookeeperOperator(namespace string) error {
	chartFile := filepath.Join(id.ChartDir, ZookeeperOperatorChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.UninstallChart("zookeeper",
		namespace, chartFile, getter)
}
func (id *InstallDependencies) InstallZookeeperCluster(namespace string) error {
	if err := id.InstallConfig.extendedClient.ApplyYAMLFiles("zookeeper", zookeeperClusterManifestFile); err != nil {
		return fmt.Errorf("failed to install zookeeper cluster: %v", err)
	}
	return nil
}
func (id *InstallDependencies) UninstallZookeeperCluster(namespace string) error {
	if err := id.InstallConfig.extendedClient.DeleteYAMLFiles("zookeeper", zookeeperClusterManifestFile); err != nil {
		return fmt.Errorf("failed to uninstall zookeeper cluster: %v", err)
	}
	return nil
}

// TODO: switch this to helm
func (id *InstallDependencies) InstallPrometheusOperator(namespace string) error {
	if err := id.InstallConfig.extendedClient.ApplyYAMLFiles(namespace, prometheusOperatorManifestFile); err != nil {
		return fmt.Errorf("failed to install prometheus operator: %v", err)
	}
	return nil
}
func (id *InstallDependencies) UninstallPrometheusOperator(namespace string) error {
	if err := id.InstallConfig.extendedClient.DeleteYAMLFiles(namespace, prometheusOperatorManifestFile); err != nil {
		return fmt.Errorf("failed to uninstall prometheus operator: %v", err)
	}
	return nil
}

// putting this here for now since it's the same routine
func (id *InstallDependencies) InstallKafkaOperator(namespace string) error {
	//var chartFile = filepath.Join(id.ChartDir, KoperatorChartDir)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.InstallChart("koperator",
		namespace, KoperatorChartDir, getter)
}
func (id *InstallDependencies) UninstallKafkaOperator(namespace string) error {
	//chartFile := filepath.Join(id.ChartDir, ZookeeperOperatorChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.UninstallChart("koperator",
		namespace, KoperatorChartDir, getter)
}
func (id *InstallDependencies) InstallKafkaCluster(namespace string, manifest string) error {
	if err := id.InstallConfig.extendedClient.ApplyYAMLFiles(namespace, manifest); err != nil {
		return fmt.Errorf("failed to install kafkaCluster: %v", err)
	}
	return nil
}
func (id *InstallDependencies) UninstallKafkaCluster(namespace string, manifest string) error {
	if err := id.InstallConfig.extendedClient.DeleteYAMLFiles(namespace, manifest); err != nil {
		return fmt.Errorf("failed to uninstall KafkaCluster: %v", err)
	}
	return nil
}
