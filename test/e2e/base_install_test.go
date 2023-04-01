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

package main

import (
	"context"
	"flag"
	"fmt"
	koperatorv1beta1 "github.com/banzaicloud/koperator/api/v1beta1"
	"github.com/banzaicloud/koperator/test/pkg/install"
	"github.com/banzaicloud/koperator/test/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"testing"
	"time"
)

var (
	kubeConfig              string
	cleanup                 bool
	kopNoPrereqs            bool
	kafkaClusterManifestDir = "../../config/samples"
)

var kClient kube.KubeClient

func init() {
	// kubeconfig is setup by test/pkg/kube
	flag.BoolVar(&cleanup, "cleanup", false, "cleanup install after test.")
	flag.BoolVar(&kopNoPrereqs, "noprereqs", false, "install koperator prerequisites")
}

func TestMain(m *testing.M) {
	var err error

	flag.Parse()

	if kClient, err = kube.NewKubeClient(); err != nil {
		fmt.Printf("Failed setting up test package KubeClient: %v", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// Perform the koperator installation steps documented in the koperator installation guide
// https://banzaicloud.com/docs/supertubes/kafka-operator/install-kafka-operator
// Prereqs:
//   - kubeconfig for cluster in normal dir/env places or passed via `--kubeconfig` param
//   - koperator CRDs loaded into the k8s cluster
//   - this method in install_components.go fails due to unable to turn off validation in the yaml apply API used
func TestInstall(t *testing.T) {

	if !kopNoPrereqs {
		t.Run("Install basic dependencies", InstallBasicDependencies)
	}
	t.Run("Install koperator and simple Kafka cluster", InstallBasicKafka)

	t.Run("Check KafkaCluster", CheckKafkaClusterStatus)

	if cleanup {
		t.Run("Uninstall Kafka", UninstallKafka)

		if !kopNoPrereqs {
			t.Run("Uninstall Dependencies", UninstallDependencies)
		}
	}
}

func UninstallKafka(t *testing.T) {
	kinstallObj := install.NewInstall("charts", kafkaClusterManifestDir, kClient.KubeConfigFile)
	if kinstallObj == nil {
		t.Fatalf("Failed to initialize installer")
	}
	kstatus := install.InstallStatus{
		Profile: install.BasicKafkaProfile,
	}
	if err := kinstallObj.Uninstall(&kstatus); err != nil {
		t.Errorf("Failed to uninstall kafka profile: %v", err)
	}
}
func UninstallDependencies(t *testing.T) {
	installObj := install.NewInstall("charts", "manifests", kClient.KubeConfigFile)
	if installObj == nil {
		t.Fatalf("Failed to initialize installer")
	}
	status := install.InstallStatus{
		Profile: install.BasicDependenciesProfile,
	}
	if err := installObj.Uninstall(&status); err != nil {
		t.Errorf("Failed to uninstall basic dependencies profile: %v", err)
	}
}

/*
   This is covered via go test -v -run=TestInstall/"Check KafkaCluster" --kubeconfig <kconf>
func TestGetKafkaCluster(t *testing.T) {
	kcluster, err := GetKafkaCluster("kafka", "kafka")
	if err != nil {
		t.Errorf("failed to get cluster: %v", err)
	}
	t.Logf("Found cluster %s, status %v", kcluster.Name, kcluster.Status.State)

	CheckKafkaClusterStatus(t)
}
*/

func InstallBasicDependencies(t *testing.T) {
	installObj := install.NewInstall("charts", "manifests", kClient.KubeConfigFile)
	if installObj == nil {
		t.Fatalf("Failed to initialize installer")
	}
	// setup dependencies -- don't fail on error
	status, err := installObj.InstallProfile(install.BasicDependenciesProfile, false)
	if err != nil {
		t.Errorf("error returned for dependencies install: %v", err)
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
}

func InstallBasicKafka(t *testing.T) {
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
}

func CheckKafkaClusterStatus(t *testing.T) {
	var kcluster *koperatorv1beta1.KafkaCluster
	var err error

	reconcileWaitTimeout := 3 * time.Minute
	for startTime := time.Now(); time.Since(startTime) < reconcileWaitTimeout; {
		if kcluster, err = GetKafkaCluster("kafka", "kafka"); err != nil {
			t.Logf("failed to get cluster: %v", err)
		} else {
			if kcluster.Status.State == koperatorv1beta1.KafkaClusterRunning {
				t.Logf("Found cluster %s, status %v", kcluster.Name, kcluster.Status.State)
				break
			}
			t.Logf("Found cluster %s, status %v is not %s",
				kcluster.Name, kcluster.Status.State, koperatorv1beta1.KafkaClusterRunning)
		}
		t.Logf("waiting 10s and retrying")
		time.Sleep(10 * time.Second)
	}
	if kcluster != nil {
		t.Logf("Done polling cluster status: cluster %s, status %v", kcluster.Name, kcluster.Status.State)
		if kcluster.Status.State != koperatorv1beta1.KafkaClusterRunning {
			t.Errorf("KafkaCluster %s, did not reach %s; status is %s ",
				kcluster.Name, koperatorv1beta1.KafkaClusterRunning, kcluster.Status.State)
		}

		if err := CheckBrokers(t, kcluster); err != nil {
			t.Errorf("broker check failed for the KafkaCluster: %v", err)
		}
	} else {
		t.Errorf("Failed to retrieve KafkaCluster")
		return
	}

}

func GetKafkaCluster(name, namespace string) (*koperatorv1beta1.KafkaCluster, error) {
	kClusterGVR := schema.GroupVersionResource{
		Group:    koperatorv1beta1.GroupVersion.Group,
		Version:  koperatorv1beta1.GroupVersion.Version,
		Resource: "kafkaclusters",
	}
	resp, err := kClient.DynamicClient.Resource(kClusterGVR).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// Convert the unstructured object to cluster.
	unstructured := resp.UnstructuredContent()
	var kcluster koperatorv1beta1.KafkaCluster
	err = runtime.DefaultUnstructuredConverter.
		FromUnstructured(unstructured, &kcluster)
	return &kcluster, err
}

func CheckBrokers(t *testing.T, kcluster *koperatorv1beta1.KafkaCluster) error {
	if kcluster == nil {
		return fmt.Errorf("no KafkaCluster to get brokers")
	}
	for _, broker := range kcluster.Spec.Brokers {
		labels := []string{"app=kafka", "brokerId=" + fmt.Sprintf("%d", broker.Id)}
		pods, err := kube.WaitForActivePodAll(kClient.Clientset, kcluster.Namespace, 120*time.Second, labels...)
		if err != nil {
			t.Errorf("Timed out waiting broker pod %d to become active: %v", broker.Id, err)
		}
		t.Logf("found pod for broker ID %d, pod %s", broker.Id, pods.Items[0].Name)
	}
	return nil
}
