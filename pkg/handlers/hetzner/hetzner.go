package hetzner

import (
	"fmt"

	"github.com/apricote/hcloud-floating-ip-controller/config"
)

type Hetzner struct {
	Token string
}

// Init initializes handler configuration
// Do nothing for default handler
func (h *Hetzner) Init(c *config.Config) error {
	token := c.Handler.Hetzner.Token

	h.Token = token

	return checkMissingHetznerVars(h)
}

func (h *Hetzner) ObjectCreated(obj interface{}) {
	verifyIPsAreAssigned(h)
}

func (h *Hetzner) ObjectDeleted(obj interface{}) {
	verifyIPsAreAssigned(h)
}

func (h *Hetzner) ObjectUpdated(oldObj, newObj interface{}) {
	verifyIPsAreAssigned(h)
}

func checkMissingHetznerVars(h *Hetzner) error {
	if h.Token == "" {
		return fmt.Errorf("Missing Hetzner Cloud Token")
	}

	return nil
}

func verifyIPsAreAssigned(h *Hetzner) {
	fmt.Printf("IP Assignment Triggered")
}
