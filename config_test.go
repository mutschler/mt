package main

import (
	"os"
	"testing"
)

func TestSaveConfig(t *testing.T) {
	currentDirectory, _ := os.Getwd()
	got := saveConfig(currentDirectory)
	if got != nil {
		t.Errorf("got %q, wanted nil", got)
	}
}
