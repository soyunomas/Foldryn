package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/soyunomas/foldryn/internal/config"
	"github.com/soyunomas/foldryn/internal/database"
)

const version = "0.3.0"

func main() {
	configPath := flag.String("config", "~/.config/foldryn/config.toml", "path to TOML config")
	limit := flag.Int("n", 20, "number of history entries")
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()
	if *showVersion {
		fmt.Printf("Foldryn ctl %s\n", version)
		return
	}
	cmd := "history"
	if flag.NArg() > 0 {
		cmd = flag.Arg(0)
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
	switch cmd {
	case "history":
		events, err := db.Recent(*limit)
		if err != nil {
			log.Fatal(err)
		}
		for _, e := range events {
			fmt.Println(e.String())
		}
	default:
		log.Fatalf("unknown command %q; supported: history", cmd)
	}
}
