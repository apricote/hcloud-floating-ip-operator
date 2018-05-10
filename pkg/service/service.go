package service

import (
	"sync"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/client-go/kubernetes"

	hcloudv1alpha1 "github.com/apricote/hcloud-floating-ip-operator/apis/hcloud/v1alpha1"
	"github.com/apricote/hcloud-floating-ip-operator/pkg/log"
)

type Syncer interface {
	EnsureFloatingIP(pt *hcloudv1alpha1.FloatingIP) error
	DeleteFloatingIP(name string) error
}

// Service is the service that will ensure that the desired floating ip CRDs are met.
// Service will have running instances of IPAssigners.
type Service struct {
	k8sCli    kubernetes.Interface
	hcloudCli *hcloud.Client
	reg       sync.Map
	logger    log.Logger
}

// NewService returns a new floating ip assigner service.
func NewService(k8sCli kubernetes.Interface, hcloudCli *hcloud.Client, logger log.Logger) *Service {
	return &Service{
		k8sCli:    k8sCli,
		hcloudCli: hcloudCli,
		reg:       sync.Map{},
		logger:    logger,
	}
}

// EnsureFloatingIP satisfies ServiceSyncer interface.
func (c *Service) EnsureFloatingIP(fip *hcloudv1alpha1.FloatingIP) error {
	ipav, ok := c.reg.Load(fip.Name)
	var ipa *IPAssigner

	// We are already running.
	if ok {
		ipa = ipav.(*IPAssigner)
		// If not the same spec means options have changed, so we don't longer need this ip assigner.
		if !ipa.SameSpec(fip) {
			c.logger.Infof("spec of %s changed, recreating ip assigner", ipa.fip.Name)
			if err := c.DeleteFloatingIP(fip.Name); err != nil {
				return err
			}
		} else { // We are ok, nothing changed.
			return nil
		}
	}

	// Create an ip assigner.
	fipCopy := fip.DeepCopy()
	ipa = NewIPAssigner(fipCopy, c.k8sCli, c.hcloudCli, c.logger)
	c.reg.Store(fip.Name, ipa)
	return ipa.Start()
	// TODO: garbage collection.
}

// DeleteFloatingIP satisfies ServiceSyncer interface.
func (c *Service) DeleteFloatingIP(name string) error {
	ipav, ok := c.reg.Load(name)
	if !ok {
		return nil
	}

	ipa := ipav.(*IPAssigner)
	if err := ipa.Stop(); err != nil {
		return err
	}

	c.reg.Delete(name)
	return nil
}
