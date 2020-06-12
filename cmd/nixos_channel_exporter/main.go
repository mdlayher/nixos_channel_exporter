// Command nixos_channel_exporter implements a Prometheus exporter for NixOS
// channel status metadata. MIT Licensed.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/mdlayher/nixos_channel_exporter/internal/channels"
)

func main() {
	var (
		config = flag.String("c", "channels.json", "path to the channels.json file derived from channels.nix")
	)
	flag.Parse()

	f, err := os.Open(*config)
	if err != nil {
		log.Fatalf("failed to open: %v", err)
	}
	defer f.Close()

	data, err := channels.ParseData(f)
	if err != nil {
		log.Fatalf("failed to parse channels data: %v", err)
	}
	_ = f.Close()

	// TODO!
	log.Println(data)
}
