package channels

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

var _ prometheus.Collector = &Exporter{}

// An Exporter is a prometheus.Collector for NixOS channel metadata.
type Exporter struct {
	// Atomics must come first.
	failures uint32

	ChannelRevision        *prometheus.Desc
	ChannelUpdateTime      *prometheus.Desc
	ChannelCurrent         *prometheus.Desc
	ChannelRequestFailures *prometheus.Desc

	// TODO: histogram of outgoing HTTP request times.

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

		// TODO: consider adding unit _seconds.

		ChannelUpdateTime: prometheus.NewDesc(
			"channel_update_time",
			"The UNIX timestamp of when a channel was last updated.",
			[]string{"channel"},
			nil,
		),

		ChannelCurrent: prometheus.NewDesc(
			"channel_current",
			"Reports whether or not a channel is expected to be current.",
			[]string{"channel"},
			nil,
		),

		ChannelRequestFailures: prometheus.NewDesc(
			"channel_request_failures_total",
			"Number of channel status requests which have failed",
			nil,
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
		e.ChannelUpdateTime,
		e.ChannelCurrent,
		e.ChannelRequestFailures,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// Set a sane upper bound on the time that metrics collection and HTTP
	// requests can take.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// If any of the goroutines return an error, that error will cancel this
	// context, causing the other goroutines to bail out of their HTTP requests
	// so we can more quickly report an error to Prometheus.
	eg, ctx := errgroup.WithContext(ctx)

	for channel, meta := range e.data.Channels {
		// Copies of range variables to pass into goroutines.
		channel := channel
		meta := meta

		eg.Go(func() error {
			if err := e.collect(ctx, ch, channel, meta); err != nil {
				// Track the number of failures so we can also export that as a
				// metric.
				atomic.AddUint32(&e.failures, 1)
				return fmt.Errorf("failed to fetch channel %q: %v", channel, err)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		log.Printf("error: %v", err)
		ch <- prometheus.NewInvalidMetric(e.ChannelRevision, err)
	}

	ch <- prometheus.MustNewConstMetric(
		e.ChannelRequestFailures,
		prometheus.CounterValue,
		float64(atomic.LoadUint32(&e.failures)),
	)
}

func (e *Exporter) collect(ctx context.Context, ch chan<- prometheus.Metric, channel string, meta Channel) error {
	// Make a copy to avoid mutating e.base.
	addr := e.base
	addr.Path = fmt.Sprintf("/channels/%s/git-revision", channel)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	res, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to HTTP GET channel revision: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response: HTTP %d %s", res.StatusCode, res.Status)
	}

	// Check for a Last-Modified header and parse it if present.
	var modified time.Time
	if h := res.Header.Get("Last-Modified"); h != "" {
		m, err := http.ParseTime(h)
		if err != nil {
			return fmt.Errorf("malformed Last-Modified header: %v", err)
		}

		modified = m
	}

	// The HTTP body should just contain a git commit hash, so assume it will
	// not be very large in size.
	rev, err := ioutil.ReadAll(io.LimitReader(res.Body, 128))
	if err != nil {
		return fmt.Errorf("failed to read HTTP channel revision body: %v", err)
	}

	ch <- prometheus.MustNewConstMetric(
		e.ChannelRevision,
		prometheus.GaugeValue,
		1,
		channel, string(rev),
	)

	if !modified.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			e.ChannelUpdateTime,
			prometheus.GaugeValue,
			float64(modified.Unix()),
			channel,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		e.ChannelCurrent,
		prometheus.GaugeValue,
		boolFloat(meta.Current),
		channel,
	)

	return nil
}

func boolFloat(b bool) float64 {
	if b {
		return 1.0
	}

	return 0.0
}
