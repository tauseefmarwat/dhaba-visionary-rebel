package main

import "testing"

func TestEnvFallback(t *testing.T) {
	if env("DHABA_TEST_MISSING", "fallback") != "fallback" {
		t.Fatal("fallback not returned")
	}
}
