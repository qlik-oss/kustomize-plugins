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
	StringData            map[string]string `json:"stringData,omitempty" yaml:"stringData,omitempty"`
	AssumeTargetWillExist bool              `json:"assumeTargetWillExist,omitempty" yaml:"assumeTargetWillExist,omitempty"`
	Prefix				  string            `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	builtin.SecretGeneratorPlugin
	supermapplugin.Base
}

var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("SuperSecret")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.Base = supermapplugin.Base{
		Hasher:    rf.RF().Hasher(),
		Decorator: p,
	}
	p.StringData = make(map[string]string)
	p.AssumeTargetWillExist = true
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
	return p.Base.Transform(m)
}

func (p *plugin) GetLogger() *log.Logger {
	return logger
}

func (p *plugin) GetName() string {
	return p.Name
}

func (p *plugin) GetType() string {
	return "Secret"
}

func (p *plugin) GetConfigData() map[string]string {
	return p.StringData
}

func (p *plugin) ShouldBase64EncodeConfigData() bool {
	return true
}

func (p *plugin) GetAssumeTargetWillExist() bool {
	return p.AssumeTargetWillExist
}

func (p *plugin) GetDisableNameSuffixHash() bool {
	return p.DisableNameSuffixHash
}

func (p *plugin) GetPrefix() string {
	return p.Prefix
}