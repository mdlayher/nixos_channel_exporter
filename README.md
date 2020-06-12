# nixos_channel_exporter [![Linux Test Status](https://github.com/mdlayher/nixos_channel_exporter/workflows/Linux%20Test/badge.svg)](https://github.com/mdlayher/nixos_channel_exporter/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/mdlayher/nixos_channel_exporter)](https://goreportcard.com/report/github.com/mdlayher/nixos_channel_exporter)

Command `nixos_channel_exporter` implements a Prometheus exporter for NixOS
channel status metadata. MIT Licensed.

This code is a candidate to replace the existing [channel-exporter.py](https://github.com/NixOS/nixos-org-configurations/blob/0486753b1ce33b5ab447840678c4fe7a662bd6c1/delft/eris/channel-exporter.py)
script with a Go exporter which can fetch data concurrently and export more
fine-grained metrics.
