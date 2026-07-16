package main

import (
	"testing"
	"time"
)

func TestGetenv(t *testing.T) {
	t.Setenv("MIRA_TEST_GETENV", "")
	if got := getenv("MIRA_TEST_GETENV", "fallback"); got != "fallback" {
		t.Errorf("absent -> %q, attendu fallback", got)
	}

	t.Setenv("MIRA_TEST_GETENV", "set")
	if got := getenv("MIRA_TEST_GETENV", "fallback"); got != "set" {
		t.Errorf("défini -> %q, attendu set", got)
	}
}

func TestGetenvInt(t *testing.T) {
	tests := []struct {
		name, value string
		fallback    int
		want        int
	}{
		{"absent -> fallback", "", 4, 4},
		{"valide -> parsé", "10", 4, 10},
		{"invalide -> fallback", "abc", 4, 4},
		{"négatif -> parsé tel quel", "-1", 4, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MIRA_TEST_GETENVINT", tt.value)
			if got := getenvInt("MIRA_TEST_GETENVINT", tt.fallback); got != tt.want {
				t.Errorf("getenvInt(%q, %d) = %d, attendu %d", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestGetenvDuration(t *testing.T) {
	tests := []struct {
		name, value string
		fallback    time.Duration
		want        time.Duration
	}{
		{"absent -> fallback", "", 30 * time.Second, 30 * time.Second},
		{"valide -> parsé", "5s", 30 * time.Second, 5 * time.Second},
		{"invalide -> fallback", "pas-une-durée", 30 * time.Second, 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MIRA_TEST_GETENVDURATION", tt.value)
			if got := getenvDuration("MIRA_TEST_GETENVDURATION", tt.fallback); got != tt.want {
				t.Errorf("getenvDuration(%q, %v) = %v, attendu %v", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}
