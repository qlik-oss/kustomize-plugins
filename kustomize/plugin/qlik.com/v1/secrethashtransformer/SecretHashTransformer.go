package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"sigs.k8s.io/kustomize/v3/pkg/resource"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"

	"sigs.k8s.io/yaml"
)

type plugin struct {
	hasher     ifc.KunstructuredHasher
	SecretName string            `json:"secretName,omitempty" yaml:"secretName,omitempty"`
	Append     map[string]string `json:"append,omitempty" yaml:"append,omitempty"`
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("SecretHashTransformer")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.hasher = rf.RF().Hasher()
	p.SecretName = ""
	p.Append = make(map[string]string)
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	newSecretName := ""
	for _, res := range m.Resources() {
		if res.GetKind() == "Secret" && res.GetName() == p.SecretName {
			if err := p.appendDataToSecret(res, p.Append); err != nil {
				logger.Printf("error appending data to secret with secretName: %v, error: %v\n", p.SecretName, err)
			  return err
			}
			hash, err := p.hasher.Hash(res)
			if err != nil {
				logger.Printf("error hashing secret resource contents for secretName: %v, error: %v\n", p.SecretName, err)
				return err
			}
			newSecretName = fmt.Sprintf("%s-%s", res.GetName(), hash)
			res.SetName(newSecretName)
			break
		}
	}

	if len(newSecretName) > 0 {
		defaultTransformerConfig := config.MakeDefaultConfig()
		nameReferenceTransformer := transformers.NewNameReferenceTransformer(defaultTransformerConfig.NameReference)
		err := nameReferenceTransformer.Transform(m)
		if err != nil {
			logger.Printf("error executing nameReferenceTransformer.Transform(): %v\n", err)
			return err
		}
	}

	return nil
}

func (p *plugin) appendDataToSecret(res *resource.Resource, dataMap map[string]string) error {
	for k, v := range dataMap {
		pathToField := []string{"data", k}
		err := transformers.MutateField(
			res.Map(),
			pathToField,
			true,
			func(interface{}) (interface{}, error) {
				return base64.StdEncoding.EncodeToString([]byte(v)), nil
			})
		if err != nil {
			logger.Printf("error executing MutateField for secret with secretName: %v, pathToField: %v, error: %v\n", p.SecretName, pathToField, err)
			return err
		}
	}
	return nil
}
