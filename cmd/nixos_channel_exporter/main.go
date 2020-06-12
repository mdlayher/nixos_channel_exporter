// Command nixos_channel_exporter implements a Prometheus exporter for NixOS
// channel status metadata. MIT Licensed.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mdlayher/nixos_channel_exporter/internal/channels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		addr   = flag.String("addr", ":9402", "address used to bind the Prometheus metrics HTTP server")
		config = flag.String("config", "channels.json", "path to the channels.json file derived from channels.nix")
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

	// Make Prometheus client aware of our collector.
	c, err := channels.NewExporter(*data, "", nil)
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(
		prometheus.NewBuildInfoCollector(),
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		c,
	)

	// Set up HTTP handler for metrics.
	mux := http.NewServeMux()
	mux.Handle("/", http.RedirectHandler("/metrics", http.StatusPermanentRedirect))
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// Start listening for HTTP connections.
	log.Printf("starting NixOS channel exporter on %q", *addr)
	server := http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("cannot start NixOS channel exporter: %s", err)
	}
}
