package main

import (
	"testing"

	"github.com/msfidelis/sales-rest-api/pkg/configuration"
)

func TestPkgConfigurationLoad(t *testing.T) {

	t.Run("Load JSON File", func(t *testing.T) {
		configs := configuration.Load()
		got := configs.Version
		want := "v1"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})

	t.Run("Assert Correct Env File", func(t *testing.T) {
		configs := configuration.Load()
		got := configs.Env
		want := "prod"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})

}
