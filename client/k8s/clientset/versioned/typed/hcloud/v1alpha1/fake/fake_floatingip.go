package fake

import (
	v1alpha1 "github.com/apricote/hcloud-floating-ip-operator/apis/hcloud/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFloatingIPs implements FloatingIPInterface
type FakeFloatingIPs struct {
	Fake *FakeHcloudV1alpha1
}

var floatingipsResource = schema.GroupVersionResource{Group: "hcloud.apricote.de", Version: "v1alpha1", Resource: "floatingips"}

var floatingipsKind = schema.GroupVersionKind{Group: "hcloud.apricote.de", Version: "v1alpha1", Kind: "FloatingIP"}

// Get takes name of the floatingIP, and returns the corresponding floatingIP object, and an error if there is any.
func (c *FakeFloatingIPs) Get(name string, options v1.GetOptions) (result *v1alpha1.FloatingIP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(floatingipsResource, name), &v1alpha1.FloatingIP{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FloatingIP), err
}

// List takes label and field selectors, and returns the list of FloatingIPs that match those selectors.
func (c *FakeFloatingIPs) List(opts v1.ListOptions) (result *v1alpha1.FloatingIPList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(floatingipsResource, floatingipsKind, opts), &v1alpha1.FloatingIPList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FloatingIPList{}
	for _, item := range obj.(*v1alpha1.FloatingIPList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested floatingIPs.
func (c *FakeFloatingIPs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(floatingipsResource, opts))
}

// Create takes the representation of a floatingIP and creates it.  Returns the server's representation of the floatingIP, and an error, if there is any.
func (c *FakeFloatingIPs) Create(floatingIP *v1alpha1.FloatingIP) (result *v1alpha1.FloatingIP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(floatingipsResource, floatingIP), &v1alpha1.FloatingIP{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FloatingIP), err
}

// Update takes the representation of a floatingIP and updates it. Returns the server's representation of the floatingIP, and an error, if there is any.
func (c *FakeFloatingIPs) Update(floatingIP *v1alpha1.FloatingIP) (result *v1alpha1.FloatingIP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(floatingipsResource, floatingIP), &v1alpha1.FloatingIP{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FloatingIP), err
}

// Delete takes name of the floatingIP and deletes it. Returns an error if one occurs.
func (c *FakeFloatingIPs) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(floatingipsResource, name), &v1alpha1.FloatingIP{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFloatingIPs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(floatingipsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FloatingIPList{})
	return err
}

// Patch applies the patch and returns the patched floatingIP.
func (c *FakeFloatingIPs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FloatingIP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(floatingipsResource, name, data, subresources...), &v1alpha1.FloatingIP{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FloatingIP), err
}
