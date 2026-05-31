package main

import "testing"

func TestEnvPort(t *testing.T) {
	if envPort("ACCOUNT_SERVICE_HTTP_PORT", 8081) != 8081 {
		t.Fatal("default port")
	}
}
