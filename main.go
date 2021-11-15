package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type CookieResponse struct {
	Key           string
	Value         string
	Expiration    string
	AllCookies    [2]map[string]string
	CanaryPercent float32
	Disable       bool
}

func GetCookieResponse(cookie Cookie, timeNow time.Time, successCookieType bool, disable bool) (CookieResponse, error) {

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
		Disable:       disable,
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
				cookieResponse, err = GetCookieResponse(app.CookieInfo, timeNow, true, app.Disable)
			} else {
				cookieResponse, err = GetCookieResponse(app.CookieInfo, timeNow, false, app.Disable)
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
				cookieResponse, err = GetCookieResponse(app.CookieInfo, time.Now(), true, app.Disable)
			} else {
				cookieResponse, err = GetCookieResponse(app.CookieInfo, time.Now(), false, app.Disable)
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

func UpdateAppConfig(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var app App
	if err := decoder.Decode(&app); err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not decode app fields from body")
		return
	}

	err := UpdateConfig(app)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "Could not save config")
		return
	}

	w.WriteHeader(http.StatusOK)
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
	router := mux.NewRouter()                            // public routes
	protected := router.PathPrefix("/admin").Subrouter() // private

	router.Use(loggingMiddleware)
	protected.Use(loggingMiddleware)
	protected.Use(ApiKeyAuthMiddleware)

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

	protected.Path("/{app}").
		Methods("PUT").
		HandlerFunc(UpdateAppConfig)

	host := ":" + strconv.Itoa(config.Port)
	log.Println(fmt.Sprintf("Starting deployment-controller server on %v", host))
	http.ListenAndServe(host, router)
}
