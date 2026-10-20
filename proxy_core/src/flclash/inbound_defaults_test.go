package main

import (
	"encoding/json"
	"net/netip"
	"testing"
)

func TestOlderSettingsRetainCoreInboundAccess(t *testing.T) {
	for _, document := range []string{`{}`, `{"lan-allowed-ips":null}`} {
		var settings struct {
			Allowed []netip.Prefix `json:"lan-allowed-ips"`
		}
		if err := json.Unmarshal([]byte(document), &settings); err != nil {
			t.Fatal(err)
		}
		allowed := defaultLanAllowedIPs(settings.Allowed)
		for _, client := range []string{"127.0.0.1", "::1", "192.0.2.10", "2001:db8::10"} {
			found := false
			for _, prefix := range allowed {
				found = found || prefix.Contains(netip.MustParseAddr(client))
			}
			if !found {
				t.Fatalf("legacy settings %s reject %s", document, client)
			}
		}
	}
}

func TestExplicitInboundPolicyIsPreserved(t *testing.T) {
	for _, document := range []string{`{"lan-allowed-ips":[]}`, `{"lan-allowed-ips":["127.0.0.0/8","::1/128"]}`} {
		var settings struct {
			Allowed []netip.Prefix `json:"lan-allowed-ips"`
		}
		if err := json.Unmarshal([]byte(document), &settings); err != nil {
			t.Fatal(err)
		}
		got := defaultLanAllowedIPs(settings.Allowed)
		if got == nil || len(got) != len(settings.Allowed) {
			t.Fatalf("explicit policy changed: %s => %v", document, got)
		}
		for i := range got {
			if got[i] != settings.Allowed[i] {
				t.Fatalf("explicit prefix changed: %s => %v", document, got)
			}
		}
	}
}
