package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	baseUrl      = "https://api.binance.com/api/v3/klines?"
	interval     = "1m"
	pairCurrency = "USDT"
	limit        = "1"
)

func convertCurrency(amount float64, exchangeRate float64) float64 {
	return amount * exchangeRate
}

func round(number float64) float64 {
	return math.Round(number*100) / 100
}

func getData(url string) (float64, error) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Ошибка при выполнении GET-запроса:", err)
		return 0, nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var data [][]interface{}

	if err := json.Unmarshal([]byte(body), &data); err != nil {
		fmt.Println("Ошибка при разборе JSON:", err)
		return 0, err
	}
	//fmt.Println(resp.Status)
	price, _ := strconv.ParseFloat(data[0][1].(string), 64)
	return price, err
}

func main() {
	records := readCsvFile("text.csv")
	allPrice := 0.0
	m := map[string]float64{}
	var wg sync.WaitGroup
	for _, t := range records {

		date, currency := strings.Trim(t[0], " "), strings.Trim(t[2], " ")
		amount, _ := strconv.ParseFloat(t[1], 64)
		parsedTime, err := time.Parse(time.RFC3339, date)
		if err != nil {
			parsedTime = time.Now()
		}
		wg.Add(1)
		timestamp := parsedTime.Unix() * 1000
		history := baseUrl + getQueryParams(currency, timestamp)
		nowPriceUrl := baseUrl + getQueryParams(currency, (time.Now().Unix()-1000)*1000)
		go func() {
			oldPrice, _ := getData(history)
			nowPrice, _ := getData(nowPriceUrl)
			defer wg.Done()
			finalPrice := convertCurrency(amount, nowPrice) - convertCurrency(amount, oldPrice)
			allPrice += round(finalPrice)
			m[currency] = m[currency] + round(finalPrice)
		}()
	}
	wg.Wait()
	fmt.Println(round(allPrice))
	fmt.Println(m)
}

func getQueryParams(currency string, timestamp int64) string {
	startTime := strconv.FormatInt(timestamp, 10)
	queryParams := url.Values{}
	queryParams.Set("symbol", currency+pairCurrency)
	queryParams.Set("interval", interval)
	queryParams.Set("startTime", startTime)
	queryParams.Set("limit", limit)
	return queryParams.Encode()
}

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}

	return records
}
