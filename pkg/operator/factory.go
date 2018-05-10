package operator

import (
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/spotahome/kooper/client/crd"
	"github.com/spotahome/kooper/operator"
	"github.com/spotahome/kooper/operator/controller"
	"k8s.io/client-go/kubernetes"

	floatingipk8scli "github.com/apricote/hcloud-floating-ip-operator/client/k8s/clientset/versioned"
	"github.com/apricote/hcloud-floating-ip-operator/pkg/log"
)

// New returns floating ip operator.
func New(cfg Config, floatingIPClie floatingipk8scli.Interface, crdCli crd.Interface, kubeCli kubernetes.Interface, hcloudCli *hcloud.Client, logger log.Logger) (operator.Operator, error) {

	// Create crd.
	ptCRD := newFloatingIPCRD(floatingIPClie, crdCli, kubeCli)

	// Create handler.
	handler := newHandler(kubeCli, hcloudCli, logger)

	// Create controller.
	ctrl := controller.NewSequential(cfg.ResyncPeriod, handler, ptCRD, nil, logger)

	// Assemble CRD and controller to create the operator.
	return operator.NewOperator(ptCRD, ctrl, logger), nil
}
