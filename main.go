package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

type ServerRequest struct {
	App string `json:"app"`
}

type KeyValue struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// cookie data from config file
type Cookie struct {
	AppName      string   `yaml:"appName"`
	Expiration   string   `yaml:"expiration"`
	Percent      float32  `yaml:"percent"`
	CookieName   string   `yaml:"cookieName"`
	IfSuccessful KeyValue `yaml:"ifSuccessful"`
	IfFail       KeyValue `yaml:"ifFail"`
}

type Config struct {
	Cookies []Cookie `yaml:"cookies"`
	Port    int      `yaml:"port"`
}

type CookieResponse struct {
	Key        string
	Value      string
	Expiration string
}

func ReadConfig() (Config, error) {
	// set path, use /workspaces/.. if unspecified
	configPath := os.Getenv("COOKIE_SETTER_CONFIG_PATH")
	if configPath == "" {
		configPath = "/workspaces/cookie-setter/cookie-setter.yaml"
	}
	config := Config{}
	configYaml, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(configYaml, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func CanaryResponse(cookie Cookie, randGenerator *rand.Rand, timeNow time.Time) (CookieResponse, error) {

	hours := strings.TrimSuffix(cookie.Expiration, "h")
	expiryHours, err := strconv.Atoi(hours)
	if err != nil {
		return CookieResponse{}, err
	}
	exp := timeNow.Add(time.Hour * time.Duration(expiryHours)).Format(time.RFC3339)

	randNum := randGenerator.Float32()
	if randNum <= cookie.Percent {
		return CookieResponse{
			Key:        cookie.IfSuccessful.Key,
			Value:      cookie.IfSuccessful.Value,
			Expiration: exp,
		}, nil
	}
	return CookieResponse{
		Key:        cookie.IfFail.Key,
		Value:      cookie.IfFail.Value,
		Expiration: exp,
	}, nil
}

func respondWithError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func respondJSON(w http.ResponseWriter, jdata []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(jdata)
}

func decodeAppName(req *http.Request) (string, error) {
	decoder := json.NewDecoder(req.Body)

	var s ServerRequest
	err := decoder.Decode(&s)
	if err != nil {
		return "", err
	}
	if s.App == "" {
		return "", errors.New("must specify key = 'app'")
	}
	return s.App, nil
}

func serveCookie(w http.ResponseWriter, req *http.Request) {
	appName, err := decodeAppName(req)
	if err != nil {
		log.Printf("Decode appName error = %v", err)
		respondWithError(w, http.StatusBadRequest, "Could not read PUT json. Make sure you PUT a json with {app: <app_name>}")
		return
	}
	config, err := ReadConfig()
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "Could not read config file!")
		return
	}
	for _, cookie := range config.Cookies {
		if cookie.AppName == appName {
			seed := rand.NewSource(time.Now().UnixNano())
			randGen := rand.New(seed)
			timeNow := time.Now()
			cookieResponse, err := CanaryResponse(cookie, randGen, timeNow)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Canary error!")
			}

			cookieJson, err := json.Marshal(cookieResponse)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Could not read cookie data!")
				return
			}
			respondJSON(w, cookieJson)
			return
		}
	}
	errMsg := fmt.Sprintf("Could not application %v", appName)
	log.Println(errMsg)
	respondWithError(w, http.StatusInternalServerError, errMsg)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.Method, r.RequestURI, r.Body)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func main() {
	config, err := ReadConfig()
	if err != nil {
		log.Println(err)
	}
	router := mux.NewRouter()
	router.Use(loggingMiddleware)
	router.HandleFunc("/", serveCookie).Methods("PUT")
	host := ":" + strconv.Itoa(config.Port)
	log.Println(fmt.Sprintf("Starting cookie-setter server on %v", host))
	http.ListenAndServe(host, router)
}
