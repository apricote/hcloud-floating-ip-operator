package service

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/hetznercloud/hcloud-go/hcloud"

	hcloudv1alpha1 "github.com/apricote/hcloud-floating-ip-operator/apis/hcloud/v1alpha1"
	"github.com/apricote/hcloud-floating-ip-operator/pkg/log"
)

// TimeWrapper is a wrapper around time so it can be mocked
type TimeWrapper interface {
	// After is the same as time.After
	After(d time.Duration) <-chan time.Time
	// Now is the same as Now.
	Now() time.Time
}

type timeStd struct{}

func (t *timeStd) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (t *timeStd) Now() time.Time                         { return time.Now() }

// IPAssigner will verify ip assignment at regular intervals.
type IPAssigner struct {
	fip       *hcloudv1alpha1.FloatingIP
	k8sCli    kubernetes.Interface
	hcloudCli *hcloud.Client
	logger    log.Logger
	time      TimeWrapper

	running bool
	mutex   sync.Mutex
	stopC   chan struct{}
}

// NewIPAssigner returns a new ip assigner.
func NewIPAssigner(fip *hcloudv1alpha1.FloatingIP, k8sCli kubernetes.Interface, hcloudCli *hcloud.Client, logger log.Logger) *IPAssigner {
	return &IPAssigner{
		fip:       fip,
		k8sCli:    k8sCli,
		hcloudCli: hcloudCli,
		logger:    logger,
		time:      &timeStd{},
	}
}

// NewCustomIPAssigner is a constructor that lets you customize everything on the object construction.
func NewCustomIPAssigner(fip *hcloudv1alpha1.FloatingIP, k8sCli kubernetes.Interface, hcloudCli *hcloud.Client, time TimeWrapper, logger log.Logger) *IPAssigner {
	return &IPAssigner{
		fip:       fip,
		k8sCli:    k8sCli,
		hcloudCli: hcloudCli,
		logger:    logger,
		time:      time,
	}
}

// SameSpec checks if the ip assigner has the same spec.
func (p *IPAssigner) SameSpec(fip *hcloudv1alpha1.FloatingIP) bool {
	return reflect.DeepEqual(p.fip.Spec, fip.Spec)
}

// Start will run the ip assigner at regular intervals.
func (p *IPAssigner) Start() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.running {
		return fmt.Errorf("already running")
	}

	p.stopC = make(chan struct{})
	p.running = true

	go func() {
		p.logger.Infof("started %s ip assigner", p.fip.Name)
		if err := p.run(); err != nil {
			p.logger.Errorf("error executing ip assigner: %s", err)
		}
	}()

	return nil
}

// Stop stops the ip assigner.
func (p *IPAssigner) Stop() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.running {
		close(p.stopC)
		p.logger.Infof("stopped %s ip assigner", p.fip.Name)
	}

	p.running = false
	return nil
}

// run will run the loop that will kill eventually the required pods.
func (p *IPAssigner) run() error {
	for {
		select {
		case <-p.time.After(time.Duration(p.fip.Spec.IntervalSeconds) * time.Second):
			if err := p.assign(); err != nil {
				p.logger.Errorf("error terminating pods: %s", err)
			}
		case <-p.stopC:
			return nil
		}
	}
}

// asign will verify current assignment of the floating ip and change
// assignment to a node matching the nodeSelector in case the floating
// ip is currently not correctly assigned
func (p *IPAssigner) assign() error {
	// Get all probable targets.
	nodes, err := p.getProbableNodes()
	if err != nil {
		return err
	}

	total := len(nodes.Items)
	if total == 0 {
		p.logger.Errorf("0 nodes probable targets")
		return fmt.Errorf("%s ip assigner: 0 nodes probable targets", p.fip.Name)
	}

	// Get random pods.
	target := p.getRandomNode(nodes)
	p.logger.Infof("%s ip assigner will assign to node %s", p.fip.Name, target.Name)

	// Assign
	hetznerIP, err := p.findHCloudFloatingIP()
	if err != nil {
		return err
	}

	server, err := p.findServer(&target)
	if err != nil {
		return err
	}

	_, _, err = p.hcloudCli.FloatingIP.Assign(context.TODO(), hetznerIP, server)
	if err != nil {
		return err
	}

	p.logger.Infof("%s ip assigner assigned to node %s", p.fip.Name, target.Name)
	return nil
}

// Gets all the pods filtered that can be a target of termination.
func (p *IPAssigner) getProbableNodes() (*corev1.NodeList, error) {
	set := labels.Set(p.fip.Spec.NodeSelector)
	slc := set.AsSelector()
	opts := metav1.ListOptions{
		LabelSelector: slc.String(),
	}
	return p.k8sCli.CoreV1().Nodes().List(opts)
}

// getRandomNode will select one node randomly.
func (p *IPAssigner) getRandomNode(nodes *corev1.NodeList) corev1.Node {
	// Return random index.
	items := nodes.Items
	return items[rand.Intn(len(items))]
}

// findHCloudFloatingIP will return a hcloud FloatingIP resource that matches
// the ip specified in the FloatingIP CRD resource
func (p *IPAssigner) findHCloudFloatingIP() (*hcloud.FloatingIP, error) {
	ip := net.ParseIP(p.fip.Spec.IP)
	if ip == nil {
		return nil, fmt.Errorf("error parsing ip from spec: %s", p.fip.Spec.IP)
	}

	fips, err := p.hcloudCli.FloatingIP.All(context.TODO())
	if err != nil {
		return nil, err
	}

	var hetznerIP *hcloud.FloatingIP
	for i := range fips {
		if fips[i].IP.Equal(ip) {
			hetznerIP = fips[i]
		}
	}

	if hetznerIP == nil {
		return nil, fmt.Errorf("ip %s does not match any floating ip resource", ip.String())
	}

	return hetznerIP, nil
}

func (p *IPAssigner) findServer(node *corev1.Node) (*hcloud.Server, error) {
	server, _, err := p.hcloudCli.Server.GetByName(context.TODO(), node.Name)

	return server, err
}
