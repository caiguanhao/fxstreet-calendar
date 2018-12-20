package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Event struct {
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

func online() *goquery.Document {
	var req *http.Request
	req, err := http.NewRequest("GET", "https://calendar.fxstreet.com/EventDateWidget/GetMini", nil)
	if err != nil {
		log.Fatal(err)
	}
	q := req.URL.Query()
	q.Add("culture", "zh-CN")
	q.Add("rows", "50")
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
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = errors.New("status code != 200")
		log.Fatal(err)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	return doc
}

func offline() *goquery.Document {
	file, err := os.Open("data.html")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		log.Fatal(err)
	}
	return doc
}

func f(input string) string {
	return strings.TrimSpace(input)
}

func main() {
	doc := online()
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
				Country:    f(tr.Find(".fxst-flag").Text()),
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
		log.Fatal(err)
	}
	fmt.Println(string(bytes))
}
