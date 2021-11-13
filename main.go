package main

import (
	"fmt"
	"log"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
)

func main() {
	// Instantiate area collector
	areaCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
		colly.Debugger(&debug.LogDebugger{}),
	)

	// Before making a request print "Visiting ..."
	areaCollector.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	// On every an element which has data-housecode attribute call callback
	areaCollector.OnHTML("div[data-role='ershoufang']", func(e *colly.HTMLElement) {
		e.ForEach("a", func(_ int, e *colly.HTMLElement) {
			fmt.Println(e.Attr("href"))
		})
	})

	// Start scraping on https://lianjia.com/
	areaCollector.Visit("https://bj.lianjia.com/ershoufang/")
}
