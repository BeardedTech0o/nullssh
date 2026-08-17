package sshexec

import (
	"testing"

	"github.com/BeardedTech0o/tether/internal/store"
)

func TestValidateAcceptsOrdinaryConnections(t *testing.T) {
	cases := []store.Connection{
		{Host: "example.com", User: "deploy"},
		{Host: "203.0.113.10", User: "root"},
		{Host: "my-server.internal", User: "user_name"},
	}
	for _, c := range cases {
		if err := Validate(c); err != nil {
			t.Errorf("Validate(%+v) = %v, want nil", c, err)
		}
	}
}

func TestValidateRejectsArgumentInjection(t *testing.T) {
	cases := []store.Connection{
		// The exact shape of attack this guards against: smuggling an
		// extra ssh flag into the Host field via an embedded space.
		{Host: "example.com -oProxyCommand=calc.exe", User: "deploy"},
		{Host: "example.com", User: "deploy -oProxyCommand=calc.exe"},
		{Host: "-oProxyCommand=calc.exe", User: "deploy"},
		{Host: "example.com\t-oProxyCommand=x", User: "deploy"},
		{Host: "example.com", User: "-root"},
	}
	for _, c := range cases {
		if err := Validate(c); err == nil {
			t.Errorf("Validate(%+v) succeeded, want error", c)
		}
	}
}
