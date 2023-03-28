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
