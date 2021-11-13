package main

import (
	"log"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/queue"
)

func main() {

	urlBase := "https://bj.lianjia.com"

	// Instantiate area collector
	areaCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),

		//colly.Debugger(&debug.LogDebugger{}),
	)

	// areaQueue is a rate limited queue which has a
	// consumer that the area collector will use
	// to request the next URL.
	areaQueue, _ := queue.New(
		1,                                        // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 20}, // Use default queue storage
	)

	// Before making a request print "Visiting ..."
	areaCollector.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	// On every an element which has data-housecode attribute call callback
	areaCollector.OnHTML("div[data-role='ershoufang']", func(e *colly.HTMLElement) {
		e.ForEach("a", func(_ int, e *colly.HTMLElement) {
			link := urlBase + e.Attr("href")
			areaQueue.AddURL(link)
			log.Println("Adding ", link)
		})
	})

	// Start scraping ershoufang information on https://lianjia.com/
	areaCollector.Visit(urlBase + "/ershoufang/")
}
