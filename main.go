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
	client            = resty.New().SetTimeout(10 * time.Second)
	cacheLock         sync.Mutex
	exchangeCache     = "latest.json"
	currenciesCache   = "currencies.json"
	appID             = "APP_ID"
	exchangeRateURL   = "https://openexchangerates.org/api/latest.json"
	currenciesListURL = "https://openexchangerates.org/api/currencies.json"
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

func saveToCache(fileName string, data interface{}) error {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	fileData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(fileName, fileData, 0644)
}

func loadFromCache(fileName string) (interface{}, error) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	fileData, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var jsonResponse interface{}
	if err := json.Unmarshal(fileData, &jsonResponse); err != nil {
		return nil, err
	}

	return jsonResponse, nil
}

func isCacheValid(fileName string) bool {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	info, err := os.Stat(fileName)
	if err != nil {
		return false
	}

	modTime := info.ModTime()
	return modTime.Year() == time.Now().Year() && modTime.YearDay() == time.Now().YearDay()
}

func getLatestExchangeRate(w http.ResponseWriter, r *http.Request) {
	handleCachedAPI(w, exchangeRateURL+"?app_id="+appID, exchangeCache)
}

func getCurrencyList(w http.ResponseWriter, r *http.Request) {
	handleCachedAPI(w, currenciesListURL, currenciesCache)
}

func handleCachedAPI(w http.ResponseWriter, url, cacheFile string) {
	var data interface{}
	var err error

	if isCacheValid(cacheFile) {
		data, err = loadFromCache(cacheFile)
	} else {
		data, err = fetchDataFromAPI(url)
		if err == nil {
			_ = saveToCache(cacheFile, data)
		}
	}

	if err != nil {
		sendResponse(w, nil, "Failed to fetch data")
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
	http.HandleFunc("/api/latest", getLatestExchangeRate)
	http.HandleFunc("/api/currencies", getCurrencyList)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
