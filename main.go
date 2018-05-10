package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/spotahome/kooper/client/crd"
	applogger "github.com/spotahome/kooper/log"
	apiextensionscli "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	floatingipk8scli "github.com/apricote/hcloud-floating-ip-operator/client/k8s/clientset/versioned"
	"github.com/apricote/hcloud-floating-ip-operator/config"
	"github.com/apricote/hcloud-floating-ip-operator/pkg/log"

	"github.com/apricote/hcloud-floating-ip-operator/pkg/operator"
)

// Main is the main program.
type Main struct {
	flags  *config.Flags
	config operator.Config
	logger log.Logger
}

// New returns the main application.
func New(logger log.Logger) *Main {
	f := config.NewFlags()
	return &Main{
		flags:  f,
		config: f.OperatorConfig(),
		logger: logger,
	}
}

// Run runs the app.
func (m *Main) Run(stopC <-chan struct{}) error {
	m.logger.Infof("initializing hcloud floating ip operator")

	// Get kubernetes rest client.
	fipCli, crdCli, k8sCli, err := m.getKubernetesClients()
	if err != nil {
		return err
	}

	hcloudCli := hcloud.NewClient(hcloud.WithToken(m.flags.HCloudToken))

	// Create the operator and run
	op, err := operator.New(m.config, fipCli, crdCli, k8sCli, hcloudCli, m.logger)
	if err != nil {
		return err
	}

	return op.Run(stopC)
}

// getKubernetesClients returns all the required clients to communicate with
// kubernetes cluster: CRD type client, pod terminator types client, kubernetes core types client.
func (m *Main) getKubernetesClients() (floatingipk8scli.Interface, crd.Interface, kubernetes.Interface, error) {
	var err error
	var cfg *rest.Config

	// If devel mode then use configuration flag path.
	if m.flags.Development {
		cfg, err = clientcmd.BuildConfigFromFlags("", m.flags.KubeConfig)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not load configuration: %s", err)
		}
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error loading kubernetes configuration inside cluster, check app is running outside kubernetes cluster or run in development mode: %s", err)
		}
	}

	// Create clients.
	k8sCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	// App CRD k8s types client.
	fipCli, err := floatingipk8scli.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	// CRD cli.
	aexCli, err := apiextensionscli.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	crdCli := crd.NewClient(aexCli, m.logger)

	return fipCli, crdCli, k8sCli, nil
}

func main() {
	logger := &applogger.Std{}

	stopC := make(chan struct{})
	finishC := make(chan error)
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, syscall.SIGTERM, syscall.SIGINT)
	m := New(logger)

	// Run in background the operator.
	go func() {
		finishC <- m.Run(stopC)
	}()

	select {
	case err := <-finishC:
		if err != nil {
			fmt.Fprintf(os.Stderr, "error running operator: %s", err)
			os.Exit(1)
		}
	case <-signalC:
		logger.Infof("Signal captured, exiting...")
	}
	close(stopC)
	time.Sleep(5 * time.Second)
}
