package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	ChartHome  string             `json:"chartHome,omitempty" yaml:"chartHome,omitempty"`
	FieldSpecs []config.FieldSpec `json:"fieldSpecs,omitempty" yaml:"fieldSpecs,omitempty"`
	Root       string
	ChartName  string
	Kind       string
}

//nolint: golint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.Root = ldr.Root()
	return yaml.Unmarshal(c, p)
}

func (p *plugin) mutate(in interface{}) (interface{}, error) {
	dir, err := ioutil.TempDir("", "temp")
	if err != nil {
		return nil, err
	}
	directory := fmt.Sprintf("%s/%s", dir, p.ChartName)
	err = os.Mkdir(directory, 0777)
	if err != nil {
		return nil, err
	}
	if p.Kind == "HelmChart" {
		err := copyDir(p.ChartHome, directory)
		if err != nil {
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
		err := transformers.MutateField(
			r.Map(),
			[]string{"chartHome"},
			true,
			p.mutate)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetFieldValue(obj ifc.Kunstructured, fieldName string) string {
	v, err := obj.GetString(fieldName)
	if err != nil {
		return ""
	}
	return v
}

// copy source file to destination location
func copyFile(source string, dest string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
			return err
		}
	}
	return nil
}

//copy source directory to destination
func copyDir(source string, dest string) error {
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}
	sourceDirectory, _ := os.Open(source)
	// read everything within source directory
	objects, _ := sourceDirectory.Readdir(-1)

	// go through all files/directories
	for _, obj := range objects {

		sourceFileName := source + "/" + obj.Name()

		destinationFileName := dest + "/" + obj.Name()

		if obj.IsDir() {
			err := copyDir(sourceFileName, destinationFileName)
			if err != nil {
				return err
			}
		} else {
			err := copyFile(sourceFileName, destinationFileName)
			if err != nil {
				return err
			}
		}

	}
	return nil
}
