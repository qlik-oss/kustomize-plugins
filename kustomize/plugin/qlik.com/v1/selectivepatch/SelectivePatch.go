package main

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/loader"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/kustomize/v3/plugin/builtin"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	Enabled bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Patches []types.Patch `json:"patches,omitempty" yaml:"patches,omitempty"`
	ts      []transformers.Transformer
}

//nolint: go-lint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) makeIndividualPatches(pat types.Patch) ([]byte, error) {
	var s struct {
		types.Patch
	}
	s.Patch = pat
	return yaml.Marshal(s)
}

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, c []byte) error {

	// To avoid https://github.com/kubernetes-sigs/kustomize/blob/master/docs/FAQ.md#security-file-foo-is-not-in-or-below-bar
	// start of work around
	fSys := fs.MakeRealFS()
	newLdr, er := loader.NewLoader(loader.RestrictionNone, ldr.Validator(), ldr.Root(), fSys)
	if er != nil {
		return errors.Wrapf(er, "Cannot create new laoder from default loader")
	}
	// End of work around
	err := yaml.Unmarshal(c, p)
	if err != nil {
		return errors.Wrapf(err, "Inside unmarshal "+string(c))
	}
	for _, v := range p.Patches {
		//fmt.Println(v.Path)
		b, _ := p.makeIndividualPatches(v)
		prefixer := builtin.NewPatchTransformerPlugin()
		err = prefixer.Config(newLdr, rf, b)
		if err != nil {
			return errors.Wrapf(
				err, "stringprefixer configure")
		}
		p.ts = append(p.ts, prefixer)

	}
	return nil
}

func (p *plugin) Transform(m resmap.ResMap) error {
	if !p.Enabled {
		return nil
	}
	for _, t := range p.ts {
		err := t.Transform(m)
		if err != nil {
			return err
		}
	}
	return nil
}
