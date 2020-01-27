package qust

import (
	"errors"
	"github.com/qlik-oss/k-apis/config"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"

	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/types"
)

const (
	// a folder named .operator exist in manifestsRoot
	operatorPatchBaseFolder = ".operator"
	FILE_PERMISION          = 0644
)

// merge two selective patch by appending sp2.patches into sp1.patch
// the final result is sp1.patchs updated
func mergeSelectivePatches(sp1, sp2 *config.SelectivePatch) (*config.SelectivePatch, error) {
	if sp1 == nil && sp2 == nil {
		return nil, errors.New("Two objects are nil")
	}
	if sp1 == nil {
		return sp2, nil
	}
	if sp2 == nil {
		return sp1, nil
	}
	if sp1.ApiVersion != sp2.ApiVersion || sp1.Kind != sp2.Kind || sp1.Metadata["name"] != sp2.Metadata["name"] {
		err := errors.New("Cannot merge selective patches [ " + sp1.Metadata["name"] + " != " + sp2.Metadata["name"])
		return nil, err
	}
	sp1.Patches = append(sp1.Patches, sp2.Patches...)
	return sp1, nil
}

func YamlToWriter(w io.Writer, yml interface{}) error {
	d, err := yaml.Marshal(yml)
	w.Write(d)
	return err
}

// add a resource file in kustomization if not that exist
func addResourceToKustomization(rsFileName string, kustFile string) error {
	fn := func(kust *types.Kustomization) {
		// if the resource exist no need to add again
		if !isResourcesInKust(rsFileName, kust) {
			kust.Resources = append(kust.Resources, rsFileName)
		}
	}
	return kustFileHelper(kustFile, fn)
}

// it is a helper to add any file as a resource,transfomer, generator, etc
// fn will define what type of file it would be
func kustFileHelper(kustFile string, fn func(*types.Kustomization)) error {
	kust := &types.Kustomization{}
	content, err := ioutil.ReadFile(kustFile)
	if err != nil {
		return err
	}
	yaml.Unmarshal(content, kust)

	fn(kust)

	kust.FixKustomizationPostUnmarshalling()
	// there is a bug if not put nil https://github.com/kubernetes-sigs/kustomize/pull/1004/files

	d, err := yaml.Marshal(kust)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(kustFile, d, FILE_PERMISION)
	return err
}

func isResourcesInKust(rsFileName string, kust *types.Kustomization) bool {
	for _, k := range kust.Resources {
		if k == rsFileName {
			return true
		}
	}
	return false
}

func getSelector(kind, svc string) *types.Selector {
	return &types.Selector{
		Gvk: resid.Gvk{
			Kind: kind,
		},
		LabelSelector: "app=" + svc,
	}
}
