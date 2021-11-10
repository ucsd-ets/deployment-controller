package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadConfig(t *testing.T) {
	config, err := ReadConfig()
	shouldEqual := Config{
		Apps: []App{
			App{
				Name: "jupyterhub",
				Mode: "ab",
				CookieInfo: Cookie{
					Expiration:    "48h",
					CanaryPercent: .90,
					IfSuccessful: KeyValue{
						Key:   "a",
						Value: "a",
					},
				},
				View: View{
					ShowSuccess: true,
					ShowFail:    true,
				},
				Logging: Logging{
					Disable: false,
				},
			},
		},
		Port: 8080,
	}

	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(config.Apps) != 1 {
		t.Fatalf("Failed cookie length = %v", config.Apps)
	}

	if !cmp.Equal(config, shouldEqual) {
		t.Fatalf("Failed equality to test = %v %v", config, shouldEqual)
	}

}
