package main

import (
	"fmt"
	"os"
	"testing"
)

func TestSaveConfig(t *testing.T) {
	testFile, _ := os.CreateTemp("", "saveTest.json")
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			fmt.Println("could not delete saveTest.json")
		}
	}(testFile.Name())

	got := saveConfig(testFile.Name())
	if got != nil {
		t.Errorf("got %q, wanted nil", got)
	}
}
