package main

import (
	"math/rand"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var cookie = Cookie{
	AppName:      "jupyterhub",
	Expiration:   "48h",
	Percent:      .90,
	IfSuccessful: KeyValue{Key: "a", Value: "a"},
	IfFail:       KeyValue{Key: "b", Value: "b"},
}

func TestReadConfig(t *testing.T) {
	config, err := ReadConfig()

	cookies := []Cookie{
		cookie,
	}
	shouldEqual := Config{
		Cookies: cookies,
		Port:    8080,
	}

	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(config.Cookies) != 1 {
		t.Fatalf("Failed cookie length = %v", config.Cookies)
	}

	if !cmp.Equal(config, shouldEqual) {
		t.Fatalf("Failed equality to test = %v %v", config, shouldEqual)
	}

}

func TestCanarayResponse(t *testing.T) {
	seed := rand.NewSource(1)
	r := rand.New(seed)
	ti := time.Date(2021, 1, 1, 1, 1, 1, 1, time.UTC)
	got, err := CanaryResponse(cookie, r, ti)
	if err != nil {
		t.Fatalf("Failed CanaryResponse %v", err)
	}

	// the first response should be pass
	success := CookieResponse{
		Key:        "a",
		Value:      "a",
		Expiration: ti.Add(time.Hour * 48).Format(time.RFC3339),
		AllCookies: [2]map[string]string{
			{"Key": "a", "Value": "a"},
			{"Key": "b", "Value": "b"},
		},
	}

	if !cmp.Equal(got, success) {
		t.Fatalf("Failed success = %v %v", got, success)
	}

	// The next response should be a fail
	got, err = CanaryResponse(cookie, r, ti)
	if err != nil {
		t.Fatalf("Failed CanaryResponse %v", err)
	}

	success.Key = "b"
	success.Value = "b"
	if !cmp.Equal(got, success) {
		t.Fatalf("Failed fail = %v %v", got, success)
	}
}
