package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-redis/redis"
)

type (
	Configs struct {
		RedisAddress  string `json:"redisAddress"`
		RedisDatabase int    `json:"redisDatabase"`
		Interval      int    `json:"interval"`
	}

	Event struct {
		Month      string `json:"month"`
		Day        string `json:"day"`
		Time       string `json:"time"`
		Country    string `json:"country"`
		Currency   string `json:"currency"`
		Title      string `json:"title"`
		Volatility string `json:"volatility"`
		Actual     string `json:"actual"`
		Consensus  string `json:"consensus"`
		Previous   string `json:"previous"`
		Revised    string `json:"revised"`
	}
)

var (
	configs = Configs{
		RedisAddress:  "localhost:6379",
		RedisDatabase: 15,
		Interval:      30,
	}

	redisClient *redis.Client
)

func online() *goquery.Document {
	var req *http.Request
	req, err := http.NewRequest("GET", "https://calendar.fxstreet.com/EventDateWidget/GetMini", nil)
	if err != nil {
		log.Println(err)
		return nil
	}
	q := req.URL.Query()
	q.Add("culture", "zh-CN")
	q.Add("rows", "100")
	q.Add("pastevents", "5")
	q.Add("hoursbefore", "20")
	q.Add("timezone", "China Standard Time")
	q.Add("columns", "date,time,country,event,consensus,previous,volatility,actual,countrycurrency")
	q.Add("countrycode", "AU,CA,JP,EMU,NZ,CH,UK,US")
	q.Add("isfree", "true")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")
	client := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = errors.New("status code != 200")
		log.Println(err)
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Println(err)
		return nil
	}
	return doc
}

func offline() *goquery.Document {
	file, err := os.Open("data.html")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer file.Close()
	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		log.Println(err)
		return nil
	}
	return doc
}

func f(input string) string {
	return strings.TrimSpace(input)
}

func getAndSet() {
	doc := online()
	if doc == nil {
		return
	}
	re := regexp.MustCompile("[0-9]+")
	currentMonth, currentDay := "", ""
	events := []Event{}
	doc.Find("tbody tr").Each(func(_ int, tr *goquery.Selection) {
		if tr.HasClass("fxst-dateRow") {
			segments := re.FindAllString(tr.Text(), -1)
			currentMonth, currentDay = segments[0], segments[1]
		} else if tr.HasClass("fxit-eventrow") {
			event := Event{
				Month:      currentMonth,
				Day:        currentDay,
				Time:       f(tr.Find(".fxst-td-time").Text()),
				Country:    f(tr.Find(".fxst-flag").AttrOr("title", "")),
				Currency:   f(tr.Find(".fxst-td-currency").Text()),
				Title:      f(tr.Find(".fxst-td-event").Text()),
				Volatility: f(tr.Find(".fxst-td-vol").Text()),
				Actual:     f(tr.Find(".fxst-td-act").Text()),
				Consensus:  f(tr.Find(".fxst-td-cons").Text()),
				Previous:   f(tr.Find(".fxst-td-prev").Text()),
				Revised:    f(tr.Find(".fxst-td-revised").Text()),
			}
			events = append(events, event)
		}
	})
	bytes, err := json.Marshal(events)
	if err != nil {
		log.Println(err)
		return
	}
	if err = redisClient.Set("forex:calendar", string(bytes), 0).Err(); err != nil {
		log.Println(err)
		return
	}
	log.Println("OK")
}

func init() {
	var err error

	var file *os.File
	file, err = os.Open("fxstreet-calendar-configs.json")
	if err == nil {
		defer file.Close()
		decoder := json.NewDecoder(file)
		if err = decoder.Decode(&configs); err != nil {
			log.Fatal(err)
		}
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: configs.RedisAddress,
		DB:   configs.RedisDatabase,
	})
	_, err = redisClient.Ping().Result()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	for {
		getAndSet()
		time.Sleep(time.Duration(configs.Interval) * time.Second)
	}
}
