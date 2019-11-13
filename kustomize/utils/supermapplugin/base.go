package supermapplugin

import (
	"encoding/base64"
	"fmt"
	"log"

	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/k8sdeps/validator"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/loader"
	"sigs.k8s.io/kustomize/v3/pkg/plugins"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/target"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/kustomize/v3/pkg/types"
)

type IDecorator interface {
	GetLogger() *log.Logger
	GetName() string
	GetType() string
	GetConfigData() map[string]string
	ShouldBase64EncodeConfigData() bool
	GetDisableNameSuffixHash() bool
	Generate() (resmap.ResMap, error)
}

type Base struct {
	AssumeTargetWillExist           bool   `json:"assumeTargetWillExist,omitempty" yaml:"assumeTargetWillExist,omitempty"`
	AssumeTargetInKustomizationPath string `json:"assumeTargetInKustomizationPath,omitempty" yaml:"assumeTargetInKustomizationPath,omitempty"`
	Prefix                          string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Rf                              *resmap.Factory
	Hasher                          ifc.KunstructuredHasher
	Decorator                       IDecorator
	Configurations                  []string `json:"configurations,omitempty" yaml:"configurations,omitempty"`
	tConfig                         *config.TransformerConfig
}

func NewBase(rf *resmap.Factory, decorator IDecorator) Base {
	return Base{
		AssumeTargetWillExist:           true,
		AssumeTargetInKustomizationPath: "",
		Prefix:                          "",
		Rf:                              rf,
		Decorator:                       decorator,
		Hasher:                          rf.RF().Hasher(),
		Configurations:                  make([]string, 0),
		tConfig:                         nil,
	}
}

func (b *Base) SetupTransformerConfig(ldr ifc.Loader) error {
	b.tConfig = &config.TransformerConfig{}
	tCustomConfig, err := config.MakeTransformerConfig(ldr, b.Configurations)
	if err != nil {
		b.Decorator.GetLogger().Printf("error making transformer config, error: %v\n", err)
		return err
	}
	b.tConfig, err = b.tConfig.Merge(tCustomConfig)
	if err != nil {
		b.Decorator.GetLogger().Printf("error merging transformer config, error: %v\n", err)
		return err
	}
	return nil
}

func (b *Base) Transform(m resmap.ResMap) error {
	resource := b.find(b.Decorator.GetName(), b.Decorator.GetType(), m)
	if resource != nil {
		return b.executeBasicTransform(resource, m)
	} else if b.AssumeTargetWillExist && !b.Decorator.GetDisableNameSuffixHash() {
		return b.executeAssumeWillExistTransform(m)
	} else {
		b.Decorator.GetLogger().Printf("NOT executing anything because resource: %v is NOT in the input stream and AssumeTargetWillExist: %v, disableNameSuffixHash: %v\n", b.Decorator.GetName(), b.AssumeTargetWillExist, b.Decorator.GetDisableNameSuffixHash())
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

	if len(b.AssumeTargetInKustomizationPath) > 0 {
		b.Decorator.GetLogger().Printf("augmenting temp resource: %v based on kustomization path: %v\n", b.Decorator.GetName(), b.AssumeTargetInKustomizationPath)
		err = b.augmentBasedOnKustomizationPath(tempResource)
		if err != nil {
			b.Decorator.GetLogger().Printf("error augmenting temp resource: %v based on kustomization path: %v, error: %v\n", b.Decorator.GetName(), b.AssumeTargetInKustomizationPath, err)
			return err
		}
	}

	err = m.Append(tempResource)
	if err != nil {
		b.Decorator.GetLogger().Printf("error appending temp resource: %v to the resource map, error: %v\n", b.Decorator.GetName(), err)
		return err
	}

	resourceName := b.Decorator.GetName()
	if len(b.Prefix) > 0 {
		resourceName = fmt.Sprintf("%s%s", b.Prefix, resourceName)
	}
	tempResource.SetName(resourceName)

	nameWithHash, err := b.generateNameWithHash(tempResource)
	if err != nil {
		b.Decorator.GetLogger().Printf("error hashing resource: %v, error: %v\n", resourceName, err)
		return err
	}
	tempResource.SetName(nameWithHash)

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

func (b *Base) augmentBasedOnKustomizationPath(tempResource *resource.Resource) error {
	resMapFromKustomizationPath, err := b.processKustomizationPath(b.AssumeTargetInKustomizationPath)
	if err != nil {
		b.Decorator.GetLogger().Printf("error processing kustomize path: %v, error: %v\n", b.AssumeTargetInKustomizationPath, err)
		return err
	}
	resFromKustomizationPath := b.find(b.Decorator.GetName(), b.Decorator.GetType(), resMapFromKustomizationPath)
	if resFromKustomizationPath == nil {
		b.Decorator.GetLogger().Printf("unable to find target resource: %v in kustomization path: %v\n", b.Decorator.GetName(), b.AssumeTargetInKustomizationPath)
	} else {
		data, err := resFromKustomizationPath.GetFieldValue("data")
		if err != nil {
			b.Decorator.GetLogger().Printf("error extracting data map from target resource: %v in kustomization path: %v, error: %v\n", b.Decorator.GetName(), b.AssumeTargetInKustomizationPath, err)
			return err
		}
		strData := make(map[string]string)
		for k, v := range data.(map[string]interface{}) {
			strData[k] = v.(string)
		}
		err = b.appendData(tempResource, strData, true)
		if err != nil {
			b.Decorator.GetLogger().Printf("error appending data from target resource: %v in kustomization path: %v, error: %v\n", b.Decorator.GetName(), b.AssumeTargetInKustomizationPath, err)
			return err
		}
	}
	return nil
}

func (b *Base) processKustomizationPath(kustomizationPath string) (resmap.ResMap, error) {
	ldr, err := loader.NewLoader(loader.RestrictionNone, validator.NewKustValidator(), kustomizationPath, fs.MakeFsOnDisk())
	if err != nil {
		return nil, err
	}
	defer ldr.Cleanup()

	kt, err := target.NewKustTarget(ldr, b.Rf, transformer.NewFactoryImpl(), plugins.NewLoader(plugins.ActivePluginConfig(), b.Rf))
	if err != nil {
		return nil, err
	}
	return kt.MakeCustomizedResMap()
}

func (b *Base) executeBasicTransform(resource *resource.Resource, m resmap.ResMap) error {
	b.Decorator.GetLogger().Printf("executeBasicTransform() for resource: %v...\n", resource)

	if err := b.appendData(resource, b.Decorator.GetConfigData(), false); err != nil {
		b.Decorator.GetLogger().Printf("error appending data to resource: %v, error: %v\n", b.Decorator.GetName(), err)
		return err
	}

	if !b.Decorator.GetDisableNameSuffixHash() {
		if err := m.Remove(resource.CurId()); err != nil {
			b.Decorator.GetLogger().Printf("error removing original resource on name change: %v\n", err)
			return err
		}
		newResource := b.Rf.RF().FromMapAndOption(resource.Map(), &types.GeneratorArgs{Behavior: "replace"}, &types.GeneratorOptions{DisableNameSuffixHash: false})
		if err := m.Append(newResource); err != nil {
			b.Decorator.GetLogger().Printf("error re-adding resource on name change: %v\n", err)
			return err
		}
		b.Decorator.GetLogger().Printf("resource should have hashing enabled: %v\n", newResource)
	}
	return nil
}

func (b *Base) executeNameReferencesTransformer(m resmap.ResMap) error {
	nameReferenceTransformer := transformers.NewNameReferenceTransformer(b.tConfig.NameReference)
	return nameReferenceTransformer.Transform(m)
}

func (b *Base) find(name string, resourceType string, m resmap.ResMap) *resource.Resource {
	for _, res := range m.Resources() {
		if res.GetKind() == resourceType && res.GetName() == b.Decorator.GetName() {
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

func (b *Base) appendData(res *resource.Resource, data map[string]string, straightCopy bool) error {
	for k, v := range data {
		pathToField := []string{"data", k}
		err := transformers.MutateField(
			res.Map(),
			pathToField,
			true,
			func(interface{}) (interface{}, error) {
				var val string
				if !straightCopy && b.Decorator.ShouldBase64EncodeConfigData() {
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
