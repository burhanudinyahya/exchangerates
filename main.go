package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

type ApiResponse struct {
	Data  interface{} `json:"data"`
	Error string      `json:"error,omitempty"`
}

var (
	client              = resty.New().SetTimeout(10 * time.Second)
	cacheLock           sync.RWMutex
	exchangeCache       interface{}
	currenciesCache     interface{}
	lastExchangeFetch   time.Time
	lastCurrenciesFetch time.Time
	appID               string
	exchangeRateURL     = "https://openexchangerates.org/api/latest.json"
	currenciesListURL   = "https://openexchangerates.org/api/currencies.json"
	cacheDuration       = time.Hour
)

func fetchDataFromAPI(url string) (interface{}, error) {
	response, err := client.R().Get(url)
	if err != nil || response.StatusCode() != http.StatusOK {
		return nil, errors.New("failed to fetch data from external API")
	}

	var jsonResponse interface{}
	if err := json.Unmarshal(response.Body(), &jsonResponse); err != nil {
		return nil, errors.New("failed to parse API response")
	}

	return jsonResponse, nil
}

func getCachedData(url string, cache *interface{}, lastFetch *time.Time) (interface{}, error) {
	cacheLock.RLock()
	if time.Since(*lastFetch) < cacheDuration {
		defer cacheLock.RUnlock()
		return *cache, nil
	}
	cacheLock.RUnlock()

	// Fetch new data
	data, err := fetchDataFromAPI(url)
	if err != nil {
		return nil, err
	}

	// Update cache
	cacheLock.Lock()
	*cache = data
	*lastFetch = time.Now()
	cacheLock.Unlock()

	return data, nil
}

func getLatestExchangeRate(w http.ResponseWriter, r *http.Request) {
	data, err := getCachedData(exchangeRateURL+"?app_id="+appID, &exchangeCache, &lastExchangeFetch)
	if err != nil {
		sendResponse(w, nil, "Failed to fetch exchange rates")
		return
	}
	sendResponse(w, data, "")
}

func getCurrencyList(w http.ResponseWriter, r *http.Request) {
	data, err := getCachedData(currenciesListURL, &currenciesCache, &lastCurrenciesFetch)
	if err != nil {
		sendResponse(w, nil, "Failed to fetch currency list")
		return
	}
	sendResponse(w, data, "")
}

func sendResponse(w http.ResponseWriter, data interface{}, errMessage string) {
	w.Header().Set("Content-Type", "application/json")
	if errMessage != "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ApiResponse{Error: errMessage})
	} else {
		json.NewEncoder(w).Encode(ApiResponse{Data: data})
	}
}

func main() {
	// Retrieve APP_ID from environment variable
	appID = os.Getenv("APP_ID")
	if appID == "" {
		log.Fatal("APP_ID environment variable is not set")
	}

	http.HandleFunc("/api/latest", getLatestExchangeRate)
	http.HandleFunc("/api/currencies", getCurrencyList)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
