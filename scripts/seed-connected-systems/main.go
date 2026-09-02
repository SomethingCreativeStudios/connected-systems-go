// seed-connected-systems creates realistic, linked OGC API - Connected Systems
// Part 1 and Part 2 data through the public HTTP API.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to the YAML seed configuration")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	// Keep the namespace as the stable ownership tag while assigning a fresh
	// run discriminator unless the config explicitly supplies one.
	cfg.RunID = cfg.EffectiveRunID(time.Now())

	seed := uint64(time.Now().UnixNano())
	if cfg.RandomSeed != nil {
		seed = uint64(*cfg.RandomSeed)
	}
	log.Printf("mode=%s endpoint=%s namespace=%s run_id=%s random_seed=%d", cfg.Mode, cfg.Endpoint, cfg.Namespace, cfg.RunID, seed)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := NewAPIClient(cfg)
	if err != nil {
		log.Fatalf("create API client: %v", err)
	}
	if err := client.Preflight(ctx); err != nil {
		log.Fatalf("target API preflight failed: %v", err)
	}

	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	var report Report
	if cfg.Mode == "seed" {
		report, err = NewSeeder(cfg, client, rng).Run(ctx)
	} else {
		report, err = NewObserver(cfg, client, rng).Run(ctx)
	}
	if err != nil {
		log.Printf("run failed: %v", err)
	}
	fmt.Println(report.JSON())
	if err != nil {
		os.Exit(1)
	}
}
