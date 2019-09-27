package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	ChartName        string                 `json:"chartName,omitempty" yaml:"chartName,omitempty"`
	ChartHome        string                 `json:"chartHome,omitempty" yaml:"chartHome,omitempty"`
	ChartVersion     string                 `json:"chartVersion,omitempty" yaml:"chartVersion,omitempty"`
	ChartRepo        string                 `json:"chartRepo,omitempty" yaml:"chartRepo,omitempty"`
	ValuesFrom       string                 `json:"valuesFrom,omitempty" yaml:"valuesFrom,omitempty"`
	Values           map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	HelmHome         string                 `json:"helmHome,omitempty" yaml:"helmHome,omitempty"`
	HelmBin          string                 `json:"helmBin,omitempty" yaml:"helmBin,omitempty"`
	ReleaseName      string                 `json:"releaseName,omitempty" yaml:"releaseName,omitempty"`
	ReleaseNamespace string                 `json:"releaseNamespace,omitempty" yaml:"releaseNamespace,omitempty"`
	ExtraArgs        string                 `json:"extraArgs,omitempty" yaml:"extraArgs,omitempty"`
	ChartPatches     string                 `json:"chartPatches,omitempty" yaml:"chartPatches,omitempty"`
	ChartVersionExp  string
	ldr              ifc.Loader
	rf               *resmap.Factory
}

//nolint: go-lint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("HelmChart")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.ldr = ldr
	p.rf = rf
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {

	// make temp directory
	dir, err := ioutil.TempDir("", "tempRoot")
	if err != nil {
		logger.Printf("error creating temporaty directory: %v\n", err)
		return nil, err
	}
	dir = path.Join(dir, "../")

	if p.HelmHome == "" {
		// make home for helm stuff
		directory := fmt.Sprintf("%s/%s", dir, "dotHelm")
		p.HelmHome = directory
	}

	if len(p.ChartHome) == 0 {
		// make home for chart stuff
		directory := fmt.Sprintf("%s/%s", dir, p.ChartName)
		p.ChartHome = directory
	}

	if p.HelmBin == "" {
		p.HelmBin = "helm"
	}

	if len(p.ChartVersion) > 0 {
		p.ChartVersionExp = fmt.Sprintf("--version=%s", p.ChartVersion)
	} else {
		p.ChartVersionExp = ""
	}

	if p.ChartRepo == "" {
		p.ChartRepo = "https://kubernetes-charts.storage.googleapis.com"
	}

	if p.ReleaseName == "" {
		p.ReleaseName = "release-name"
	}

	if p.ReleaseNamespace == "" {
		p.ReleaseName = "default"
	}

	err = p.initHelm()
	if err != nil {
		logger.Printf("error executing initHelm(), error: %v\n", err)
		return nil, err
	}
	if _, err := os.Stat(p.ChartHome); os.IsNotExist(err) {
		err = p.fetchHelm()
		if err != nil {
			logger.Printf("error executing fetchHelm(), error: %v\n", err)
			return nil, err
		}
	} else if err != nil {
		logger.Printf("error executing stat on file: %v, error: %v\n", p.ChartHome, err)
	}

	err = deleteRequirements(p.ChartHome)
	if err != nil {
		logger.Printf("error executing deleteRequirements() for dir: %v, error: %v\n", p.ChartHome, err)
		return nil, err
	}

	templatedYaml, err := p.templateHelm()
	if err != nil {
		logger.Printf("error executing templateHelm(), error: %v\n", err)
		return nil, err
	}

	if len(p.ChartPatches) > 0 {
		templatedYaml, err = p.applyPatches(templatedYaml)
		if err != nil {
			logger.Printf("error executing applyPatches(), error: %v\n", err)
			return nil, err
		}
	}

	return p.rf.NewResMapFromBytes(templatedYaml)
}

func deleteRequirements(dir string) error {

	d, err := os.Open(dir)
	if err != nil {
		logger.Printf("error opening directory %v, error: %v\n", dir, err)
		return err
	}
	defer d.Close()

	files, err := d.Readdir(-1)
	if err != nil {
		logger.Printf("error listing directory %v, error: %v\n", d.Name(), err)
		return err
	}

	for _, file := range files {
		if file.Mode().IsRegular() {
			ext := filepath.Ext(file.Name())
			name := file.Name()[0 : len(file.Name())-len(ext)]
			if name == "requirements" {
				filePath := dir + "/" + file.Name()
				err := os.Remove(filePath)
				if err != nil {
					logger.Printf("error deleting the requirements file %v, error: %v\n", filePath, err)
					return err
				}
			}
		}
	}

	return nil
}

func (p *plugin) initHelm() error {
	// build helm flags
	home := fmt.Sprintf("--home=%s", p.HelmHome)
	helmCmd := exec.Command(p.HelmBin, "init", home, "--client-only")
	err := helmCmd.Run()
	if err != nil {
		logger.Printf("error executing command: %v with args: %v, error: %v\n", helmCmd.Path, helmCmd.Args, err)
		return err
	}
	return nil
}

func (p *plugin) fetchHelm() error {

	// build helm flags
	home := fmt.Sprintf("--home=%s", p.HelmHome)
	untarDir := fmt.Sprintf("--untardir=%s", p.ChartHome)
	repo := fmt.Sprintf("--repo=%s", p.ChartRepo)
	helmCmd := exec.Command("helm", "fetch", home, "--untar", untarDir, repo, p.ChartVersionExp, p.ChartName)

	var out bytes.Buffer
	helmCmd.Stdout = &out
	err := helmCmd.Run()
	if err != nil {
		logger.Printf("error executing command: %v with args: %v, error: %v\n", helmCmd.Path, helmCmd.Args, err)
		return err
	}

	fileLocation := fmt.Sprintf("%s/%s", p.ChartHome, p.ChartName)
	tempFileLocation := fileLocation + "-temp"

	err = os.Rename(fileLocation, tempFileLocation)
	if err != nil {
		logger.Printf("error renaming: %v to: %v, error: %v\n", fileLocation, tempFileLocation, err)
		return err
	}

	err = utils.CopyDir(tempFileLocation, p.ChartHome, logger)
	if err != nil {
		logger.Printf("error copying directory from: %v, to: %v, error: %v\n", tempFileLocation, p.ChartHome, err)
		return err
	}

	err = os.RemoveAll(tempFileLocation)
	if err != nil {
		logger.Printf("error removing: %v, error: %v\n", tempFileLocation, err)
		return err
	}
	return nil

}

func (p *plugin) templateHelm() ([]byte, error) {

	valuesYaml, err := yaml.Marshal(p.Values)
	if err != nil {
		logger.Printf("error marshalling values to yaml, error: %v\n", err)
		return nil, err
	}
	file, err := ioutil.TempFile("", "yaml")
	if err != nil {
		logger.Printf("error creating temp file, error: %v\n", err)
		return nil, err
	}
	_, err = file.Write(valuesYaml)
	if err != nil {
		logger.Printf("error writing yaml to file: %v, error: %v\n", file.Name(), err)
		return nil, err
	}

	// build helm flags
	home := fmt.Sprintf("--home=%s", p.HelmHome)
	values := fmt.Sprintf("--values=%s", file.Name())
	name := fmt.Sprintf("--name=%s", p.ReleaseName)
	nameSpace := fmt.Sprintf("--namespace=%s", p.ReleaseNamespace)
	helmCmd := exec.Command("helm", "template", home, values, name, nameSpace, p.ChartHome)

	if len(p.ExtraArgs) > 0 && p.ExtraArgs != "null" {
		helmCmd.Args = append(helmCmd.Args, p.ExtraArgs)
	}

	if len(p.ValuesFrom) > 0 && p.ValuesFrom != "null" {
		templatedValues := fmt.Sprintf("--values=%s", p.ValuesFrom)
		helmCmd.Args = append(helmCmd.Args, templatedValues)
	}

	var out bytes.Buffer
	var stderr bytes.Buffer
	helmCmd.Stdout = &out
	helmCmd.Stderr = &stderr
	err = helmCmd.Run()
	if err != nil {
		logger.Printf("error executing command: %v with args: %v, error: %v, stderr: %v\n", helmCmd.Path, helmCmd.Args, err, stderr.String())
		return nil, err
	}
	return out.Bytes(), nil
}

func (p *plugin) applyPatches(templatedHelm []byte) ([]byte, error) {
	// get the patches
	path := filepath.Join(p.ChartHome + "/" + p.ChartPatches + "/kustomization.yaml")
	origYamlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Printf("error reading file: %v, error: %v\n", path, err)
		return nil, err
	}

	var originalYamlMap map[string]interface{}

	if err := yaml.Unmarshal(origYamlBytes, &originalYamlMap); err != nil {
		logger.Printf("error unmarshalling kustomization yaml from file: %v, error: %v\n", path, err)
	}

	// helmoutput file for kustomize build
	helpOutputPath := p.ChartHome + "/" + p.ChartPatches + "/helmoutput.yaml"
	f, err := os.Create(helpOutputPath)
	if err != nil {
		logger.Printf("error creating helm output file: %v, error: %v\n", helpOutputPath, err)
		return nil, err
	}

	_, err = f.Write(templatedHelm)
	if err != nil {
		logger.Printf("error writing to helm output file: %v, error: %v\n", helpOutputPath, err)
		return nil, err
	}

	kustomizeYaml, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Printf("error reading file: %v, error: %v\n", path, err)
		return nil, err
	}

	var kustomizeYamlMap map[string]interface{}
	if err := yaml.Unmarshal(kustomizeYaml, &kustomizeYamlMap); err != nil {
		logger.Printf("error unmarshalling kustomization yaml from file: %v, error: %v\n", path, err)
	}

	delete(kustomizeYamlMap, "resources")

	kustomizeYamlMap["resources"] = []string{"helmoutput.yaml"}

	yamlM, err := yaml.Marshal(kustomizeYamlMap)
	if err != nil {
		logger.Printf("error marshalling kustomization yaml map, error: %v\n", err)
		return nil, err
	}

	if err := ioutil.WriteFile(path, yamlM, 0644); err != nil {
		logger.Printf("error writing kustomization yaml to file: %v, error: %v\n", path, err)
	}

	// kustomize build
	templatedHelm, err = p.buildPatches()
	if err != nil {
		logger.Printf("error executing buildPatches(), error: %v\n", err)
		return nil, err
	}

	return templatedHelm, nil
}

func (p *plugin) buildPatches() ([]byte, error) {
	path := filepath.Join(p.ChartHome + "/" + p.ChartPatches)
	kustomizeCmd := exec.Command("kustomize", "build", path)

	var out bytes.Buffer
	kustomizeCmd.Stdout = &out

	err := kustomizeCmd.Run()
	if err != nil {
		logger.Printf("error executing command: %v with args: %v, error: %v\n", kustomizeCmd.Path, kustomizeCmd.Args, err)
		return nil, err
	}
	return out.Bytes(), nil
}
