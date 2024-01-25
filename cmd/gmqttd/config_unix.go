//go:build !windows
// +build !windows

package main

var (
	DefaultConfigDir = "/gmqttd"
)

func getDefaultConfigDir() (string, error) {
	return DefaultConfigDir, nil
}
