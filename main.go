package main

import (
	"fmt"
	"log"

	"github.com/gocolly/colly"
)

func main() {
	fmt.Println("Hello, Lianjia Scraper!")
	// Instantiate area collector
	areaCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
	)

	// Before making a request print "Visiting ..."
	areaCollector.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	// On every an element which has data-housecode attribute call callback
	areaCollector.OnHTML("#district-filter-box", func(e *colly.HTMLElement) {
		e.ForEach("a", func(_ int, e *colly.HTMLElement) {
			fmt.Println(e.Text)
		})
	})
}
