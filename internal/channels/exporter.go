package channels

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &Exporter{}

// An Exporter is a prometheus.Collector for NixOS channel metadata.
type Exporter struct {
	ChannelRevision *prometheus.Desc

	base   url.URL
	data   Data
	client *http.Client
}

// NewExporter constructs a prometheus.Collector which gathers NixOS channel
// metadata about the channels defined in data, from the HTTP server specified
// by baseURL.
//
// If baseURL is empty, a sane default is used. If client is nil, a default HTTP
// client with timeouts is used.
func NewExporter(data Data, baseURL string, client *http.Client) (prometheus.Collector, error) {
	if baseURL == "" {
		baseURL = "https://nixos.org/"
	}

	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Exporter{
		// TODO: consider namespacing metrics with nixos_.

		ChannelRevision: prometheus.NewDesc(
			"channel_revision",
			"Synthetic information metric with labels for the current revision of a channel.",
			[]string{"channel", "revision"},
			nil,
		),

		base:   *base,
		data:   data,
		client: client,
	}, nil
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		e.ChannelRevision,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for channel, meta := range e.data.Channels {
		if err := e.collect(ch, channel, meta); err != nil {
			err := fmt.Errorf("failed to fetch channel %q: %v", channel, err)
			log.Printf("error: %v", err)
			ch <- prometheus.NewInvalidMetric(e.ChannelRevision, err)
			continue
		}
	}
}

func (e *Exporter) collect(ch chan<- prometheus.Metric, channel string, meta Channel) error {
	// Make a copy to avoid mutating e.base.
	addr := e.base
	addr.Path = fmt.Sprintf("/channels/%s/git-revision", channel)

	res, err := e.client.Get(addr.String())
	if err != nil {
		return fmt.Errorf("failed to HTTP GET channel revision: %v", err)
	}
	defer res.Body.Close()

	rev, err := ioutil.ReadAll(io.LimitReader(res.Body, 128))
	if err != nil {
		return fmt.Errorf("failed to read HTTP channel revision body: %v", err)
	}

	log.Println(res)
	log.Println(string(rev))

	ch <- prometheus.MustNewConstMetric(
		e.ChannelRevision,
		prometheus.GaugeValue,
		1,
		channel, string(rev),
	)

	return nil
}
