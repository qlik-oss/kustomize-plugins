package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/transform"
	"sigs.k8s.io/kustomize/api/types"
	v1 "sigs.k8s.io/kustomize/pseudo/k8s/api/core/v1"
	"sigs.k8s.io/kustomize/pseudo/k8s/apimachinery/pkg/util/yaml"
)

type EnvVarType struct {
	v1.EnvVar
	Delete bool `json:"delete,omitempty" yaml:"delete,omitempty"`
}

type plugin struct {
	Enabled   bool            `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Target    *types.Selector `json:"target,omitempty" yaml:"target,omitempty"`
	Path      string          `json:"path,omitempty" yaml:"path,omitempty"`
	EnvVars   []EnvVarType    `json:"env,omitempty" yaml:"env,omitempty"`
	fieldSpec types.FieldSpec
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("EnvUpsert")
}

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.Enabled = false
	p.Target = nil
	p.Path = ""
	p.EnvVars = make([]EnvVarType, 0)

	if jsonBytes, err := yaml.ToJSON(c); err != nil {
		logger.Printf("error converting yaml to json, error: %v\n", err)
		return err
	} else if err := json.Unmarshal(jsonBytes, &p); err != nil {
		logger.Printf("error unmarshalling config from json, error: %v\n", err)
		return err
	}
	if p.Target == nil {
		return fmt.Errorf("must specify a target in the config for the environment variables upsert")
	}
	for _, envVar := range p.EnvVars {
		if envVar.Name == "" {
			err := fmt.Errorf("env var config has no name: %v", envVar)
			logger.Printf("config error: %v\n", err)
			return err
		}
		if envVar.Value == "" && envVar.ValueFrom == nil && !envVar.Delete {
			err := fmt.Errorf("env var config has no value or valueFrom: %v", envVar)
			logger.Printf("config error: %v\n", err)
			return err
		}
	}
	p.fieldSpec = types.FieldSpec{Path: p.Path}
	return nil
}

func (p *plugin) Transform(m resmap.ResMap) error {
	if p.Enabled {
		resources, err := m.Select(*p.Target)

		if err != nil {
			logger.Printf("error selecting resources based on the target selector, error: %v\n", err)
			return err
		}
		for _, r := range resources {
			err := transform.MutateField(
				r.Map(),
				p.fieldSpec.PathSlice(),
				false,
				p.upsertEnvironmentVariables)
			if err != nil {
				logger.Printf("error executing transformers.MutateField(), error: %v\n", err)
				return err
			}
		}
	}
	return nil
}

func (p *plugin) upsertEnvironmentVariables(in interface{}) (interface{}, error) {
	presentEnvVars, ok := in.([]interface{})
	if ok {
		for _, envVar := range p.EnvVars {
			foundMatching := false
			for i := 0; i < len(presentEnvVars); i++ {
				presentEnvVar, ok := presentEnvVars[i].(map[string]interface{})
				if ok {
					name, ok := presentEnvVar["name"].(string)
					if ok {
						if name == envVar.Name {
							foundMatching = true
							if envVar.Delete {
								//delete:
								presentEnvVars = append(presentEnvVars[:i], presentEnvVars[i+1:]...)
								i--
							} else {
								//update:
								if err := p.setEnvVar(presentEnvVar, envVar); err != nil {
									logger.Printf("error executing p.setEnvVar(), error: %v\n", err)
									return nil, err
								}
							}
							break
						}
					}
				}
			}
			if !foundMatching && !envVar.Delete {
				//insert:
				newEnvVar := map[string]interface{}{
					"name": envVar.Name,
				}
				if err := p.setEnvVar(newEnvVar, envVar); err != nil {
					logger.Printf("error executing p.setEnvVar(), error: %v\n", err)
					return nil, err
				}
				presentEnvVars = append(presentEnvVars, newEnvVar)
			}
		}
		return presentEnvVars, nil
	}
	return in, nil
}

func (p *plugin) setEnvVar(setEnvVar map[string]interface{}, fromEnvVar EnvVarType) error {
	if fromEnvVar.ValueFrom != nil {
		var valueFrom map[string]interface{}
		if bytes, err := json.Marshal(fromEnvVar.ValueFrom); err != nil {
			return err
		} else if err := json.Unmarshal(bytes, &valueFrom); err != nil {
			return err
		}
		setEnvVar["valueFrom"] = valueFrom
	} else {
		setEnvVar["value"] = fromEnvVar.Value
	}
	return nil
}
