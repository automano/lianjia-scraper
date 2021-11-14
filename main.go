package main

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/queue"
)

func main() {

	file, err := os.OpenFile("output/output.csv", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	file.WriteString("id,name,price,description,image,category,subcategory,subsubcategory,brand,model,color,size")
	file.WriteString("\n")

	urlPrefix := "https://bj.lianjia.com"

	// start areaCollector
	areaCount := 0
	// Instantiate area collector
	areaCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
		//colly.Debugger(&debug.LogDebugger{}),
	)

	// areaQueue is a rate limited queue
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

	// start subAreaCollector
	// subAreaStore can check whether a URL has already been visited
	subAreaStore := make(map[string]bool)
	subAreaCount := 0 // Count of subArea URLs

	// Instantiate subArea collector
	subAreaCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
		// colly.Debugger(&debug.LogDebugger{}),
	)

	subAreaQueue, _ := queue.New(
		3, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 300}, // Use default queue storage
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
				// add the url to the store
				subAreaStore[urlSuffix] = true
				subAreaCount += 1
				link := urlPrefix + urlSuffix
				// add the subarea url to the queue
				subAreaQueue.AddURL(link)
				log.Printf("Adding SubArea URL [%d]: %s", subAreaCount, link)
			}
		})
	})

	// start pageCollector
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

	// pageQueue is a rate limited queue
	pageQueue, _ := queue.New(
		5, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 10000}, // Use default queue storage
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
			pageQueue.AddURL(link)
			log.Printf("Adding Page URL [%d]: %s", i, link)
		}
	})

	// instantiate detailCollector
	detailCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
		// colly.Debugger(&debug.LogDebugger{}),
	)

	detailQueue, _ := queue.New(
		5, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 10000}, // Use default queue storage
	)

	// Before making a request print "Visiting ..."
	detailCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	detailCollector.OnHTML("a[data-housecode]", func(e *colly.HTMLElement) {
		// get house detail url from a element href attribute
		link := e.Attr("href")
		//If link meet regex pattern
		re := regexp.MustCompile(`(?m)https://bj.lianjia.com/ershoufang/\d{12}.html`)

		if !re.MatchString(link) {
			return
		}
		// start scraping the page under the link found
		log.Printf("Adding house detail URL [%d]: %s", e.Index, link)
		// _ = detailCollector.Visit(link)

		detailQueue.AddURL(link)
	})

	// House stores information about a Lian Jia house
	type House struct {
		Title             string  // 房屋页面标题
		URL               string  // 房屋页面链接
		TotalPrice        int     // 总价
		TotalPriceUnit    string  // 总价单位
		UnitPrice         int     // 单价
		UnitPriceUnit     string  // 单价单位
		Community         string  // 小区名称
		Location          string  // 小区位置
		Type              string  // 房屋类型
		Floor             string  // 所在楼层
		GrossArea         float64 // 建筑面积
		Structure         string  // 户型结构
		NetArea           float64 // 套内面积
		BuildingType      string  // 建筑类型
		Orientation       string  // 房屋朝向
		BuildingStructure string  // 建筑结构
		Decoration        string  // 装修情况
		Elevator          string  // 配备电梯
		ElevatorNum       string  // 梯户比例
		HeatingMode       string  // 供暖方式
		ListingTime       string  // 挂牌时间
		Transaction       string  // 交易权属
		LastTransaction   string  // 上次交易
		Usage             string  // 房屋用途
		Year              string  // 房屋年限
		Property          string  // 产权所属
		Mortgage          string  // 抵押信息
		PropertyCert      string  // 房本备件
	}

	// Instantiate house collector
	houseCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
		// colly.Debugger(&debug.LogDebugger{}),
	)

	// Before making a request print "Visiting ..."
	houseCollector.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	// Extract details of the house
	houseCollector.OnHTML("body", func(e *colly.HTMLElement) {

		// title
		title := e.ChildText("div.title > h1")

		// total price
		totalPrice, err := strconv.Atoi(e.ChildText("div.price > span.total"))
		if err != nil {
			log.Println("Total price is not integer.")
		}
		totalPriceUnit := e.ChildText("div.price > span.unit")

		// unit price
		unitPriceUnit := e.ChildText("div.unitPrice > span.unitPriceValue > i")

		unitPriceString := e.ChildText("div.unitPrice > span.unitPriceValue")
		unitPriceString = strings.TrimSuffix(unitPriceString, unitPriceUnit)
		unitPrice, err := strconv.Atoi(unitPriceString)
		if err != nil {
			log.Println("Unit Price is not integer.")
		}

		// community
		community := e.ChildText("div.communityName > a.info")

		// area
		location := e.ChildText("div.areaName > span.info")

		// create house instance
		house := House{
			Title:          title,
			URL:            e.Request.URL.String(),
			TotalPrice:     totalPrice,
			TotalPriceUnit: totalPriceUnit,
			UnitPrice:      unitPrice,
			UnitPriceUnit:  unitPriceUnit,
			Community:      community,
			Location:       location,
		}

		// fill base information
		e.ForEach("div.base > div.content > ul > li ", func(_ int, el *colly.HTMLElement) {
			label := el.ChildText("span.label")
			switch label {
			case "房屋户型":
				house.Type = strings.TrimPrefix(el.Text, label)
			case "所在楼层":
				house.Floor = strings.TrimPrefix(el.Text, label)
			case "建筑面积":
				grossAreaString := strings.TrimSuffix(el.Text, "㎡")
				grossAreaString = strings.TrimPrefix(grossAreaString, label)
				grossArea, err := strconv.ParseFloat(grossAreaString, 2)
				if err != nil {
					log.Println("Can't parse the gross area.")
				}
				house.GrossArea = grossArea

			case "户型结构":
				house.Structure = strings.TrimPrefix(el.Text, label)
			case "套内面积":
				netAreaString := strings.TrimSuffix(el.Text, "㎡")
				netAreaString = strings.TrimPrefix(netAreaString, label)
				netArea, err := strconv.ParseFloat(netAreaString, 2)
				if err != nil {
					log.Println("Can't parse the net area.")
				}
				house.NetArea = netArea
			case "建筑类型":
				house.BuildingType = strings.TrimPrefix(el.Text, label)
			case "房屋朝向":
				house.Orientation = strings.TrimPrefix(el.Text, label)
			case "建筑结构":
				house.BuildingStructure = strings.TrimPrefix(el.Text, label)
			case "装修情况":
				house.Decoration = strings.TrimPrefix(el.Text, label)
			case "梯户比例":
				house.ElevatorNum = strings.TrimPrefix(el.Text, label)
			case "供暖方式":
				house.HeatingMode = strings.TrimPrefix(el.Text, label)
			case "配备电梯":
				house.Elevator = strings.TrimPrefix(el.Text, label)
			}
		})

		// transaction information
		e.ForEach("div.transaction > div.content > ul > li ", func(_ int, el *colly.HTMLElement) {
			label := el.ChildText("span.label")
			// format content
			content := strings.TrimPrefix(el.Text, label)
			content = strings.ReplaceAll(content, " ", "")
			content = strings.ReplaceAll(content, "\n", "")
			switch label {
			case "挂牌时间":
				house.ListingTime = strings.TrimPrefix(content, label)
			case "交易权属":
				house.Transaction = strings.TrimPrefix(content, label)
			case "上次交易":
				house.LastTransaction = strings.TrimPrefix(content, label)
			case "房屋用途":
				house.Usage = strings.TrimPrefix(content, label)
			case "房屋年限":
				house.Year = strings.TrimPrefix(content, label)
			case "产权所属":
				house.Property = strings.TrimPrefix(content, label)
			case "抵押信息":
				house.Mortgage = strings.TrimPrefix(content, label)
			case "房本备件":
				house.PropertyCert = strings.TrimPrefix(content, label)
			}
		})

		// append into houses slice
		log.Println("Appending house:", house)
	})

	// Start scraping ershoufang information
	// areaCollector.Visit(urlPrefix + "/ershoufang/")
	// areaQueue.Run(subAreaCollector)
	// subAreaQueue.Run(pageCollector)
	detailQueue.AddURL("https://bj.lianjia.com/ershoufang/101111350123.html")
	detailQueue.Run(houseCollector)
}
