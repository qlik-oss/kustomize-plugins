package main

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"github.com/qlik-oss/kustomize-plugins/kustomize/utils/supermapplugin"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/plugin/builtin"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	StringData          map[string]string `json:"stringData,omitempty" yaml:"stringData,omitempty"`
	Data                map[string]string `json:"data,omitempty" yaml:"data,omitempty"`
	aggregateConfigData map[string]string
	builtin.SecretGeneratorPlugin
	supermapplugin.Base
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("SuperSecret")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.Base = supermapplugin.NewBase(rf, p)
	p.Data = make(map[string]string)
	p.StringData = make(map[string]string)
	err = yaml.Unmarshal(c, p)
	if err != nil {
		logger.Printf("error unmarshalling yaml, error: %v\n", err)
		return err
	}
	p.aggregateConfigData, err = p.getAggregateConfigData()
	if err != nil {
		logger.Printf("error accumulating config data: %v\n", err)
		return err
	}
	return p.SecretGeneratorPlugin.Config(ldr, rf, c)
}

func (p *plugin) getAggregateConfigData() (map[string]string, error) {
	aggregateConfigData := make(map[string]string)
	for k, v := range p.StringData {
		aggregateConfigData[k] = v
	}
	for k, v := range p.Data {
		decodedValue, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			logger.Printf("error base64 decoding value: %v for key: %v, error: %v\n", v, k, err)
			return nil, err
		}
		aggregateConfigData[k] = string(decodedValue)
	}
	return aggregateConfigData, nil
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	for k, v := range p.aggregateConfigData {
		p.LiteralSources = append(p.LiteralSources, fmt.Sprintf("%v=%v", k, v))
	}
	return p.SecretGeneratorPlugin.Generate()
}

func (p *plugin) Transform(m resmap.ResMap) error {
	return p.Base.Transform(m)
}

func (p *plugin) GetLogger() *log.Logger {
	return logger
}

func (p *plugin) GetName() string {
	return p.SecretGeneratorPlugin.Name
}

func (p *plugin) GetType() string {
	return "Secret"
}

func (p *plugin) GetConfigData() map[string]string {
	return p.aggregateConfigData
}

func (p *plugin) ShouldBase64EncodeConfigData() bool {
	return true
}

func (p *plugin) GetDisableNameSuffixHash() bool {
	return p.SecretGeneratorPlugin.DisableNameSuffixHash
}
