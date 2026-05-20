package web

import (
	"net/netip"
	"testing"
)

func TestDefaultBindTargets(t *testing.T) {
	targets := DefaultBindTargets(23832, []netip.Addr{
		netip.MustParseAddr("100.64.1.2"),
		netip.MustParseAddr("100.127.255.254"),
		netip.MustParseAddr("100.128.0.1"),
		netip.MustParseAddr("8.8.8.8"),
	})
	got := map[string]bool{}
	for _, target := range targets {
		got[target.Addr.String()] = true
	}
	if !got["127.0.0.1"] || !got["100.64.1.2"] || !got["100.127.255.254"] {
		t.Fatalf("expected 127 and 100.64/10 targets, got %#v", got)
	}
	if got["100.128.0.1"] || got["8.8.8.8"] {
		t.Fatalf("unexpected public targets: %#v", got)
	}
}

func TestRemoteAllowed(t *testing.T) {
	prefixes, err := ParseAllowPrefixes([]string{"192.168.1.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	allowed := []string{"127.0.0.1:1", "100.64.0.1:1", "100.127.255.254:1", "192.168.1.5:1"}
	for _, remote := range allowed {
		if !RemoteAllowed(remote, prefixes) {
			t.Fatalf("expected %s to be allowed", remote)
		}
	}
	rejected := []string{"100.128.0.1:1", "8.8.8.8:1", "192.168.2.5:1"}
	for _, remote := range rejected {
		if RemoteAllowed(remote, prefixes) {
			t.Fatalf("expected %s to be rejected", remote)
		}
	}
}

func TestResolveBindTargetsRejectsWildcardWithoutAllow(t *testing.T) {
	if _, err := ResolveBindTargets("0.0.0.0:23832", 23832, nil, nil); err == nil {
		t.Fatal("expected wildcard bind without allow to fail")
	}
	targets, err := ResolveBindTargets("0.0.0.0:23832", 23832, nil, []string{"192.168.1.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].ListenAddress() != "0.0.0.0:23832" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}
