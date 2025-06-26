package cmd

import (
	"testing"
)

func TestServerCommandDefined(t *testing.T) {
	if serverCmd == nil {
		t.Fatal("serverCmd should be defined")
	}
	if serverCmd.Use != "server" {
		t.Errorf("expected command use 'server', got %s", serverCmd.Use)
	}
	portFlag := serverCmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Fatal("expected 'port' flag to be defined")
	}
	if portFlag.Value.Type() != "string" {
		t.Errorf("expected 'port' flag to be string type, got %s", portFlag.Value.Type())
	}
}
