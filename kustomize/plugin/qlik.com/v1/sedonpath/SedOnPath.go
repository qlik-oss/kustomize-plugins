package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	Path      string `json:"path,omitempty" yaml:"path,omitempty"`
	Regex     string `json:"regex,omitempty" yaml:"regex,omitempty"`
	fieldSpec config.FieldSpec
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("SedOnPath")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.Path = ""
	p.Regex = ""
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
			true,
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
			zNewValue, err := p.executeSedOnValue(zValue)
			if err != nil {
				return nil, err
			}
			zNewArray = append(zNewArray, zNewValue)
		}
		return zNewArray, nil
	}

	return nil, fmt.Errorf("%#v is expected to be a string or []string", in)
}

func (p *plugin) runCommand(cmd *exec.Cmd, env []string, dir *string) ([]byte, error) {
	cmd.Env = env
	if dir != nil {
		cmd.Dir = *dir
	}
	return cmd.Output()
}

func (p *plugin) executeSed(zString string) (string, error) {
	cmd := exec.Command("sed", "-e", p.Regex)
	cmd.Stdin = bytes.NewBuffer([]byte(zString))
	output, err := cmd.Output()
	return string(output), err
}