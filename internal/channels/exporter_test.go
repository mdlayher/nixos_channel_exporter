package channels_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mdlayher/nixos_channel_exporter/internal/channels"
	"github.com/mdlayher/promtest"
)

func TestExporter(t *testing.T) {
	tests := []struct {
		name    string
		d       channels.Data
		handler http.HandlerFunc
		ok      bool
		metrics []string
	}{
		{
			name: "bad HTTP status",
			d: channels.Data{
				Channels: map[string]channels.Channel{
					"nixos-unstable": {
						Job:     "nixos/trunk-combined/tested",
						Current: true,
					},
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
		},
		{
			name: "no data",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("should never be called")
			},
			// Technically OK due to no inputs.
			ok: true,
		},
		{
			name: "multiple channels",
			d: channels.Data{
				Channels: map[string]channels.Channel{
					"nixos-unstable": {
						Job:     "nixos/trunk-combined/tested",
						Current: true,
					},
					"nixos-unstable-small": {
						Job:     "nixos/unstable-small/tested",
						Current: true,
					},
					"nixpkgs-unstable": {
						Job:     "nixpkgs/trunk/unstable",
						Current: false,
					},
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Ensure the client passes a sanely-formatted URL and return
				// something that looks like a git commit hash for the revision.
				var b byte
				switch r.URL.Path {
				case "/channels/nixos-unstable/git-revision":
					b = 0x11
				case "/channels/nixos-unstable-small/git-revision":
					b = 0x22
				case "/channels/nixpkgs-unstable/git-revision":
					b = 0x33
				default:
					panicf("unexpected URL path: %q", r.URL.Path)
				}

				s := hex.EncodeToString(bytes.Repeat([]byte{b}, 20))
				_, _ = io.WriteString(w, s)
			},
			ok: true,
			metrics: []string{
				`channel_revision{channel="nixos-unstable",revision="1111111111111111111111111111111111111111"} 1`,
				`channel_revision{channel="nixos-unstable-small",revision="2222222222222222222222222222222222222222"} 1`,
				`channel_revision{channel="nixpkgs-unstable",revision="3333333333333333333333333333333333333333"} 1`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up a fake nixos.org which returns channel metadata in the
			// same format as the real thing.
			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			e, err := channels.NewExporter(tt.d, srv.URL, nil)
			if err != nil {
				t.Fatalf("failed to create exporter: %v", err)
			}

			// Gather metrics and verify them for correctness.
			b := promtest.Collect(t, e)

			if !tt.ok {
				// Collection should have failed with some kind of error output
				// from the Prometheus client library.
				if !strings.Contains(string(b), "error collecting metric") {
					t.Fatal("expected Prometheus client library error, but did not find one")
				}

				return
			}

			// Collection succeeded, verify the correctness of the output
			// metrics.
			if !promtest.Lint(t, b) {
				t.Fatalf("failed to lint metrics: %v", err)
			}

			if !promtest.Match(t, b, tt.metrics) {
				t.Fatal("failed to match expected metrics")
			}
		})
	}
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
