package install

import (
	"context"
	"fmt"
	"github.com/banzaicloud/koperator/test/pkg/helm"
	"helm.sh/helm/v3/pkg/kube"
	istio "istio.io/istio/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"path/filepath"
)

var (
	CertManagerChartTgz            = "cert-manager-v1.6.2.tgz"
	CertManagerCRDManifest         = "manifests/cert-manager.crds.yaml"
	ZookeeperOperatorChartTgz      = "zookeeper-operator-0.2.14.tgz"
	KoperatorCrdsDir               = "../../config/base/crds"
	KoperatorCrdsManifest          = "manifests/kafka-operator-v0.22.0.crds.yaml"
	KoperatorChartDir              = "../../charts/kafka-operator"
	KoperatorImageRepo             = "tiswanso/kafka-operator"
	KoperatorImageTag              = "openshift-support-2"
	zookeeperClusterManifestFile   = "manifests/zookeeperCluster.yaml"
	PrometheusOperatorChartTgz     = "kube-prometheus-stack-45.8.0.tgz"
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

func (id *InstallDependencies) InstallCertManagerCRDs() error {
	if err := id.InstallConfig.extendedClient.ApplyYAMLFiles("", CertManagerCRDManifest); err != nil {
		return fmt.Errorf("failed to install cert-manager CRDs: %v", err)
	}
	return nil
}
func (id *InstallDependencies) InstallCertManager(namespace string) error {
	if err := id.InstallCertManagerCRDs(); err != nil {
		return err
	}
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

func addEnableKVObjToMap(intfmap map[string]interface{}, key string, value bool) {
	intfmap[key] = map[string]interface{}{
		"enabled": value,
	}
}

// TODO: switch this to helm
func (id *InstallDependencies) InstallPrometheusOperator(namespace string) error {
	// Do the equivalent of
	// helm install prometheus --namespace default prometheus-community/kube-prometheus-stack \
	//--set prometheusOperator.createCustomResource=true \
	//--set defaultRules.enabled=false \
	//--set alertmanager.enabled=false \
	//--set grafana.enabled=false \
	//--set kubeApiServer.enabled=false \
	//--set kubelet.enabled=false \
	//--set kubeControllerManager.enabled=false \
	//--set coreDNS.enabled=false \
	//--set kubeEtcd.enabled=false \
	//--set kubeScheduler.enabled=false \
	//--set kubeProxy.enabled=false \
	//--set kubeStateMetrics.enabled=false \
	//--set nodeExporter.enabled=false \
	//--set prometheus.enabled=false

	vals := map[string]interface{}{
		"prometheusOperator": map[string]interface{}{
			"createCustomResource": "true",
			"sentinel": map[string]interface{}{
				"masterName": "BigMaster",
				"pass":       "random",
				"addr":       "localhost",
				"port":       "26379",
			},
		},
	}
	addEnableKVObjToMap(vals, "defaultRules", false)
	addEnableKVObjToMap(vals, "alertmanager", false)
	addEnableKVObjToMap(vals, "grafana", false)
	addEnableKVObjToMap(vals, "kubeApiServer", false)
	addEnableKVObjToMap(vals, "kubelet", false)
	addEnableKVObjToMap(vals, "kubeControllerManager", false)
	addEnableKVObjToMap(vals, "coreDNS", false)
	addEnableKVObjToMap(vals, "kubeEtcd", false)
	addEnableKVObjToMap(vals, "kubeScheduler", false)
	addEnableKVObjToMap(vals, "kubeProxy", false)
	addEnableKVObjToMap(vals, "kubeStateMetrics", false)
	addEnableKVObjToMap(vals, "nodeExporter", false)
	addEnableKVObjToMap(vals, "prometheus", false)

	chartFile := filepath.Join(id.ChartDir, PrometheusOperatorChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.InstallChartWithValues("prometheus",
		namespace, chartFile, getter, vals)

	return nil
}
func (id *InstallDependencies) UninstallPrometheusOperator(namespace string) error {
	chartFile := filepath.Join(id.ChartDir, PrometheusOperatorChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.UninstallChart("prometheus",
		namespace, chartFile, getter)
}

// putting this here for now since it's the same routine
func (id *InstallDependencies) InstallKafkaOperatorCRDs() error {
	// install CRDs
	/*
		entries, err := os.ReadDir(KoperatorCrdsDir)
		if err != nil {
			return err
		}

		var files []string
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, filepath.Join(KoperatorCrdsDir, e.Name()))
			}
		}
		if err := id.InstallConfig.extendedClient.ApplyYAMLFiles("", files...); err != nil {
			return fmt.Errorf("failed to install kafka operator CRDs: %v", err)
		}
	*/
	// This fails due to validation in k8s api because KafkaCluster CRD's annotation length is
	// too long.  The only way to apply the CRD is to disable validation which is buried in this extendedClient API
	if err := id.InstallConfig.extendedClient.ApplyYAMLFiles("", KoperatorCrdsManifest); err != nil {
		return fmt.Errorf("failed to install kafka operator CRDs: %v", err)
	}
	return nil
}

func (id *InstallDependencies) InstallKafkaOperator(namespace string) error {
	if err := id.InstallKafkaOperatorCRDs(); err != nil {
		// continue with failure to allow for CRD install as prereq
		// return fmt.Errorf("Failed to install Koperator CRDs: %v", err)
	}
	getter := kube.GetConfig(id.KubeConfig, "", namespace)

	// use image built from changeset
	vals := map[string]interface{}{
		"operator": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": KoperatorImageRepo,
				"tag":        KoperatorImageTag,
			},
		},
	}

	return helm.InstallChartWithValues("koperator",
		namespace, KoperatorChartDir, getter, vals)
}
func (id *InstallDependencies) UninstallKafkaOperator(namespace string) error {
	//chartFile := filepath.Join(id.ChartDir, KoperatorChartDir)
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

func (id *InstallDependencies) CreateNs(namespace string) error {
	config, err := clientcmd.BuildConfigFromFlags("", id.KubeConfig)
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		nsSpec := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = clientset.CoreV1().Namespaces().Create(context.Background(), nsSpec, metav1.CreateOptions{})
	}
	return err
}
