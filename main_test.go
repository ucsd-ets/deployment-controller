package main

import (
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func setUp() App {
	return App{
		Name:    "jupyterhub",
		Disable: false,
		CookieInfo: Cookie{
			Expiration:    "48h",
			CanaryPercent: .90,
			IfSuccessful: KeyValue{
				Key:   "a",
				Value: "a",
			},
			IfFail: KeyValue{
				Key:   "b",
				Value: "b",
			},
		},
		View: View{
			ShowSuccess: true,
			ShowFail:    true,
		},
		Logging: Logging{
			Disable: false,
		},
	}
}

func cleanUp(t *testing.T) {
	app := setUp()
	err := UpdateConfig(app)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadConfig(t *testing.T) {
	app := setUp()
	config, err := ReadConfig()
	shouldEqual := Config{
		Apps: []App{
			app,
		},
		Port: 8080,
	}

	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(config.Apps) != 1 {
		t.Fatalf("Failed cookie length = %v", config.Apps)
	}

	log.Println(config)

	if !cmp.Equal(config, shouldEqual) {
		t.Fatalf("Failed equality to test = %v %v", config, shouldEqual)
	}
}

func TestUpdateConfig(t *testing.T) {
	app := setUp()
	app.Logging.Disable = true
	err := UpdateConfig(app)
	if err != nil {
		t.Fatal(err)
	}

	savedConfig, err := ReadConfig()
	if err != nil {
		t.Fatal(err)
	}

	defer cleanUp(t)

	if !cmp.Equal(savedConfig.Apps[0], app) {
		t.Fatalf("Failed equality test after update = %v %v", savedConfig.Apps[0], app)
	}
}
