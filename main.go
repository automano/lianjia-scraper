package main

import (
	"log"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/queue"
)

func main() {

	urlPrefix := "https://bj.lianjia.com"

	areaCount := 0
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

	// On every an element which has <div data-role="ershoufang"/> attribute call callback
	areaCollector.OnHTML("div[data-role='ershoufang']", func(e *colly.HTMLElement) {
		e.ForEach("a", func(_ int, e *colly.HTMLElement) {
			urlSuffix := e.Attr("href")
			link := urlPrefix + urlSuffix
			areaQueue.AddURL(link)
			log.Printf("Adding Area URL [%d]: %s", areaCount, link)
		})
	})

	// subAreaStore can check whether a URL has already been visited
	subAreaStore := make(map[string]bool)
	subAreaCount := 0 // Count of subArea URLs

	// Instantiate subArea collector
	subAreaCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
		// colly.Debugger(&debug.LogDebugger{}),
	)

	// Before making a request print "Visiting ..."
	subAreaCollector.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	subAreaCollector.OnHTML("div[data-role=ershoufang] > div:nth-child(2)", func(e *colly.HTMLElement) {
		e.ForEach("a", func(_ int, e *colly.HTMLElement) {

			urlSuffix := e.Attr("href")
			// check whether the url has been visited
			if !subAreaStore[urlSuffix] {
				subAreaStore[urlSuffix] = true
				subAreaCount += 1
				link := urlPrefix + urlSuffix
				//subAreaQueue.AddURL(link)
				log.Printf("Adding SubArea URL [%d]: %s", subAreaCount, link)
			}
		})
	})

	// Start scraping ershoufang information on https://lianjia.com/
	areaCollector.Visit(urlPrefix + "/ershoufang/")
	areaQueue.Run(subAreaCollector)
}
