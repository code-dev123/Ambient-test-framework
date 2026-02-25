// pkg/manifest/applier.go
package manifest

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"go.uber.org/multierr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// Applier handles applying and deleting Kubernetes/Istio YAML manifests.
type Applier struct {
	dynClient dynamic.Interface
	mapper    *restmapper.DeferredDiscoveryRESTMapper
	logger    *slog.Logger
}

// NewApplier creates a new Applier.
func NewApplier(
	dynClient dynamic.Interface,
	mapper *restmapper.DeferredDiscoveryRESTMapper,
	logger *slog.Logger,
) *Applier {
	return &Applier{
		dynClient: dynClient,
		mapper:    mapper,
		logger:    logger,
	}
}

// ApplyResult tracks what was applied so we can delete in reverse order.
type ApplyResult struct {
	Resources []AppliedResource
}

// AppliedResource records enough info to locate and delete a resource.
type AppliedResource struct {
	GVR       schema.GroupVersionResource
	Namespace string
	Name      string
}

// ApplyFolder reads all .yaml/.yml files from an fs.FS path,
// templates them with the provided values, and applies them
// in filesystem order. Returns an ApplyResult for teardown.
func (a *Applier) ApplyFolder(ctx context.Context, fsys fs.FS,
	dir string, values TemplateValues) (*ApplyResult, error) {

	result := &ApplyResult{}

	files, err := fs.Glob(fsys, filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("glob manifests: %w", err)
	}

	// Also match .yml
	ymlFiles, _ := fs.Glob(fsys, filepath.Join(dir, "*.yml"))
	files = append(files, ymlFiles...)
	sort.Strings(files) // deterministic ordering

	for _, f := range files {
		raw, err := fs.ReadFile(fsys, f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}

		// Template the manifest (inject namespace, names, etc.)
		rendered, err := RenderTemplate(string(raw), values)
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", f, err)
		}

		// A single file can contain multiple YAML docs (--- separated)
		docs := splitYAMLDocs(rendered)
		for _, doc := range docs {
			applied, err := a.applyOne(ctx, doc)
			if err != nil {
				return result, fmt.Errorf("apply %s: %w", f, err)
			}
			result.Resources = append(result.Resources, *applied)
		}

		a.logger.Info("applied manifest",
			"file", f,
			"resources", len(docs))
	}

	return result, nil
}

// ApplyFile applies a single manifest file.
func (a *Applier) ApplyFile(ctx context.Context, fsys fs.FS,
	path string, values TemplateValues) (*ApplyResult, error) {

	raw, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	rendered, err := RenderTemplate(string(raw), values)
	if err != nil {
		return nil, fmt.Errorf("template %s: %w", path, err)
	}

	result := &ApplyResult{}
	docs := splitYAMLDocs(rendered)
	for _, doc := range docs {
		applied, err := a.applyOne(ctx, doc)
		if err != nil {
			return result, err
		}
		result.Resources = append(result.Resources, *applied)
	}
	return result, nil
}

// DeleteResult tears down everything from an ApplyResult in
// reverse order (LIFO) — routes before gateways, policies before workloads.
func (a *Applier) DeleteResult(ctx context.Context,
	result *ApplyResult) error {

	if result == nil {
		return nil
	}

	var errs []error
	// Delete in reverse order of creation
	for i := len(result.Resources) - 1; i >= 0; i-- {
		r := result.Resources[i]
		err := a.dynClient.Resource(r.GVR).
			Namespace(r.Namespace).
			Delete(ctx, r.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("delete %s/%s: %w",
				r.Namespace, r.Name, err))
		} else {
			a.logger.Info("deleted resource",
				"kind", r.GVR.Resource,
				"namespace", r.Namespace,
				"name", r.Name)
		}
	}
	return multierr.Combine(errs...)
}

// applyOne decodes a single YAML doc and applies it via server-side apply.
func (a *Applier) applyOne(ctx context.Context,
	yamlDoc string) (*AppliedResource, error) {

	obj := &unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode([]byte(yamlDoc), nil, obj)
	if err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	mapping, err := a.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("rest mapping for %s: %w", gvk, err)
	}

	// Server-side apply — idempotent, handles conflicts
	_, err = a.dynClient.Resource(mapping.Resource).
		Namespace(obj.GetNamespace()).
		Apply(ctx, obj.GetName(),
			obj, metav1.ApplyOptions{
				FieldManager: "ambient-test-framework",
				Force:        true,
			})
	if err != nil {
		return nil, fmt.Errorf("apply %s/%s: %w",
			obj.GetNamespace(), obj.GetName(), err)
	}

	return &AppliedResource{
		GVR:       mapping.Resource,
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}, nil
}

func splitYAMLDocs(raw string) []string {
	docs := strings.Split(raw, "\n---")
	var out []string
	for _, d := range docs {
		trimmed := strings.TrimSpace(d)
		if trimmed != "" && trimmed != "---" {
			out = append(out, trimmed)
		}
	}
	return out
}
