package main

import (
	"encoding/base64"
	"fmt"
	"log"

	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/plugin/builtin"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"

	"sigs.k8s.io/yaml"
)

type plugin struct {
	hasher     ifc.KunstructuredHasher
	StringData map[string]string `json:"stringData,omitempty" yaml:"stringData,omitempty"`
	builtin.SecretGeneratorPlugin
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("SuperSecret")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.hasher = rf.RF().Hasher()
	p.StringData = make(map[string]string)
	err = yaml.Unmarshal(c, p)
	if err != nil {
		logger.Printf("error unmarshalling yaml, error: %v\n", err)
		return err
	}
	return p.SecretGeneratorPlugin.Config(ldr, rf, c)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	for k, v := range p.StringData {
		p.LiteralSources = append(p.LiteralSources, fmt.Sprintf("%v=%v", k, v))
	}
	return p.SecretGeneratorPlugin.Generate()
}

func (p *plugin) Transform(m resmap.ResMap) error {
	var updatedSecretName string
	var err error

	for _, res := range m.Resources() {
		if res.GetKind() == "Secret" && res.GetName() == p.Name {
			if err := p.appendDataToSecret(res, p.StringData); err != nil {
				logger.Printf("error appending data to secret with secretName: %v, error: %v\n", p.Name, err)
				return err
			}
			if !p.DisableNameSuffixHash {
				updatedSecretName, err = p.generateNameWithHash(res)
				if err != nil {
					logger.Printf("error hashing secret resource contents for secretName: %v, error: %v\n", p.Name, err)
					return err
				}
				res.SetName(updatedSecretName)
			}
			break
		}
	}

	if len(updatedSecretName) > 0 {
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

func (p *plugin) generateNameWithHash(res *resource.Resource) (string, error) {
	hash, err := p.hasher.Hash(res)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", res.GetName(), hash), nil
}

func (p *plugin) appendDataToSecret(res *resource.Resource, stringData map[string]string) error {
	for k, v := range stringData {
		pathToField := []string{"data", k}
		err := transformers.MutateField(
			res.Map(),
			pathToField,
			true,
			func(interface{}) (interface{}, error) {
				return base64.StdEncoding.EncodeToString([]byte(v)), nil
			})
		if err != nil {
			logger.Printf("error executing MutateField for secret with secretName: %v, pathToField: %v, error: %v\n", p.Name, pathToField, err)
			return err
		}
	}
	return nil
}
