package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var cookie = Cookie{
	AppName:      "jupyterhub",
	Expiration:   "48h",
	Percent:      .90,
	IfSuccessful: KeyValue{Key: "a", Value: "a"},
	IfFail:       KeyValue{Key: "b", Value: "b"},
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
    rr := httptest.NewRecorder()
    a.Router.ServeHTTP(rr, req)

    return rr
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

func TestIndex(t *testing.T) {
    req, _ := http.NewRequest("GET", "/products", nil)
    response := executeRequest(req)

    checkResponseCode(t, http.StatusOK, response.Code)

    if body := response.Body.String(); body != "[]" {
        t.Errorf("Expected an empty array. Got %s", body)
    }
}