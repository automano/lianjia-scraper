package main

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/gocolly/colly"
)

func main() {

	// urlPrefix := "https://bj.lianjia.com"

	// areaCount := 0
	// // Instantiate area collector
	// areaCollector := colly.NewCollector(
	// 	// Visit only domains: lianjia.com, bj.lianjia.com
	// 	colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
	// 	//colly.Debugger(&debug.LogDebugger{}),
	// )

	// // areaQueue is a rate limited queue
	// areaQueue, _ := queue.New(
	// 	1,                                        // Number of consumer threads
	// 	&queue.InMemoryQueueStorage{MaxSize: 20}, // Use default queue storage
	// )

	// // Before making a request print "Visiting ..."
	// areaCollector.OnRequest(func(r *colly.Request) {
	// 	log.Println("Visiting", r.URL.String())
	// })

	// // On every an element which has <div data-role="ershoufang"/> attribute call callback
	// areaCollector.OnHTML("div[data-role='ershoufang']", func(e *colly.HTMLElement) {
	// 	e.ForEach("a", func(_ int, e *colly.HTMLElement) {
	// 		urlSuffix := e.Attr("href")
	// 		link := urlPrefix + urlSuffix
	// 		areaQueue.AddURL(link)
	// 		log.Printf("Adding Area URL [%d]: %s", areaCount, link)
	// 	})
	// })

	// // subAreaStore can check whether a URL has already been visited
	// subAreaStore := make(map[string]bool)
	// subAreaCount := 0 // Count of subArea URLs

	// // Instantiate subArea collector
	// subAreaCollector := colly.NewCollector(
	// 	// Visit only domains: lianjia.com, bj.lianjia.com
	// 	colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
	// 	// colly.Debugger(&debug.LogDebugger{}),
	// )

	// subAreaQueue, _ := queue.New(
	// 	3, // Number of consumer threads
	// 	&queue.InMemoryQueueStorage{MaxSize: 300}, // Use default queue storage
	// )

	// // Before making a request print "Visiting ..."
	// subAreaCollector.OnRequest(func(r *colly.Request) {
	// 	log.Println("Visiting", r.URL.String())
	// })

	// subAreaCollector.OnHTML("div[data-role=ershoufang] > div:nth-child(2)", func(e *colly.HTMLElement) {
	// 	e.ForEach("a", func(_ int, e *colly.HTMLElement) {

	// 		urlSuffix := e.Attr("href")
	// 		// check whether the url has been visited
	// 		if !subAreaStore[urlSuffix] {
	// 			// add the url to the store
	// 			subAreaStore[urlSuffix] = true
	// 			subAreaCount += 1
	// 			link := urlPrefix + urlSuffix
	// 			// add the subarea url to the queue
	// 			subAreaQueue.AddURL(link)
	// 			log.Printf("Adding SubArea URL [%d]: %s", subAreaCount, link)
	// 		}
	// 	})
	// })

	// // Start scraping ershoufang information
	// areaCollector.Visit(urlPrefix + "/ershoufang/")
	// areaQueue.Run(subAreaCollector)

	type pageData struct {
		TotalPage int `json:"totalPage"`
		CurPage   int `json:"curPage"`
	}

	// Instantiate page collector
	pageCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
		// colly.Debugger(&debug.LogDebugger{}),
	)

	pageCollector.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	pageCollector.OnHTML("div.page-box.house-lst-page-box", func(e *colly.HTMLElement) {

		var page pageData
		json.Unmarshal([]byte(e.Attr("page-data")), &page)
		log.Printf("Adding %d pages for %s", page.TotalPage, e.Request.URL.String())
		for i := 1; i <= page.TotalPage; i++ {
			link := e.Request.URL.String() + "pg" + strconv.Itoa(i) + "/"
			log.Printf("Adding Page URL [%d]: %s", i, link)
		}
	})

	pageCollector.Visit("https://bj.lianjia.com/ershoufang/andingmen/")

}
