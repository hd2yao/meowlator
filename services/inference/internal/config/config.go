package config

import (
	"os"
)

type Config struct {
	Addr string
}

func Load() Config {
	addr := os.Getenv("INFERENCE_ADDR")
	if addr == "" {
		addr = ":8081"
	}
	return Config{Addr: addr}
}
