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

package kube

import (
	"context"
	"flag"
	"fmt"
	koperatorv1beta1 "github.com/banzaicloud/koperator/api/v1beta1"
	"io/ioutil"
	istiokube "istio.io/istio/pkg/kube"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"
	"time"
)

type KubeClient struct {
	KubeConfigFile string
	Config         *rest.Config
	Getter         genericclioptions.RESTClientGetter
	Clientset      *kubernetes.Clientset
	ExtendedClient istiokube.ExtendedClient
	DynamicClient  dynamic.Interface
	Scheme         *runtime.Scheme
}

var (
	kubeConfigEnvName         = "KUBECONFIG"
	kubeConfigDefaultFilename = "~/.kube/config"
	KubeConfig                string
)

func init() {
	flag.StringVar(&KubeConfig, "kubeconfig", "", "The kubeconfig to for the cluster to test")
}

func NewKubeClient() (kClient KubeClient, err error) {

	kClient.KubeConfigFile = getKubeConfigFileName()
	kClient.Config, err = getKubeRestConfig(kClient.KubeConfigFile)
	if err != nil {
		return kClient, fmt.Errorf("[NewKubeClient] failed to get RestConfig: %w", err)
	}
	kClient.Clientset, err = kubernetes.NewForConfig(kClient.Config)
	if err != nil {
		return kClient, fmt.Errorf("[NewKubeClient] failed to get ClientSet: %w", err)
	}
	kubeClientConfig, err := getClientCfgFromKubeConfigFile(kClient.KubeConfigFile)
	if err != nil {
		return kClient, fmt.Errorf("[NewKubeClient] failed to get clientcmd.ClientConfig: %w", err)
	}
	kClient.ExtendedClient, err = istiokube.NewExtendedClient(kubeClientConfig, "")
	if err != nil {
		return kClient, fmt.Errorf("[NewKubeClient] failed to get ExtendedClient: %w", err)
	}
	kClient.DynamicClient, err = getDynamicClient(kClient.Config)
	if err != nil {
		return kClient, fmt.Errorf("[NewKubeClient] failed to get DynamicClient: %w", err)
	}
	// don't think this is needed but leaving in case
	kClient.Scheme = runtime.NewScheme()
	_ = koperatorv1beta1.AddToScheme(kClient.Scheme)
	return kClient, err
}

// IsPodActiveReady
// Checks if the pod is in running state and has ContainersReady condition of true
func IsPodActiveReady(pod v1.Pod) bool {
	if pod.Status.Phase != v1.PodRunning {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.ContainersReady && condition.Status != v1.ConditionTrue {
			return false
		}
		if condition.Type == v1.PodReady && condition.Status != v1.ConditionTrue {
			return false
		}
	}
	return true
}

// WaitForActivePodAny
// use label selector to check if any pods in the selected set are active (running and ready)
// busy wait samples on 1s intervals and times out if total time of function exceeds timeout duration
func WaitForActivePodAny(clientset *kubernetes.Clientset, namespace string, timeout time.Duration, labelSelectors ...string) (*v1.PodList, error) {
	if timeout < 1*time.Second {
		// timeout is too tight for this sort of op so set it to the minimum
		timeout = 1 * time.Second
	}
	var pods *v1.PodList
	var err error
	for curTime := time.Now(); time.Since(curTime) < timeout; {
		pods, err = GetPodsByLabels(clientset, namespace, labelSelectors...)
		if err == nil {
			for _, pod := range pods.Items {
				if IsPodActiveReady(pod) {
					return pods, nil
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return pods, fmt.Errorf("timedout waiting for running pods")
}

// WaitForActivePodAll
// use label selector to check if all pods in the selected set are active (running and ready)
// busy wait samples on 1s intervals and times out if total time of function exceeds timeout duration
func WaitForActivePodAll(clientset *kubernetes.Clientset, namespace string, timeout time.Duration, labelSelectors ...string) (*v1.PodList, error) {
	if timeout < 1*time.Second {
		// timeout is too tight for this sort of op so set it to the minimum
		timeout = 1 * time.Second
	}
	var pods *v1.PodList
	var err error
	for curTime := time.Now(); time.Since(curTime) < timeout; {
		pods, err = GetPodsByLabels(clientset, namespace, labelSelectors...)
		if err == nil && len(pods.Items) > 0 {
			numActivePods := 0
			for _, pod := range pods.Items {
				if !IsPodActiveReady(pod) {
					break
				} else {
					numActivePods++
				}
			}
			if numActivePods == len(pods.Items) {
				// All pods are active
				return pods, nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return pods, fmt.Errorf("timedout waiting for all pods to become active")
}
func GetPodsByLabels(clientset *kubernetes.Clientset, namespace string, labels ...string) (*v1.PodList, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: strings.Join(labels, ","),
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	return pods, err
}

// GetPodsByLabelSelectorObj
// example labelSelector
//
//	labelSelector := metav1.LabelSelector{
//				MatchLabels: map[string]string{
//					"app":      "kafka",
//					"brokerId": fmt.Sprintf("%d", broker.Id),
//				},
//			}
func GetPodsByLabelSelectorObj(clientset *kubernetes.Clientset, namespace string, labelSelector metav1.LabelSelector) (*v1.PodList, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	return pods, err
}

func getKubeConfigFileName() string {
	if KubeConfig != "" {
		return KubeConfig
	}
	KubeConfigFilename := os.Getenv(kubeConfigEnvName)
	// Fallback to default KubeConfig file location if no env variable set
	if KubeConfigFilename == "" {
		KubeConfigFilename = kubeConfigDefaultFilename
	}
	return KubeConfigFilename
}

// getKubeConfig is a wrapper function to get *rest.Config and set proper QPS and Burst time.
func getKubeRestConfig(KubeConfigPath string) (*rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfigPath)
	if err != nil {
		return nil, err
	}
	// Set higher QPS and Burst for to avoid throttling messages
	config.QPS = 200
	config.Burst = 200

	return config, nil
}

func getClientCfgFromKubeConfigFile(KubeConfigPath string) (clientcmd.ClientConfig, error) {
	kconfContents, err := ioutil.ReadFile(KubeConfigPath)
	if err != nil {
		fmt.Printf("Error reading KubeConfig file: %v", err)
		return nil, err
	}
	return clientcmd.NewClientConfigFromBytes(kconfContents)
}
func getDynamicClient(config *rest.Config) (dynamic.Interface, error) {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}
	return client, nil
}
