package main

import (
	"log"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/yaml"
)

// A secret generator example that gets data
// from a database (simulated by a hardcoded map).
type plugin struct {
	rf               *resmap.Factory
	ldr              ifc.Loader
	types.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// List of keys to use in database lookups
	types.GeneratorOptions
	types.ConfigMapArgs
	Data map[string]string `json:"data,omitempty" protobuf:"bytes,2,rep,name=data"`
}

//nolint: golint
//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("ConfigMapGeneratorPlus")
}

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, config []byte) (err error) {
	p.GeneratorOptions = types.GeneratorOptions{}
	p.ConfigMapArgs = types.ConfigMapArgs{}
	err = yaml.Unmarshal(config, p)
	if p.ConfigMapArgs.Name == "" {
		p.ConfigMapArgs.Name = p.Name
	}
	if p.ConfigMapArgs.Namespace == "" {
		p.ConfigMapArgs.Namespace = p.Namespace
	}
	p.ldr = ldr
	p.rf = rf
	return
}

// The plan here is to convert the plugin's input
// into the format used by the builtin secret generator plugin.
func (p *plugin) Generate() (resmap.ResMap, error) {
	for v := range p.Data {
		if k, ok := p.Data[v]; ok {
			p.LiteralSources = append(
				p.LiteralSources, v+"="+k)
		}
	}
	return p.rf.FromConfigMapArgs(p.ldr, &p.GeneratorOptions, p.ConfigMapArgs)
}
