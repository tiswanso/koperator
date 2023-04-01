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

package helm

import (
	"fmt"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
)

func InstallChart(releaseName, namespace, chartTarFile string, getter genericclioptions.RESTClientGetter) error {
	return InstallChartWithValues(releaseName, namespace, chartTarFile, getter, nil)
}
func InstallChartWithValues(releaseName, namespace, chartTarFile string, getter genericclioptions.RESTClientGetter, values map[string]interface{}) error {
	chart, err := loader.Load(chartTarFile)
	if err != nil {
		return err
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(getter, namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		fmt.Sprintf(format, v)
	}); err != nil {
		return err
	}

	hinstall := action.NewInstall(actionConfig)
	hinstall.Namespace = namespace
	hinstall.ReleaseName = releaseName
	rel, err := hinstall.Run(chart, values)
	if err != nil {
		return err
	}
	fmt.Println("Successfully installed release: ", rel.Name)
	return nil
}

func UninstallChart(releaseName, namespace, chartTarFile string, getter genericclioptions.RESTClientGetter) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(getter, namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		fmt.Sprintf(format, v)
	}); err != nil {
		return err
	}

	uninstall := action.NewUninstall(actionConfig)
	rel, err := uninstall.Run(releaseName)
	if err != nil {
		return err
	}
	fmt.Printf("Successfully uninstalled release %s: %s", releaseName, rel.Info)
	return nil
}
