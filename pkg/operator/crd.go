package operator

import (
	"github.com/spotahome/kooper/client/crd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	hcloudv1alpha1 "github.com/apricote/hcloud-floating-ip-operator/apis/hcloud/v1alpha1"
	floatingipk8scli "github.com/apricote/hcloud-floating-ip-operator/client/k8s/clientset/versioned"
)

// floatingIPCRD is the crd floating ip.
type floatingIPCRD struct {
	crdCli        crd.Interface
	kubecCli      kubernetes.Interface
	floatingIPCli floatingipk8scli.Interface
}

func newFloatingIPCRD(floatingIPCli floatingipk8scli.Interface, crdCli crd.Interface, kubeCli kubernetes.Interface) *floatingIPCRD {
	return &floatingIPCRD{
		crdCli:        crdCli,
		floatingIPCli: floatingIPCli,
		kubecCli:      kubeCli,
	}
}

// floatingIPCRD satisfies resource.crd interface.
func (p *floatingIPCRD) Initialize() error {
	crd := crd.Conf{
		Kind:       hcloudv1alpha1.FloatingIPKind,
		NamePlural: hcloudv1alpha1.FloatingIPNamePlural,
		Group:      hcloudv1alpha1.SchemeGroupVersion.Group,
		Version:    hcloudv1alpha1.SchemeGroupVersion.Version,
		Scope:      hcloudv1alpha1.FloatingIPScope,
	}

	return p.crdCli.EnsurePresent(crd)
}

// GetListerWatcher satisfies resource.crd interface (and retrieve.Retriever).
func (p *floatingIPCRD) GetListerWatcher() cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return p.floatingIPCli.HcloudV1alpha1().FloatingIPs().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return p.floatingIPCli.HcloudV1alpha1().FloatingIPs().Watch(options)
		},
	}
}

// GetObject satisfies resource.crd interface (and retrieve.Retriever).
func (p *floatingIPCRD) GetObject() runtime.Object {
	return &hcloudv1alpha1.FloatingIP{}
}
