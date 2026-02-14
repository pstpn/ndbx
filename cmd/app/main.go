package main

import (
	"context"
	"log"

	"ndbx/config"
	"ndbx/internal/app"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("new config: %s", err.Error())
	}

	ctx := context.Background()
	if err = app.Run(ctx, cfg); err != nil {
		log.Fatalf("run app: %s", err.Error())
	}
}
