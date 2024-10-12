// Package k8s provides an interface to interact with Kubernetes resources.
package k8s

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Option is a functional option for the Controller.
type Option = func(*Controller)

// Controller is a Kubernetes controller.
type Controller struct {
	client dynamic.Interface
	mapper *restmapper.DeferredDiscoveryRESTMapper
	ctx    context.Context

	Timeout time.Duration
}

// WithTimeout sets the timeout for the controller.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Controller) {
		c.Timeout = timeout
	}
}

// NewController creates a new Kubernetes controller.
func NewController(opts ...Option) (*Controller, error) {
	_inst := new(Controller)
	for _, opt := range opts {
		opt(_inst)
	}

	cfg, err := getKubeConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubeconfig")
	}

	cfg.Timeout = _inst.Timeout
	clt, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dynamic client")
	}

	dc, _ := discovery.NewDiscoveryClientForConfig(cfg)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	_inst.mapper = mapper
	_inst.client = clt
	_inst.ctx = context.Background()
	return _inst, nil
}

// GetResource gets a resource from the Kubernetes cluster.
func (c *Controller) GetResource(ctx context.Context, namespace, name string, resource schema.GroupVersionKind, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	if ctx == nil {
		ctx = c.ctx
	}

	mapping, err := c.mapper.RESTMapping(schema.GroupKind{
		Group: resource.Group,
		Kind:  resource.Kind,
	}, resource.Version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get REST mapping")
	}

	client := c.client.Resource(mapping.Resource)
	namespacedClient := client.Namespace(namespace)
	res, err := namespacedClient.Get(ctx, name, opts)
	if err != nil {
		res, err = client.Get(ctx, name, opts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get resource")
		}
	}
	return res, nil
}

// CreateResource creates a resource in the Kubernetes cluster.
func (c *Controller) CreateResource(ctx context.Context, namespace string, resource *unstructured.Unstructured, opts metav1.CreateOptions) (*unstructured.Unstructured, error) {
	gvk := resource.GroupVersionKind()
	client := c.client.Resource(schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind) + "s",
	})

	if _, err := c.GetResource(ctx, namespace, resource.GetName(), gvk, metav1.GetOptions{}); err == nil {
		return c.UpdateResource(ctx, namespace, resource, metav1.UpdateOptions{})
	}

	namespacedClient := client.Namespace(namespace)
	res, err := namespacedClient.Create(ctx, resource, opts)
	if err != nil {
		res, err = client.Create(ctx, resource, opts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create resource")
		}
	}
	return res, nil
}

// CreateSecret creates a secret in the Kubernetes cluster.
func (c *Controller) CreateSecret(ctx context.Context, namespace string, secret *coreV1.Secret, opts metav1.CreateOptions) (*unstructured.Unstructured, error) {
	runtimeObject := &unstructured.Unstructured{
		Object: map[string]any{
			"metadata": map[string]any{
				"name":        secret.Name,
				"namespace":   namespace,
				"labels":      secret.Labels,
				"annotations": secret.Annotations,
			},
			"stringData": secret.StringData,
		},
	}
	runtimeObject.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	})
	return c.CreateResource(ctx, namespace, runtimeObject, opts)
}

// UpdateResource updates a resource in the Kubernetes cluster.
func (c *Controller) UpdateResource(ctx context.Context, namespace string, resource *unstructured.Unstructured, opts metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	gvk := resource.GroupVersionKind()
	client := c.client.Resource(schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind) + "s",
	})
	namespacedClient := client.Namespace(namespace)
	res, err := namespacedClient.Update(ctx, resource, opts)
	if err != nil {
		res, err = client.Update(ctx, resource, opts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update resource")
		}
	}
	return res, nil
}

func getKubeConfig() (config *rest.Config, err error) {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		// in-cluster config
		return rest.InClusterConfig()
	}
	// out-of-cluster config
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return clientConfig.ClientConfig()
}
