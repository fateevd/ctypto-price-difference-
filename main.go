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
	"reflect"
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
	nowUrl       = "https://api.binance.com/api/v3/ticker/price?"
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
	price, err := strconv.ParseFloat(data[0][1].(string), 64)
	if err != nil {
		fmt.Println(url)
	}
	return price, err
}

func main() {
	records := readCsvFile("values.csv")
	allPrice := 0.0
	m := map[string]float64{}
	startBank := 0.0

	var wg sync.WaitGroup
	for _, str := range records {
		t := strings.Split(str[0], ";")

		date, currency := t[0], t[2]
		amount, _ := strconv.ParseFloat(t[1], 64)
		parsedTime, err := time.Parse(time.RFC3339, date)

		if err != nil {
			parsedTime = time.Now()
		}
		wg.Add(1)
		timestamp := parsedTime.Unix() * 1000

		history := createLinkWithQueryParams(baseUrl, TypeArgsForCreateLink{
			symbol:    currency + pairCurrency,
			limit:     limit,
			interval:  interval,
			startTime: strconv.FormatInt(timestamp, 10),
		})

		nowPriceUrl := createLinkWithQueryParams(nowUrl, TypeArgsForCreateLink{
			symbol: currency + pairCurrency,
		})

		go func() {
			oldPrice, err := getData(history)
			if err != nil {
				fmt.Println(currency, timestamp)
				return
			}
			startBank = round(startBank + convertCurrency(amount, oldPrice))
			nowPrice, err := getNowPrice(nowPriceUrl)
			if err != nil {
				fmt.Println(currency, timestamp)
				return
			}
			defer wg.Done()
			finalPrice := convertCurrency(amount, nowPrice) - convertCurrency(amount, oldPrice)
			allPrice += round(finalPrice)
			m[currency] = m[currency] + round(finalPrice)
		}()
	}
	wg.Wait()
	fmt.Println("start bank:", startBank, "\nnow bank:", round(startBank+allPrice), "\nprofit:", round(allPrice))
	fmt.Println("Your coins:")
	for c, v := range m {
		fmt.Println(c, round(v))
	}
}

type TypeArgsForCreateLink struct {
	symbol    string
	interval  string
	startTime string
	limit     string
}

func createLinkWithQueryParams(link string, args TypeArgsForCreateLink) string {
	val := reflect.ValueOf(args)
	typ := reflect.TypeOf(args)
	queryParams := url.Values{}
	for i := 0; i < val.NumField(); i++ {
		fieldName := typ.Field(i).Name
		fieldValue := val.Field(i).String()
		if fieldValue == "" {
			continue
		}
		queryParams.Set(fieldName, fieldValue)
	}
	return link + queryParams.Encode()
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

type Data struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

func getNowPrice(url string) (float64, error) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Ошибка при выполнении GET-запроса:", err)
		return 0, nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var data Data

	if err := json.Unmarshal([]byte(body), &data); err != nil {
		fmt.Println("Ошибка при разборе JSON:", err)
		return 0, err
	}

	price, err := strconv.ParseFloat(data.Price, 64)
	return price, err
}
