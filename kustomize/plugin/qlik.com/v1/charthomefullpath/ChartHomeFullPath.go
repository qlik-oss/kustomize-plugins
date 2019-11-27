package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"sigs.k8s.io/kustomize/v3/pkg/transformers"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/ifc"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
)

type plugin struct {
	ChartHome  string            `json:"chartHome,omitempty" yaml:"chartHome,omitempty"`
	FieldSpecs []types.FieldSpec `json:"fieldSpecs,omitempty" yaml:"fieldSpecs,omitempty"`
	Root       string
	ChartName  string
	Kind       string
}

//nolint: golint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("ChartHomeFullPath")
}

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.Root = h.Loader().Root()
	return yaml.Unmarshal(c, p)
}

func (p *plugin) mutate(in interface{}) (interface{}, error) {
	dir, err := ioutil.TempDir("", "temp")
	if err != nil {
		logger.Printf("error creating temporaty directory: %v\n", err)
		return nil, err
	}
	directory := fmt.Sprintf("%s/%s", dir, p.ChartName)
	err = os.Mkdir(directory, 0777)
	if err != nil {
		logger.Printf("error creating directory: %v, error: %v\n", directory, err)
		return nil, err
	}
	if p.Kind == "HelmChart" {
		err := utils.CopyDir(p.ChartHome, directory, logger)
		if err != nil {
			logger.Printf("error copying directory from: %v, to: %v, error: %v\n", p.ChartHome, directory, err)
			return nil, err
		}
	}
	return directory, nil
}

func (p *plugin) Transform(m resmap.ResMap) error {
	//join the root(root of kustomize file) + location to chartHome
	p.ChartHome = path.Join(p.Root, p.ChartHome)

	for _, r := range m.Resources() {
		p.ChartName = GetFieldValue(r, "chartName")
		p.Kind = GetFieldValue(r, "kind")
		pathToField := []string{"chartHome"}
		err := transformers.MutateField(
			r.Map(),
			pathToField,
			true,
			p.mutate)
		if err != nil {
			logger.Printf("error executing MutateField for chart: %v, pathToField: %v, error: %v\n", p.ChartName, pathToField, err)
			return err
		}
	}
	return nil
}

func GetFieldValue(obj ifc.Kunstructured, fieldName string) string {
	v, err := obj.GetString(fieldName)
	if err != nil {
		logger.Printf("error extracting fieldName: %v (will return empty string), error: %v\n", fieldName, err)
		return ""
	}
	return v
}
