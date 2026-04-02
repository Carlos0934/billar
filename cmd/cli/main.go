package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Carlos0934/billar/internal/app"
	connectorcli "github.com/Carlos0934/billar/internal/connectors/cli"
	"github.com/Carlos0934/billar/internal/infra/config"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	cmd := connectorcli.NewCommand(app.NewHealthService(cfg.AppName), cfg.ColorEnabled)

	if err := cmd.Run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
