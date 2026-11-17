package main

import (
	"auth_service/internal/config"
	httpServer "auth_service/internal/transport"
	"fmt"
)

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Errorf("failed to load configuration: %v", err))
	}

	server, err := httpServer.NewServer(cfg)
	if err != nil {
		panic(fmt.Errorf("failed to initialize server: %v", err))
	}

	server.Run()
}
