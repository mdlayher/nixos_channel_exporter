package channels_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/nixos_channel_exporter/internal/channels"
)

func TestParseData(t *testing.T) {
	tests := []struct {
		name string
		s    string
		d    *channels.Data
		ok   bool
	}{
		{
			name: "bad JSON",
			s:    "xxx",
		},
		{
			name: "empty channel name",
			s:    `{"channels":{"":{}}}`,
		},
		{
			name: "empty channel job",
			s:    `{"channels":{"foo":{}}}`,
		},
		{
			name: "ok",
			s: `
{
  "channels": {
    "nixos-unstable": {
      "job": "nixos/trunk-combined/tested",
      "current": true
    },
    "nixos-unstable-small": {
      "job": "nixos/unstable-small/tested",
      "current": true
    },
    "nixpkgs-unstable": {
      "job": "nixpkgs/trunk/unstable",
      "current": false
    }
  }
}
`,
			d: &channels.Data{
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
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := channels.ParseData(strings.NewReader(tt.s))
			if tt.ok && err != nil {
				t.Fatalf("failed to parse data: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("expected an error, but none occurred")
			}
			if err != nil {
				t.Logf("err: %v", err)
				return
			}

			if diff := cmp.Diff(tt.d, d); diff != "" {
				t.Fatalf("unexpected Data (-want +got):\n%s", diff)
			}
		})
	}
}
