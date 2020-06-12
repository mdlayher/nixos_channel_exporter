package channels

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Data is the top-level wrapper for NixOS channel metadata.
type Data struct {
	Channels map[string]Channel `json:"channels"`
}

// A Channel describes an individual NixOS channel.
type Channel struct {
	Job     string `json:"job"`
	Current bool   `json:"current"`
}

// ParseData parses Data from an input JSON stream.
func ParseData(r io.Reader) (*Data, error) {
	var d Data
	if err := json.NewDecoder(r).Decode(&d); err != nil {
		return nil, err
	}

	// Do some basic sanity checks to ensure the input JSON has metadata we will
	// need to proceed later.
	for k, v := range d.Channels {
		if k == "" {
			return nil, errors.New("empty key in channels")
		}

		if v.Job == "" {
			return nil, fmt.Errorf("channel %q has empty job value", k)
		}
	}

	return &d, nil
}
