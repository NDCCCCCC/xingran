package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/xingran-next/rpa-worker/internal/config"
    "github.com/xingran-next/rpa-worker/internal/worker"
)

var (
    configFile = flag.String("config", "configs/config.yaml", "Configuration file path")
    version   = "1.0.0"
)

func main() {
    flag.Parse()

    // Print version
    if len(flag.Args()) > 0 && flag.Arg(0) == "version" {
        fmt.Printf("RPA Worker version %s\n", version)
        os.Exit(0)
    }

    // Load configuration
    cfg, err := config.Load(*configFile)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
        os.Exit(1)
    }

    // Create Worker
    w, err := worker.New(cfg)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create worker: %v\n", err)
        os.Exit(1)
    }

    // Start Worker
    if err := w.Start(); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to start worker: %v\n", err)
        os.Exit(1)
    }

    // Wait for signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    <-sigCh

    // Graceful shutdown
    w.Shutdown(context.Background())
}
