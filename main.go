package main

import (
	"github.com/apricote/hcloud-floating-ip-controller/config"
	"github.com/apricote/hcloud-floating-ip-controller/pkg/controller"
	"github.com/apricote/hcloud-floating-ip-controller/pkg/handlers"
)

func main() {
	controller.Start(config.Config{}, handlers.Map["hetzner"])
}
