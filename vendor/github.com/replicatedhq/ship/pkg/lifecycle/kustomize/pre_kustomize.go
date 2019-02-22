package kustomize

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/util"
	yaml "gopkg.in/yaml.v2"
	ktypes "sigs.k8s.io/kustomize/pkg/types"
)

func (l *Kustomizer) PreExecute(ctx context.Context, step api.Step) error {
	// Check if the 'base' already includes a kustomization.yaml
	// if it does, and that refers to another base, we should apply those patches to the upstream base, and then use that in the future
	newBase, err := l.containsBase(ctx, step.Kustomize.Base)
	if err != nil {
		return errors.Wrap(err, "maybe find existing base")
	}

	oldBase := step.Kustomize.Base
	if newBase != "" {
		// use this base for future steps
		step.Kustomize.Base = newBase
	}

	// Split multi doc yaml first as it will be unmarshalled incorrectly in the following steps
	if err := util.MaybeSplitMultidocYaml(ctx, l.FS, step.Kustomize.Base); err != nil {
		return errors.Wrap(err, "maybe split multi doc yaml")
	}

	if err := l.maybeSplitListYaml(ctx, step.Kustomize.Base); err != nil {
		return errors.Wrap(err, "maybe split list yaml")
	}

	if newBase != "" {
		err = l.runProvidedOverlays(ctx, oldBase, newBase)
		if err != nil {
			return errors.Wrap(err, "run provided kustomization yaml")
		}
	}

	if err := l.initialKustomizeRun(ctx, *step.Kustomize); err != nil {
		return errors.Wrap(err, "initial kustomize run")
	}

	return nil
}

func (l *Kustomizer) containsBase(ctx context.Context, path string) (string, error) {
	debug := level.Debug(log.With(l.Logger, "step.type", "render", "render.phase", "execute"))
	debug.Log("event", "readDir", "path", path)

	files, err := l.FS.ReadDir(path)
	if err != nil {
		return "", errors.Wrapf(err, "read files in %s", path)
	}

	for _, file := range files {
		if file.Name() == "kustomization.yaml" {
			// read and parse the kustomization yaml
			fileBytes, err := l.FS.ReadFile(filepath.Join(path, file.Name()))
			if err != nil {
				return "", errors.Wrapf(err, "read %s", filepath.Join(path, file.Name()))
			}

			kustomizeResource := ktypes.Kustomization{}

			err = yaml.Unmarshal(fileBytes, &kustomizeResource)
			if err != nil {
				return "", errors.Wrapf(err, "parse file at %s", filepath.Join(path, file.Name()))
			}

			if len(kustomizeResource.Bases) > 0 {
				if len(kustomizeResource.Bases) > 1 {
					return "", errors.New("kustomization.yaml files with multiple bases are not yet supported")
				}

				newBase := filepath.Join(path, kustomizeResource.Bases[0])

				return newBase, nil
			}

			return "", nil
		}
	}

	return "", nil
}

func (l *Kustomizer) maybeSplitListYaml(ctx context.Context, path string) error {
	debug := level.Debug(log.With(l.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "github"))

	debug.Log("event", "readDir", "path", path)
	files, err := l.FS.ReadDir(path)
	if err != nil {
		return errors.Wrapf(err, "read files in %s", path)
	}

	for _, file := range files {
		filePath := filepath.Join(path, file.Name())

		if file.IsDir() {
			return l.maybeSplitListYaml(ctx, filepath.Join(path, file.Name()))
		}

		if filepath.Ext(file.Name()) != ".yaml" && filepath.Ext(file.Name()) != ".yml" {
			// not yaml, nothing to do
			return nil
		}

		fileB, err := l.FS.ReadFile(filePath)
		if err != nil {
			return errors.Wrapf(err, "read %s", filePath)
		}

		k8sYaml := util.ListK8sYaml{}
		if err := yaml.Unmarshal(fileB, &k8sYaml); err != nil {
			return errors.Wrapf(err, "unmarshal %s", filePath)
		}

		if k8sYaml.Kind == "List" {
			listItems := make([]util.MinimalK8sYaml, 0)
			for idx, item := range k8sYaml.Items {
				itemK8sYaml := util.MinimalK8sYaml{}
				itemB, err := yaml.Marshal(item)
				if err != nil {
					return errors.Wrapf(err, "marshal item %d from %s", idx, filePath)
				}

				if err := yaml.Unmarshal(itemB, &itemK8sYaml); err != nil {
					return errors.Wrap(err, "unmarshal item")
				}

				fileName := util.GenerateNameFromMetadata(itemK8sYaml, idx)
				if err := l.FS.WriteFile(filepath.Join(path, fileName+".yaml"), []byte(itemB), os.FileMode(0644)); err != nil {
					return errors.Wrap(err, "write yaml")
				}

				listItems = append(listItems, itemK8sYaml)
			}

			if err := l.FS.Remove(filePath); err != nil {
				return errors.Wrapf(err, "remove k8s list %s", filePath)
			}

			list := util.List{
				APIVersion: k8sYaml.APIVersion,
				Path:       filePath,
				Items:      listItems,
			}

			debug.Log("event", "serializeListsMetadata")
			if err := l.State.SerializeListsMetadata(list); err != nil {
				return errors.Wrapf(err, "serialize list metadata")
			}
		}
	}

	return nil
}

func (l *Kustomizer) initialKustomizeRun(ctx context.Context, step api.Kustomize) error {
	if err := l.writeBase(step.Base); err != nil {
		return errors.Wrap(err, "write base kustomization")
	}

	if err := l.generateTillerPatches(step); err != nil {
		return errors.Wrap(err, "generate tiller patches")
	}

	defer l.FS.RemoveAll(constants.TempApplyOverlayPath)

	built, err := l.kustomizeBuild(constants.TempApplyOverlayPath)
	if err != nil {
		return errors.Wrap(err, "build overlay")
	}

	if err := l.writePostKustomizeFiles(step, built); err != nil {
		return errors.Wrap(err, "write initial kustomized yaml")
	}

	if err := l.replaceOriginal(step.Base, built); err != nil {
		return errors.Wrap(err, "replace original yaml")
	}

	return nil
}

// 'originalBase' should refer to a kustomize originalBase to render, and 'newBase' the kustomize originalBase to overwrite/modify
// 'newBase' will be overwritten with yaml containing the patches from 'originalBase', allowing 'originalBase' to be ignored or discarded
func (l *Kustomizer) runProvidedOverlays(ctx context.Context, originalBase, newBase string) error {
	if err := l.writeBase(newBase); err != nil {
		return errors.Wrap(err, "write new base kustomization")
	}

	built, err := l.kustomizeBuild(originalBase)
	if err != nil {
		return errors.Wrap(err, "build overlay")
	}

	if err := l.replaceOriginal(newBase, built); err != nil {
		return errors.Wrap(err, "replace original yaml")
	}

	return nil
}

func (l *Kustomizer) replaceOriginal(base string, built []util.PostKustomizeFile) error {
	builtMap := make(map[util.MinimalK8sYaml]util.PostKustomizeFile)
	for _, builtFile := range built {
		builtMap[builtFile.Minimal] = builtFile
	}

	if err := l.FS.Walk(base, func(targetPath string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "failed to walk base path")
		}

		if !l.shouldAddFileToBase([]string{}, targetPath) {
			if strings.HasSuffix(targetPath, "kustomization.yaml") {
				if err := l.FS.Remove(targetPath); err != nil {
					return errors.Wrap(err, "remove kustomization yaml")
				}
			}

			return nil
		}

		originalFileB, err := l.FS.ReadFile(targetPath)
		if err != nil {
			return errors.Wrap(err, "read original file")
		}

		originalMinimal := util.MinimalK8sYaml{}
		if err := yaml.Unmarshal(originalFileB, &originalMinimal); err != nil {
			return errors.Wrap(err, "unmarshal original")
		}

		if originalMinimal.Kind == "CustomResourceDefinition" {
			// Skip CRDs
			return nil
		}

		initKustomized, exists := builtMap[originalMinimal]
		if !exists {
			// Skip if the file does not have a kustomized equivalent
			return nil
		}

		if err := l.FS.Remove(targetPath); err != nil {
			return errors.Wrap(err, "remove original file")
		}

		initKustomizedB, err := yaml.Marshal(initKustomized.Full)
		if err != nil {
			return errors.Wrap(err, "marshal init kustomized")
		}

		if err := l.FS.WriteFile(targetPath, initKustomizedB, info.Mode()); err != nil {
			return errors.Wrap(err, "write init kustomized file")
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "replace original with init kustomized")
	}

	return nil
}
