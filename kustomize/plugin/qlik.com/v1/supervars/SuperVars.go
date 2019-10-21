package main

import (
	"fmt"
	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"log"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	Vars           []types.Var `json:"vars,omitempty" yaml:"vars,omitempty"`
	Configurations []string    `json:"configurations,omitempty" yaml:"configurations,omitempty"`
	tConfig        *config.TransformerConfig
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("SuperVars")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.Vars = make([]types.Var, 0)
	p.Configurations = make([]string, 0)

	err = yaml.Unmarshal(c, p)
	if err != nil {
		logger.Printf("error unmarshelling plugin config yaml, error: %v\n", err)
		return err
	}

	p.tConfig = &config.TransformerConfig{}
	tCustomConfig, err := config.MakeTransformerConfig(ldr, p.Configurations)
	if err != nil {
		logger.Printf("error making transformer config, error: %v\n", err)
		return err
	}
	p.tConfig, err = p.tConfig.Merge(tCustomConfig)
	if err != nil {
		logger.Printf("error merging transformer config, error: %v\n", err)
		return err
	}

	return nil
}

func (p *plugin) Transform(m resmap.ResMap) error {
	varReplacementMap, err := p.resolveVarValues(m)
	if err != nil {
		return err
	}
	refVarTransformer := transformers.NewRefVarTransformer(varReplacementMap, p.tConfig.VarReference)
	return refVarTransformer.Transform(m)
}

func (p *plugin) resolveVarValues(m resmap.ResMap) (map[string]interface{}, error) {
	varValues := make(map[string]interface{})
	foundVars := make(map[string]bool)
	for _, zVar := range p.Vars {
		foundVars[zVar.Name] = false
	}

	for _, res := range m.Resources() {
		for _, zVar := range p.Vars {
			if !foundVars[zVar.Name] && res.OrgId().IsSelected(&zVar.ObjRef.Gvk) && res.GetName() == zVar.ObjRef.Name {
				val, err := res.GetFieldValue(zVar.FieldRef.FieldPath)
				if err != nil {
					logger.Printf("error getting field value for var: '%v', error: %v\n", zVar.Name, err)
					return nil, err
				}
				varValues[zVar.Name] = val
				foundVars[zVar.Name] = true
				break
			}
		}
	}
	for _, zVar := range p.Vars {
		if !foundVars[zVar.Name] {
			return nil, fmt.Errorf("var: '%v' cannot be mapped to a field in the set of known resources", zVar.Name)
		}
	}
	return varValues, nil
}
