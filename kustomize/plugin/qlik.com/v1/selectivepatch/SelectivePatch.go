package main

import (
	"log"

	"github.com/pkg/errors"
	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/loader"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/kustomize/v3/plugin/builtin"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	Enabled bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Patches []types.Patch `json:"patches,omitempty" yaml:"patches,omitempty"`
	ts      []resmap.Transformer
}

//nolint: go-lint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("SelectivePatch")
}

func (p *plugin) makeIndividualPatches(pat types.Patch) ([]byte, error) {
	var s struct {
		types.Patch
	}
	s.Patch = pat
	return yaml.Marshal(s)
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) error {
	// To avoid https://github.com/kubernetes-sigs/kustomize/blob/master/docs/FAQ.md#security-file-foo-is-not-in-or-below-bar
	// start of work around
	fSys := fs.MakeRealFS()
	newLdr, err := loader.NewLoader(loader.RestrictionNone, ldr.Validator(), ldr.Root(), fSys)
	if err != nil {
		logger.Printf("error creating a new loader from default loader, error: %v\n", err)
		return errors.Wrapf(err, "Cannot create new loader from default loader")
	}
	// End of work around
	if err := yaml.Unmarshal(c, p); err != nil {
		logger.Printf("error unmarshalling bytes: %v, error: %v\n", string(c), err)
		return errors.Wrapf(err, "Inside unmarshal "+string(c))
	}
	for _, v := range p.Patches {
		//fmt.Println(v.Path)
		b, _ := p.makeIndividualPatches(v)
		prefixer := builtin.NewPatchTransformerPlugin()
		err = prefixer.Config(newLdr, rf, b)
		if err != nil {
			logger.Printf("error executing PatchTransformerPlugin.Config(), error: %v\n", err)
			return errors.Wrapf(err, "stringprefixer configure")
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
			logger.Printf("error executing Transform(), error: %v\n", err)
			return err
		}
	}
	return nil
}
