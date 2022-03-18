package core

import (
	"context"
	"time"

	"github.com/yubo/apiserver/pkg/apis/authentication"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/watch"
)

// ServiceAccountsGetter has a method to return a ServiceAccountInterface.
// A group's client should implement this interface.
type ServiceAccountsGetter interface {
	ServiceAccounts() ServiceAccountInterface
}

// ServiceAccountInterface has methods to work with ServiceAccount resources.
type ServiceAccountInterface interface {
	Create(ctx context.Context, serviceAccount *api.ServiceAccount, opts api.CreateOptions) (*api.ServiceAccount, error)
	Update(ctx context.Context, serviceAccount *api.ServiceAccount, opts api.UpdateOptions) (*api.ServiceAccount, error)
	Delete(ctx context.Context, name string, opts api.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts api.DeleteOptions, listOpts api.ListOptions) error
	Get(ctx context.Context, name string, opts api.GetOptions) (*api.ServiceAccount, error)
	List(ctx context.Context, opts api.ListOptions) (*api.ServiceAccountList, error)
	Watch(ctx context.Context, opts api.ListOptions) (watch.Interface, error)
	//Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts api.PatchOptions, subresources ...string) (result *api.ServiceAccount, err error)
	CreateToken(ctx context.Context, serviceAccountName string, tokenRequest *authentication.TokenRequest, opts api.CreateOptions) (*authentication.TokenRequest, error)

	//ServiceAccountExpansion
}

// serviceAccounts implements ServiceAccountInterface
type serviceAccounts struct {
	client rest.Interface
	ns     string
}

// newServiceAccounts returns a ServiceAccounts
func newServiceAccounts(c *CoreV1Client) *serviceAccounts {
	return &serviceAccounts{
		client: c.RESTClient(),
	}
}

// Get takes name of the serviceAccount, and returns the corresponding serviceAccount object, and an error if there is any.
func (c *serviceAccounts) Get(ctx context.Context, name string, options api.GetOptions) (result *api.ServiceAccount, err error) {
	result = &api.ServiceAccount{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("serviceaccounts").
		Name(name).
		VersionedParams(&options, rest.NewParameterCodec()).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ServiceAccounts that match those selectors.
func (c *serviceAccounts) List(ctx context.Context, opts api.ListOptions) (result *api.ServiceAccountList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &api.ServiceAccountList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("serviceaccounts").
		VersionedParams(&opts, c.client.Codec()).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested serviceAccounts.
func (c *serviceAccounts) Watch(ctx context.Context, opts api.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("serviceaccounts").
		VersionedParams(&opts, c.client.Codec()).
		Timeout(timeout).
		Watch(ctx, c)
}

// Create takes the representation of a serviceAccount and creates it.  Returns the server's representation of the serviceAccount, and an error, if there is any.
func (c *serviceAccounts) Create(ctx context.Context, serviceAccount *api.ServiceAccount, opts api.CreateOptions) (result *api.ServiceAccount, err error) {
	result = &api.ServiceAccount{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("serviceaccounts").
		VersionedParams(&opts, c.client.Codec()).
		Body(serviceAccount).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a serviceAccount and updates it. Returns the server's representation of the serviceAccount, and an error, if there is any.
func (c *serviceAccounts) Update(ctx context.Context, serviceAccount *api.ServiceAccount, opts api.UpdateOptions) (result *api.ServiceAccount, err error) {
	result = &api.ServiceAccount{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("serviceaccounts").
		Name(serviceAccount.Name).
		VersionedParams(&opts, c.client.Codec()).
		Body(serviceAccount).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the serviceAccount and deletes it. Returns an error if one occurs.
func (c *serviceAccounts) Delete(ctx context.Context, name string, opts api.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("serviceaccounts").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *serviceAccounts) DeleteCollection(ctx context.Context, opts api.DeleteOptions, listOpts api.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("serviceaccounts").
		VersionedParams(&listOpts, c.client.Codec()).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched serviceAccount.
//func (c *serviceAccounts) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts api.PatchOptions, subresources ...string) (result *api.ServiceAccount, err error) {
//	result = &api.ServiceAccount{}
//	err = c.client.Patch(pt).
//		Namespace(c.ns).
//		Resource("serviceaccounts").
//		Name(name).
//		SubResource(subresources...).
//		VersionedParams(&opts, scheme.ParameterCodec).
//		Body(data).
//		Do(ctx).
//		Into(result)
//	return
//}

// CreateToken takes the representation of a tokenRequest and creates it.  Returns the server's representation of the tokenRequest, and an error, if there is any.
func (c *serviceAccounts) CreateToken(ctx context.Context, serviceAccountName string, tokenRequest *authentication.TokenRequest, opts api.CreateOptions) (result *authentication.TokenRequest, err error) {
	result = &authentication.TokenRequest{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("serviceaccounts").
		Name(serviceAccountName).
		SubResource("token").
		VersionedParams(&opts, c.client.Codec()).
		Body(tokenRequest).
		Do(ctx).
		Into(result)
	return
}
