package config

type Hetzner struct {
	Token string
}

type Handler struct {
	Hetzner Hetzner
}

type Config struct {
	Handler Handler
}
