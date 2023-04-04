// Copyright © 2023 Cisco Systems, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	KoperatorImageRepo             = "local/kafka-operator"
	KoperatorImageTag              = "ci-test"
	zookeeperClusterManifestFile   = "manifests/zookeeperCluster.yaml"
	PrometheusOperatorChartTgz     = "kube-prometheus-stack-45.8.0.tgz"
	prometheusOperatorManifestFile = "manifests/prometheus-operator-bundle.yaml"
)

type InstallComponents struct {
	InstallConfig
	//Getter   genericclioptions.RESTClientGetter
}

func NewInstallComponents(chartDir, manifestDir, kubeConfig string, extendedClient istio.ExtendedClient) *InstallComponents {
	return &InstallComponents{
		InstallConfig{
			ChartDir:       chartDir,
			ManifestDir:    manifestDir,
			KubeConfig:     kubeConfig,
			extendedClient: extendedClient,
		},
	}
}

func (id *InstallComponents) InstallCertManagerCRDs() error {
	if err := id.InstallConfig.extendedClient.ApplyYAMLFiles("", CertManagerCRDManifest); err != nil {
		return fmt.Errorf("failed to install cert-manager CRDs: %v", err)
	}
	return nil
}
func (id *InstallComponents) InstallCertManager(namespace string) error {
	if err := id.InstallCertManagerCRDs(); err != nil {
		return err
	}
	chartFile := filepath.Join(id.ChartDir, CertManagerChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.InstallChart("cert-manager",
		namespace, chartFile, getter)
}
func (id *InstallComponents) UninstallCertManager(namespace string) error {
	chartFile := filepath.Join(id.ChartDir, CertManagerChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.UninstallChart("cert-manager",
		namespace, chartFile, getter)
}

func (id *InstallComponents) InstallZookeeperOperator(namespace string) error {
	chartFile := filepath.Join(id.ChartDir, ZookeeperOperatorChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.InstallChart("zookeeper",
		namespace, chartFile, getter)
}
func (id *InstallComponents) UninstallZookeeperOperator(namespace string) error {
	chartFile := filepath.Join(id.ChartDir, ZookeeperOperatorChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.UninstallChart("zookeeper",
		namespace, chartFile, getter)
}
func (id *InstallComponents) InstallZookeeperCluster(namespace string) error {
	if err := id.InstallConfig.extendedClient.ApplyYAMLFiles("zookeeper", zookeeperClusterManifestFile); err != nil {
		return fmt.Errorf("failed to install zookeeper cluster: %v", err)
	}
	return nil
}
func (id *InstallComponents) UninstallZookeeperCluster(namespace string) error {
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

func (id *InstallComponents) InstallPrometheusOperator(namespace string) error {
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
func (id *InstallComponents) UninstallPrometheusOperator(namespace string) error {
	chartFile := filepath.Join(id.ChartDir, PrometheusOperatorChartTgz)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.UninstallChart("prometheus",
		namespace, chartFile, getter)
}

// putting this here for now since it's the same routine
func (id *InstallComponents) InstallKafkaOperatorCRDs() error {
	// install CRDs
	// NOTE:
	// Both methods fail due to validation in k8s api because KafkaCluster CRD's annotation length is
	// too long.  The only way to apply the CRD is to disable validation which is buried in this extendedClient API
	/*  // this method is using a dir of separate manifest files
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
	// this method is a single file with all the crd manifests
	if err := id.InstallConfig.extendedClient.ApplyYAMLFiles("", KoperatorCrdsManifest); err != nil {
		return fmt.Errorf("failed to install kafka operator CRDs: %v", err)
	}
	return nil
}

func (id *InstallComponents) InstallKafkaOperator(namespace string) error {
	if err := id.InstallKafkaOperatorCRDs(); err != nil {
		// continue with failure to allow for CRD install as prereq
		// return fmt.Errorf("Failed to install Koperator CRDs: %v", err)
	}
	getter := kube.GetConfig(id.KubeConfig, "", namespace)

	// use image locallybuilt from changeset
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
func (id *InstallComponents) UninstallKafkaOperator(namespace string) error {
	//chartFile := filepath.Join(id.ChartDir, KoperatorChartDir)
	getter := kube.GetConfig(id.KubeConfig, "", namespace)
	return helm.UninstallChart("koperator",
		namespace, KoperatorChartDir, getter)
}
func (id *InstallComponents) InstallKafkaCluster(namespace string, manifest string) error {
	if err := id.InstallConfig.extendedClient.ApplyYAMLFiles(namespace, manifest); err != nil {
		return fmt.Errorf("failed to install kafkaCluster: %v", err)
	}
	return nil
}
func (id *InstallComponents) UninstallKafkaCluster(namespace string, manifest string) error {
	if err := id.InstallConfig.extendedClient.DeleteYAMLFiles(namespace, manifest); err != nil {
		return fmt.Errorf("failed to uninstall KafkaCluster: %v", err)
	}
	return nil
}

func (id *InstallComponents) CreateNs(namespace string) error {
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
