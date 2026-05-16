package main

import (
	"flag"
	"log"
	"time"

	"github.com/soyunomas/foldryn/internal/config"
	"github.com/soyunomas/foldryn/internal/database"
	"github.com/soyunomas/foldryn/internal/notifier"
	"github.com/soyunomas/foldryn/internal/organizer"
	"github.com/soyunomas/foldryn/internal/rules"
	"github.com/soyunomas/foldryn/internal/watcher"
)

const version = "0.3.0"

func main() {
	showVersion := flag.Bool("version", false, "print version")
	configPath := flag.String("config", "~/.config/foldryn/config.toml", "path to TOML config")
	flag.Parse()
	if *showVersion {
		log.Printf("Foldryn %s", version)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Open(config.Expand(cfg.App.DatabasePath))
	if err != nil {
		log.Fatalf("history: %v", err)
	}
	defer db.Close()

	compiled, err := rules.Compile(cfg.Rules)
	if err != nil {
		log.Fatalf("rules: %v", err)
	}

	org := &organizer.Organizer{
		Rules:    compiled,
		DryRun:   cfg.App.DryRun,
		DB:       db,
		Notifier: notifier.Notifier{Enabled: cfg.App.Notifications},
	}

	delay := time.Duration(cfg.App.SettleDelayMS) * time.Millisecond
	log.Printf("Foldryn %s starting dry_run=%v rules=%d settle_delay=%s", version, cfg.App.DryRun, len(compiled), delay)
	w := watcher.New(cfg.Watch, delay, org.Handle)
	if err := w.Run(); err != nil {
		log.Fatal(err)
	}
}
