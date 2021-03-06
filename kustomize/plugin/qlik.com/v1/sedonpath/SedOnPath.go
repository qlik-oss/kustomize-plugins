package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	Path      string `json:"path,omitempty" yaml:"path,omitempty"`
	Regex     []string `json:"regex,omitempty" yaml:"regex,omitempty"`
	fieldSpec config.FieldSpec
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("SedOnPath")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.Path = ""
	p.Regex = make([]string, 0)
	err = yaml.Unmarshal(c, p)
	if err != nil {
		logger.Printf("error unmarshalling config from yaml, error: %v\n", err)
		return err
	}
	p.fieldSpec = config.FieldSpec{Path: p.Path}
	return nil
}

func (p *plugin) Transform(m resmap.ResMap) error {
	for _, r := range m.Resources() {
		err := transformers.MutateField(
			r.Map(),
			p.fieldSpec.PathSlice(),
			false,
			p.executeSedOnValue)
		if err != nil {
			logger.Printf("error executing transformers.MutateField(), error: %v\n", err)
			return err
		}
	}
	return nil
}

func (p *plugin) executeSedOnValue(in interface{}) (interface{}, error) {
	zString, ok := in.(string)
	if ok {
		return p.executeSed(zString)
	}

	zArray, ok := in.([]interface{})
	if ok {
		zNewArray := zArray[:0]
		for _, zValue := range zArray {
			zString, ok := zValue.(string)
			if !ok {
				return nil, fmt.Errorf("%#v is expected to be a string or []string", in)
			}
			zNewValue, err := p.executeSed(zString)
			if err != nil {
				return nil, err
			}
			zNewArray = append(zNewArray, zNewValue)
		}
		return zNewArray, nil
	}

	return nil, fmt.Errorf("%#v is expected to be a string or []string", in)
}

func (p *plugin) executeSed(zString string) (string, error) {
	for _, regex := range p.Regex {
		cmd := exec.Command("sed", "-e", regex)
		cmd.Stdin = bytes.NewBuffer([]byte(zString))
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		zString = string(output)
	}
	return zString, nil
}
