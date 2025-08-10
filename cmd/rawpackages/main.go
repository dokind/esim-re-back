package main

import (
	"encoding/json"
	"esim-platform/internal/config"
	"esim-platform/internal/services"
	"flag"
	"fmt"
	"log"
)

func main() {
	sku := flag.String("sku", "155", "SKU ID to fetch packages for")
	flag.Parse()

	cfg := config.Load()
	rw := services.NewRoamWiFiService(cfg.RoamWiFi)
	raw, err := rw.GetPackagesRaw(*sku)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	b, _ := json.MarshalIndent(raw, "", "  ")
	fmt.Println(string(b))
}
