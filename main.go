package main

import (
	"log"
	"os"

	"fyne.io/fyne/v2/app"

	"watcher-client/api"
	"watcher-client/config"
	"watcher-client/ui"
)

const backendURL = "http://localhost:8080"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	cfg.BackendURL = backendURL

	if cfg.InstanceKey == "" || cfg.InstanceSecret == "" {
		if err := autoRegisterInstance(cfg); err != nil {
			log.Fatalf("auto-register failed: %v", err)
		}
		if err := config.Save(cfg); err != nil {
			log.Printf("config save error after register: %v", err)
		}
	}

	client := api.NewClient(cfg.BackendURL, cfg.InstanceKey, cfg.InstanceSecret)

	a := app.New()
	mw := ui.NewMainWindow(a, client)

	defer func() {
		if err := config.Save(cfg); err != nil {
			log.Printf("config save error: %v", err)
		}
	}()

	mw.Window.ShowAndRun()
}

func autoRegisterInstance(cfg *config.InstanceConfig) error {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "watcher-device"
	}

	log.Printf("No instance credentials found, registering new instance as %q...", hostname)

	key, secret, err := api.RegisterInstance(cfg.BackendURL, hostname)
	if err != nil {
		return err
	}

	cfg.InstanceKey = key
	cfg.InstanceSecret = secret

	log.Printf("Registered instance. Key: %s", key)
	return nil
}
