package main

import "errors"

var (
	// ErrCannotSaveConfigFile occurs when a configuration file cannot be opened.
	ErrCannotSaveConfigFile = errors.New("cannot save configuration file")
)
