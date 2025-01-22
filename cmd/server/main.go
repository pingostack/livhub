package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pingostack/livhub/pkg/logger"
	"github.com/pingostack/livhub/pkg/plugo"
	_ "github.com/pingostack/livhub/servers"
)

var (
	// These variables are set during build time using -ldflags
	Version   = "development"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	var (
		configFile  string
		showVersion bool
	)

	flag.StringVar(&configFile, "c", "config.yaml", "path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.Parse()

	if showVersion {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}

	// 创建一个带取消的context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		fmt.Printf("Received signal %v, initiating shutdown...\n", sig)
		cancel()
	}()

	// Create file provider for configuration
	provider, err := plugo.NewFileProvider(configFile)
	if err != nil {
		log.Fatalf("Failed to create config provider: %v", err)
	}

	// Set config provider
	if err := plugo.SetConfigProvider(provider); err != nil {
		log.Fatalf("Failed to set config provider: %v", err)
	}

	// Start all plugins
	if err := plugo.Start(); err != nil {
		log.Fatalf("Failed to start plugins: %v", err)
	}

	logger.Infof("Livhub server started. Version: %s, Git Commit: %s, Build Time: %s", Version, GitCommit, BuildTime)
	// Wait for termination signal
	<-ctx.Done()

	logger.Info("Livhub server shutting down")

	// Clean up
	plugo.Close()

	logger.Info("Livhub server stopped")
}
