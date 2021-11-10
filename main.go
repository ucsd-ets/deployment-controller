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
}

type View struct {
	ShowSuccess bool `yaml:"showSuccess"`
	ShowFail    bool `yaml:"showFail"`
}

type Logging struct {
	Disable bool `yaml:"disable"`
}

type App struct {
	Name       string  `yaml:"appName"`
	Mode       string  `yaml:"mode"`
	CookieInfo Cookie  `yaml:"cookieInfo"`
	View       View    `yaml:"view"`
	Logging    Logging `yaml:"logging"`
}

type Config struct {
	Apps []App `yaml:"apps"`
	Port int   `yaml:"port"`
}

type AppResponse struct {
	Key           string
	Value         string
	Expiration    string
	CanaryPercent float32
	Mode          string
}

// https://stackoverflow.com/questions/15323767/does-go-have-if-x-in-construct-similar-to-python
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
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

func GetAppResponse(cookie Cookie, timeNow time.Time, successCookieType bool, mode string) (AppResponse, error) {

	hours := strings.TrimSuffix(cookie.Expiration, "h")
	expiryHours, err := strconv.Atoi(hours)
	if err != nil {
		return AppResponse{}, err
	}
	exp := timeNow.Add(time.Hour * time.Duration(expiryHours)).Format(time.RFC3339)

	responseCookie := AppResponse{
		Key:           cookie.IfSuccessful.Key,
		Value:         cookie.IfSuccessful.Value,
		Expiration:    exp,
		CanaryPercent: cookie.CanaryPercent,
		Mode:          mode,
	}

	if successCookieType {
		return responseCookie, nil
	}

	responseCookie.Key = ""
	responseCookie.Value = ""
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
			var cookieResponse AppResponse
			if randNum < app.CookieInfo.CanaryPercent {
				cookieResponse, err = GetAppResponse(app.CookieInfo, timeNow, true, app.Mode)
			} else {
				cookieResponse, err = GetAppResponse(app.CookieInfo, timeNow, false, app.Mode)
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
			var cookieResponse AppResponse
			if cookieType == "success" {
				cookieResponse, err = GetAppResponse(app.CookieInfo, time.Now(), true, app.Mode)
			} else {
				cookieResponse, err = GetAppResponse(app.CookieInfo, time.Now(), false, app.Mode)
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

func GetViews(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	appName := vars["app"]

	config, err := ReadConfig()
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "Could not process request!")
	}

	for _, app := range config.Apps {
		if appName == app.Name {
			viewResponse, err := json.Marshal(app.View)
			if err != nil {
				log.Println(err)
				respondWithError(w, http.StatusInternalServerError, "Could not process request!")
			}

			respondJSON(w, viewResponse)
			return
		}
	}
	respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Could not find app = %v", appName))
}

func GetLogging(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	appName := vars["app"]

	config, err := ReadConfig()
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "Could not process request!")
	}
	for _, app := range config.Apps {
		if appName == app.Name {
			loggingResponse, err := json.Marshal(app.Logging)
			if err != nil {
				log.Println(err)
				respondWithError(w, http.StatusInternalServerError, "Could not process request!")
			}

			respondJSON(w, loggingResponse)
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
	router.Path("/apps/{app}").
		Queries("cookie-type", "{cookie-type:success|fail}").
		Methods("GET").
		HandlerFunc(GetCookieByType)

	router.Path("/apps/{app}/views").
		Methods("GET").
		HandlerFunc(GetViews)

	router.Path("/apps/{app}").
		Methods("GET").
		HandlerFunc(GetCanaryCookie)

	router.Path("/apps/{app}/logging").
		Methods("GET").
		HandlerFunc(GetLogging)

	host := ":" + strconv.Itoa(config.Port)
	log.Println(fmt.Sprintf("Starting deployment-controller server on %v", host))
	http.ListenAndServe(host, router)
}
