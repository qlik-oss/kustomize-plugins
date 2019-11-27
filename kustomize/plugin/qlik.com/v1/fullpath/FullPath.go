package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/transform"
	"sigs.k8s.io/kustomize/api/types"
)

type plugin struct {
	RootDir    string
	FieldSpecs []types.FieldSpec `json:"fieldSpecs,omitempty" yaml:"fieldSpecs,omitempty"`
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("FullPath")
}

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.RootDir = h.Loader().Root()
	p.FieldSpecs = make([]types.FieldSpec, 0)

	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	for _, r := range m.Resources() {
		id := r.OrgId()
		for _, fieldSpec := range p.FieldSpecs {
			if !id.IsSelected(&fieldSpec.Gvk) {
				continue
			}

			err := transform.MutateField(
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
