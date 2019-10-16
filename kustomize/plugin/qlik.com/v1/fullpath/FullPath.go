package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	RootDir    string
	FieldSpecs []config.FieldSpec `json:"fieldSpecs,omitempty" yaml:"fieldSpecs,omitempty"`
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("FullPath")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.RootDir = ldr.Root()
	p.FieldSpecs = make([]config.FieldSpec, 0)

	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	for _, r := range m.Resources() {
		id := r.OrgId()
		for _, fieldSpec := range p.FieldSpecs {
			if !id.IsSelected(&fieldSpec.Gvk) {
				continue
			}

			err := transformers.MutateField(
				r.Map(),
				fieldSpec.PathSlice(),
				fieldSpec.CreateIfNotPresent,
				p.computePath)
			if err != nil {
				logger.Printf("error executing transformers.MutateField(), error: %v\n", err)
				return err
			}
		}
	}
	return nil
}

func (p *plugin) computePath(in interface{}) (interface{}, error) {
	relativePath, ok := in.(string)
	if !ok {
		return nil, fmt.Errorf("%#v is expected to be %T", in, relativePath)
	}
	return filepath.Join(p.RootDir, relativePath), nil
}
