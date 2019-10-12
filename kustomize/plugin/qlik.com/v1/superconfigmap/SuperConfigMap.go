package main

import (
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
	Data map[string]string `json:"data,omitempty" yaml:"data,omitempty"`
	builtin.ConfigMapGeneratorPlugin
	supermapplugin.Base
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("SuperConfigMap")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.Base = supermapplugin.NewBase(rf, p)
	p.Data = make(map[string]string)
	err = yaml.Unmarshal(c, p)
	if err != nil {
		logger.Printf("error unmarshalling yaml, error: %v\n", err)
		return err
	}
	return p.ConfigMapGeneratorPlugin.Config(ldr, rf, c)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	for k, v := range p.Data {
		p.LiteralSources = append(p.LiteralSources, fmt.Sprintf("%v=%v", k, v))
	}
	return p.ConfigMapGeneratorPlugin.Generate()
}

func (p *plugin) Transform(m resmap.ResMap) error {
	return p.Base.Transform(m)
}

func (p *plugin) GetLogger() *log.Logger {
	return logger
}

func (p *plugin) GetName() string {
	return p.ConfigMapGeneratorPlugin.Name
}

func (p *plugin) GetType() string {
	return "ConfigMap"
}

func (p *plugin) GetConfigData() map[string]string {
	return p.Data
}

func (p *plugin) ShouldBase64EncodeConfigData() bool {
	return false
}

func (p *plugin) GetDisableNameSuffixHash() bool {
	return p.ConfigMapGeneratorPlugin.DisableNameSuffixHash
}
