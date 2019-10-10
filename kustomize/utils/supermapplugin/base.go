package supermapplugin

import (
	"encoding/base64"
	"fmt"
	"log"

	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
)

type IDecorator interface {
	GetLogger() *log.Logger
	GetName() string
	GetType() string
	GetConfigData() map[string]string
	ShouldBase64EncodeConfigData() bool
	GetAssumeTargetWillExist() bool
	GetDisableNameSuffixHash() bool
	Generate() (resmap.ResMap, error)
	GetPrefix() string
}

type Base struct {
	Hasher    ifc.KunstructuredHasher
	Decorator IDecorator
}

func (b *Base) Transform(m resmap.ResMap) error {
	resource := b.find(b.Decorator.GetName(), b.Decorator.GetType(), m)
	if resource != nil {
		return b.executeBasicTransform(resource, m)
	} else if b.Decorator.GetAssumeTargetWillExist() && !b.Decorator.GetDisableNameSuffixHash() {
		return b.executeAssumeWillExistTransform(m)
	} else {
		b.Decorator.GetLogger().Printf("NOT executing anything because resource: %v is NOT in the input stream and AssumeTargetWillExist: %v, disableNameSuffixHash: %v\n", b.Decorator.GetName(), b.Decorator.GetAssumeTargetWillExist(), b.Decorator.GetDisableNameSuffixHash())
	}
	return nil
}

func (b *Base) executeAssumeWillExistTransform(m resmap.ResMap) error {
	b.Decorator.GetLogger().Printf("executeAssumeWillExistTransform() for imaginary resource: %v\n", b.Decorator.GetName())

	generateResourceMap, err := b.Decorator.Generate()
	if err != nil {
		b.Decorator.GetLogger().Printf("error generating temp resource: %v, error: %v\n", b.Decorator.GetName(), err)
		return err
	}
	tempResource := b.find(b.Decorator.GetName(), b.Decorator.GetType(), generateResourceMap)
	if tempResource == nil {
		err := fmt.Errorf("error locating generated temp resource: %v", b.Decorator.GetName())
		b.Decorator.GetLogger().Printf("%v\n", err)
		return err
	}
	err = m.Append(tempResource)
	if err != nil {
		b.Decorator.GetLogger().Printf("error appending temp resource: %v to the resource map, error: %v\n", b.Decorator.GetName(), err)
		return err
	}
	updatedName, err := b.generateNameWithHash(tempResource)
	if err != nil {
		b.Decorator.GetLogger().Printf("error hashing resource: %v, error: %v\n", b.Decorator.GetName(), err)
		return err
	}
	prefix := b.Decorator.GetPrefix()
	if len(prefix) > 0 {
		updatedName = fmt.Sprintf("%s%s", prefix, updatedName)
	}
	tempResource.SetName(updatedName)
	err = b.executeNameReferencesTransformer(m)
	if err != nil {
		b.Decorator.GetLogger().Printf("error executing nameReferenceTransformer.Transform(): %v\n", err)
		return err
	}
	err = m.Remove(tempResource.CurId())
	if err != nil {
		b.Decorator.GetLogger().Printf("error removing temp resource: %v from the resource map, error: %v\n", b.Decorator.GetName(), err)
		return err
	}
	return nil
}

func (b *Base) executeBasicTransform(resource *resource.Resource, m resmap.ResMap) error {
	b.Decorator.GetLogger().Printf("executeBasicTransform() for resource: %v...\n", resource)

	var updatedName string
	var err error
	if err := b.appendData(resource, b.Decorator.GetConfigData()); err != nil {
		b.Decorator.GetLogger().Printf("error appending data to resource: %v, error: %v\n", b.Decorator.GetName(), err)
		return err
	}
	if !b.Decorator.GetDisableNameSuffixHash() {
		updatedName, err = b.generateNameWithHash(resource)
		if err != nil {
			b.Decorator.GetLogger().Printf("error hashing resource: %v, error: %v\n", b.Decorator.GetName(), err)
			return err
		}
		resource.SetName(updatedName)
	}
	if len(updatedName) > 0 {
		err := b.executeNameReferencesTransformer(m)
		if err != nil {
			b.Decorator.GetLogger().Printf("error executing nameReferenceTransformer.Transform(): %v\n", err)
			return err
		}
	}
	return nil
}

func (b *Base) executeNameReferencesTransformer(m resmap.ResMap) error {
	defaultTransformerConfig := config.MakeDefaultConfig()
	nameReferenceTransformer := transformers.NewNameReferenceTransformer(defaultTransformerConfig.NameReference)
	return nameReferenceTransformer.Transform(m)
}

func (b *Base) find(name string, resourceType string, m resmap.ResMap) *resource.Resource {
	for _, res := range m.Resources() {
		if res.GetKind() == resourceType && res.GetOriginalName() == b.Decorator.GetName() {
			return res
		}
	}
	return nil
}

func (b *Base) generateNameWithHash(res *resource.Resource) (string, error) {
	hash, err := b.Hasher.Hash(res)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", res.GetName(), hash), nil
}

func (b *Base) appendData(res *resource.Resource, data map[string]string) error {
	for k, v := range data {
		pathToField := []string{"data", k}
		err := transformers.MutateField(
			res.Map(),
			pathToField,
			true,
			func(interface{}) (interface{}, error) {
				var val string
				if b.Decorator.ShouldBase64EncodeConfigData() {
					val = base64.StdEncoding.EncodeToString([]byte(v))
				} else {
					val = v
				}
				return val, nil
			})
		if err != nil {
			b.Decorator.GetLogger().Printf("error executing MutateField for resource: %v, pathToField: %v, error: %v\n", b.Decorator.GetName(), pathToField, err)
			return err
		}
	}
	return nil
}
