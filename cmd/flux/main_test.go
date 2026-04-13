package main

import (
	"testing"
)

func TestRun_MissingFile(t *testing.T) {
	err := run([]string{"non_existent.go"})
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	if err.Error() != "file non_existent.go not found" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRun_Version(t *testing.T) {
	// Testing version shouldn't error
	err := run([]string{"--version"})
	if err != nil {
		t.Errorf("expected no error for version flag, got %v", err)
	}
}

func TestRun_Help(t *testing.T) {
	err := run([]string{"help"})
	if err != nil {
		t.Errorf("expected no error for help command, got %v", err)
	}
}

func TestRun_RunCommand(t *testing.T) {
	// Should fail if run is used without a file
	err := run([]string{"run"})
	if err == nil {
		t.Fatal("expected error for run without args, got nil")
	}
}
