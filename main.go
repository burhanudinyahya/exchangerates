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
	cacheMinute         = 5
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

func nextCacheExpirationTime() time.Time {
	now := time.Now()
	nextHour := now.Truncate(time.Hour).Add(time.Hour)
	return nextHour.Add(time.Minute * time.Duration(cacheMinute))
}

func isCacheExpired(lastFetch time.Time) bool {
	now := time.Now()
	expiration := nextCacheExpirationTime()
	if now.Before(expiration) && now.Sub(lastFetch) < time.Hour {
		return false
	}
	return true
}

func getCachedData(url string, cache *interface{}, lastFetch *time.Time) (interface{}, error) {
	cacheLock.RLock()
	if !isCacheExpired(*lastFetch) {
		defer cacheLock.RUnlock()
		return *cache, nil
	}
	cacheLock.RUnlock()

	data, err := fetchDataFromAPI(url)
	if err != nil {
		return nil, err
	}

	cacheLock.Lock()
	*cache = data
	*lastFetch = time.Now()
	cacheLock.Unlock()

	return data, nil
}

func getLatestExchangeRate(w http.ResponseWriter, r *http.Request) {
	data, err := getCachedData(exchangeRateURL+"?app_id="+appID, &exchangeCache, &lastExchangeFetch)
	if err != nil {
		sendErrorResponse(w, "Failed to fetch exchange rates")
		return
	}
	sendDataResponse(w, data)
}

func getCurrencyList(w http.ResponseWriter, r *http.Request) {
	data, err := getCachedData(currenciesListURL, &currenciesCache, &lastCurrenciesFetch)
	if err != nil {
		sendErrorResponse(w, "Failed to fetch currency list")
		return
	}
	sendDataResponse(w, data)
}

func sendDataResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func sendErrorResponse(w http.ResponseWriter, errMessage string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{"error": errMessage})
}

func main() {
	appID = os.Getenv("APP_ID")
	if appID == "" {
		log.Fatal("APP_ID environment variable is not set")
	}

	http.HandleFunc("/api/latest", getLatestExchangeRate)
	http.HandleFunc("/api/currencies", getCurrencyList)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
