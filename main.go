package main

import (
	"encoding/json"
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

type KeyValue struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// cookie data from config file
type Cookie struct {
	Expiration    string   `yaml:"expiration"`
	CanaryPercent float32  `yaml:"canaryPercent"`
	IfSuccessful  KeyValue `yaml:"ifSuccessful"`
	IfFail        KeyValue `yaml:"ifFail"`
}

type View struct {
	ShowSuccess bool `yaml:"showSuccess"`
	ShowFail    bool `yaml:"showFail"`
}

type App struct {
	Name       string `yaml:"appName"`
	Disable    bool   `yaml:"disable"`
	CookieInfo Cookie `yaml:"cookieInfo"`
	View       View   `yaml:"view"`
}

type Config struct {
	Apps []App `yaml:"apps"`
	Port int   `yaml:"port"`
}

type CookieResponse struct {
	Key           string
	Value         string
	Expiration    string
	AllCookies    [2]map[string]string
	CanaryPercent float32
	Disable       bool
}

func ReadConfig() (Config, error) {
	// set path, use /workspaces/.. if unspecified
	configPath := os.Getenv("APP_CONFIG_PATH")
	if configPath == "" {
		configPath = "/workspaces/deployment-controller/deployment-controller.yaml"
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

func GetCookieResponse(cookie Cookie, timeNow time.Time, successCookieType bool) (CookieResponse, error) {

	hours := strings.TrimSuffix(cookie.Expiration, "h")
	expiryHours, err := strconv.Atoi(hours)
	if err != nil {
		return CookieResponse{}, err
	}
	exp := timeNow.Add(time.Hour * time.Duration(expiryHours)).Format(time.RFC3339)

	allCookies := [2]map[string]string{
		{"Key": cookie.IfSuccessful.Key, "Value": cookie.IfSuccessful.Value},
		{"Key": cookie.IfFail.Key, "Value": cookie.IfFail.Value},
	}

	responseCookie := CookieResponse{
		Key:           cookie.IfSuccessful.Key,
		Value:         cookie.IfSuccessful.Value,
		Expiration:    exp,
		AllCookies:    allCookies,
		CanaryPercent: cookie.CanaryPercent,
	}

	if successCookieType {
		return responseCookie, nil
	}

	responseCookie.Key = cookie.IfFail.Key
	responseCookie.Value = cookie.IfFail.Value
	return responseCookie, nil
}

func GetCanaryCookie(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	appName := vars["app"]

	config, err := ReadConfig()
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "Could not read config file!")
		return
	}

	for _, app := range config.Apps {
		if app.Name == appName {
			seed := rand.NewSource(time.Now().UnixNano())
			randGen := rand.New(seed)
			randNum := randGen.Float32()
			timeNow := time.Now()

			// generate the cookie response
			var cookieResponse CookieResponse
			if randNum < app.CookieInfo.CanaryPercent {
				cookieResponse, err = GetCookieResponse(app.CookieInfo, timeNow, true)
			} else {
				cookieResponse, err = GetCookieResponse(app.CookieInfo, timeNow, false)
			}
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Could not get canary cookie!")
			}

			// convert cookie response to json
			cookieJson, err := json.Marshal(cookieResponse)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Could not read cookie data!")
				return
			}

			respondJSON(w, cookieJson)
			return
		}
	}

	errMsg := fmt.Sprintf("Could not find application %v", appName)
	log.Println(errMsg)
	respondWithError(w, http.StatusInternalServerError, errMsg)
}

func respondWithError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func respondJSON(w http.ResponseWriter, jdata []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(jdata)
}

func GetCookieByType(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	appName := vars["app"]
	cookieType := vars["cookie-type"]

	config, err := ReadConfig()
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "Could not read config file!")
		return
	}

	for _, app := range config.Apps {
		if appName == app.Name {
			var cookieResponse CookieResponse
			if cookieType == "success" {
				cookieResponse, err = GetCookieResponse(app.CookieInfo, time.Now(), true)
			} else {
				cookieResponse, err = GetCookieResponse(app.CookieInfo, time.Now(), false)
			}
			if err != nil {
				log.Println(err)
				respondWithError(w, http.StatusInternalServerError, "Could not process request!")
			}

			kvJson, err := json.Marshal(cookieResponse)
			if err != nil {
				log.Println(err)
				respondWithError(w, http.StatusInternalServerError, "Could not process request!")
			}
			respondJSON(w, kvJson)
			return
		}
	}

	respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Could not find app = %v", appName))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.Method, r.RequestURI)
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
	router.Path("/").
		Queries("cookie-type", "{cookie-type:success|fail}", "app", "{app}").
		Methods("GET").
		HandlerFunc(GetCookieByType)

	router.Path("/").
		Queries("app", "{app}").
		Methods("GET").
		HandlerFunc(GetCanaryCookie)

	host := ":" + strconv.Itoa(config.Port)
	log.Println(fmt.Sprintf("Starting deployment-controller server on %v", host))
	http.ListenAndServe(host, router)
}
