package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	nested "github.com/automano/nested-logrus-formatter"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/queue"
	"github.com/sirupsen/logrus"
	_ "go.uber.org/automaxprocs"
)

// Create a new instance of the logger. You can have any number of instances.
var log = logrus.New()

// House stores information about a Lian Jia house
type House struct {
	ID                  int     // 房屋ID
	Title               string  // 房屋页面标题
	URL                 string  // 房屋页面链接
	TotalPrice          float32 // 总价
	TotalPriceUnit      string  // 总价单位
	UnitPrice           float32 // 单价
	UnitPriceUnit       string  // 单价单位
	Community           string  // 小区名称
	Area                string  // 小区位置
	SubArea             string  // 细分区域
	RingRoad            string  // 环路
	Type                string  // 房屋类型
	Floor               string  // 所在楼层
	GrossArea           float32 // 建筑面积
	Structure           string  // 户型结构
	NetArea             float32 // 套内面积
	BuildingType        string  // 建筑类型
	Orientation         string  // 房屋朝向
	BuildingStructure   string  // 建筑结构
	Decoration          string  // 装修情况
	Elevator            string  // 配备电梯
	ElevatorNum         string  // 梯户比例
	HeatingMode         string  // 供暖方式
	ListingTime         string  // 挂牌时间
	Transaction         string  // 交易权属
	LastTransactionTime string  // 上次交易
	Usage               string  // 房屋用途
	Year                string  // 房屋年限
	Property            string  // 产权所属
	Mortgage            string  // 抵押信息
	PropertyCert        string  // 房本备件
}

// toStringSlice convert Struct member to string slice
func (h House) toStringSlice() []string {
	var record []string
	val := reflect.ValueOf(h)
	num := val.NumField()

	for i := 0; i < num; i++ {
		switch val.Field(i).Kind() {
		case reflect.String:
			record = append(record, val.Field(i).String())
		case reflect.Int:
			record = append(record, strconv.Itoa(int(val.Field(i).Int())))
		case reflect.Float32:
			record = append(record, strconv.FormatFloat(val.Field(i).Float(), 'f', 2, 32))
		default:
			return nil
		}
	}
	return record
}

func main() {
	// The API for setting attributes is a little different from the package level
	// exported logger. See Godoc.
	// log related settings
	log.Out = os.Stdout
	log.SetLevel(logrus.InfoLevel) // set log level - change to InfoLevel to show less logs, DebugLevel to show more logs
	log.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		ShowFullLevel:   true,
		NoColors:        true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logFileName := fmt.Sprintf("log/log-%v.log", time.Now().Format("2006-01-02"))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		mw := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(mw)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}

	// reporter variables
	var (
		areaCount    int
		subAreaCount int
		pageCount    int
		detailCount  int
		houseCount   int
	)
	//open file to write
	file, err := os.OpenFile("output/output.csv", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	// UTF-8 BOM
	file.WriteString("\xEF\xBB\xBF")

	// Create a csv writer
	w := csv.NewWriter(file)
	defer w.Flush()

	// write header to csv
	w.Write([]string{"页面标题", "页面链接", "房屋总价", "总价单位", "房屋单价", "单价单位", "小区名称",
		"小区位置", "细分区域", "环路范围", "房屋类型", "所在楼层", "建筑面积", "户型结构", "套内面积",
		"建筑类型", "房屋朝向", "建筑结构", "装修情况", "配备电梯", "梯户比例", "供暖方式", "挂牌时间",
		"交易权属", "上次交易", "房屋用途", "房屋年限", "产权所属", "抵押信息", "房本备件"})

	// scraper

	const (
		ThreadsNum  = 5
		RandomDelay = 2
	)
	// url prefix
	urlPrefix := "https://bj.lianjia.com"

	// start areaCollector

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
		log.Info("Visiting ", r.URL.String())
	})

	// On every an element which has <div data-role="ershoufang"/> attribute call callback
	areaCollector.OnHTML("div[data-role='ershoufang']", func(e *colly.HTMLElement) {
		e.ForEach("a", func(_ int, e *colly.HTMLElement) {
			urlSuffix := e.Attr("href")
			link := urlPrefix + urlSuffix
			areaQueue.AddURL(link)
			areaCount += 1
			log.Infof("Adding Area URL [%d]: %s", areaCount, link)
		})
	})

	// start subAreaCollector
	// subAreaStore can check whether a URL has already been visited
	subAreaStore := make(map[string]bool)

	// Instantiate subArea collector
	subAreaCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
		// colly.Debugger(&debug.LogDebugger{}),
	)

	subAreaCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*lianjia.com",
		Parallelism: ThreadsNum, // Max parallelism
		Delay:       RandomDelay * time.Second,
	})

	subAreaQueue, _ := queue.New(
		ThreadsNum, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 300}, // Use default queue storage
	)

	// Before making a request print "Visiting ..."
	subAreaCollector.OnRequest(func(r *colly.Request) {
		log.Info("Visiting ", r.URL.String())
	})

	subAreaCollector.OnHTML("div[data-role=ershoufang] > div:nth-child(2)", func(e *colly.HTMLElement) {
		e.ForEach("a", func(_ int, e *colly.HTMLElement) {

			urlSuffix := e.Attr("href")
			// check whether the url has been visited
			if !subAreaStore[urlSuffix] {
				// add the url to the store
				subAreaStore[urlSuffix] = true
				link := urlPrefix + urlSuffix
				// add the subarea url to the queue
				subAreaQueue.AddURL(link)
				subAreaCount += 1
				log.Infof("Adding SubArea URL [%d]: %s", subAreaCount, link)
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

	pageCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*lianjia.com",
		Parallelism: ThreadsNum, // Max parallelism
		Delay:       RandomDelay * time.Second,
	})

	// pageQueue is a rate limited queue
	pageQueue, _ := queue.New(
		ThreadsNum, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 5000}, // Use default queue storage
	)

	pageCollector.OnRequest(func(r *colly.Request) {
		log.Info("Visiting ", r.URL.String())
	})

	pageCollector.OnHTML("div.page-box.house-lst-page-box", func(e *colly.HTMLElement) {

		var page pageData
		json.Unmarshal([]byte(e.Attr("page-data")), &page)
		log.Infof("Adding %d pages for %s", page.TotalPage, e.Request.URL.String())
		for i := 1; i <= page.TotalPage; i++ {
			link := e.Request.URL.String() + "pg" + strconv.Itoa(i) + "/"
			pageQueue.AddURL(link)
			pageCount += 1
			log.Infof("Adding Page URL [%d]: %s", pageCount, link)
		}
	})

	// instantiate detailCollector
	detailCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
		// colly.Debugger(&debug.LogDebugger{}),
	)

	detailCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*lianjia.com",
		Parallelism: ThreadsNum, // Max parallelism
		Delay:       RandomDelay * time.Second,
	})

	detailQueue, _ := queue.New(
		ThreadsNum, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 100000}, // Use default queue storage
	)

	// Before making a request print "Visiting ..."
	detailCollector.OnRequest(func(r *colly.Request) {
		log.Info("visiting ", r.URL.String())
	})

	detailCollector.OnHTML("div.content > div.leftContent > ul.sellListContent", func(e *colly.HTMLElement) {
		e.ForEach("li > a", func(_ int, e *colly.HTMLElement) {
			// get house detail url from a element href attribute
			link := e.Attr("href")
			//If link meet regex pattern
			re := regexp.MustCompile(`(?m)https://bj.lianjia.com/ershoufang/\d{12}.html`)

			if !re.MatchString(link) {
				return
			}
			// start scraping the page under the link found

			detailQueue.AddURL(link)
			detailCount += 1
			log.Infof("Adding house detail URL [%d]: %s", detailCount, link)
		})
	})

	// Instantiate house collector
	houseCollector := colly.NewCollector(
		// Visit only domains: lianjia.com, bj.lianjia.com
		colly.AllowedDomains("lianjia.com", "bj.lianjia.com"),
		// colly.Debugger(&debug.LogDebugger{}),
	)

	houseCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*lianjia.com",
		Parallelism: ThreadsNum, // Max parallelism
		Delay:       RandomDelay * time.Second,
	})

	// Before making a request print "Visiting ..."
	houseCollector.OnRequest(func(r *colly.Request) {
		log.Info("Visiting ", r.URL.String())
	})

	// Extract details of the house
	houseCollector.OnHTML("body", func(e *colly.HTMLElement) {

		// ID
		id := houseCount

		// Title
		title := removeComma(e.ChildText("div.title > h1"))

		// URL
		url := e.Request.URL.String()

		// Total Price
		totalPrice, err := strconv.ParseFloat(e.ChildText("div.price > span.total"), 2)
		if err != nil {
			log.Info("Total price is not integer.")
		}

		// Total Price Unit
		totalPriceUnit := e.ChildText("div.price > span.unit")

		// Unit Price Unit
		unitPriceUnit := e.ChildText("div.unitPrice > span.unitPriceValue > i")

		// Unit Price
		unitPriceString := e.ChildText("div.unitPrice > span.unitPriceValue")
		unitPriceString = strings.ReplaceAll(unitPriceString, unitPriceUnit, "")
		unitPrice, err := strconv.ParseFloat(unitPriceString, 2)
		if err != nil {
			log.Info("Unit Price is not integer.")
		}

		// community
		community := e.ChildText("div.communityName > a.info")

		// area, sub-area, ring-road
		location := e.ChildText("div.areaName > span.info")

		// get members of the slice
		locationSlice := strings.Fields(location)
		var area, subArea, ringRoad string

		switch len(locationSlice) {
		case 1:
			area = locationSlice[0]
		case 2:
			area = locationSlice[0]
			subArea = locationSlice[1]
		case 3:
			area = locationSlice[0]
			subArea = locationSlice[1]
			ringRoad = locationSlice[2]
		default:
			log.Info("Location is not in the right format.")
		}

		// create house instance
		house := House{
			ID:             id,
			Title:          title,
			URL:            url,
			TotalPrice:     float32(totalPrice),
			TotalPriceUnit: totalPriceUnit,
			UnitPrice:      float32(unitPrice),
			UnitPriceUnit:  unitPriceUnit,
			Community:      community,
			Area:           area,
			SubArea:        subArea,
			RingRoad:       ringRoad,
		}

		// fill base information
		e.ForEach("div.base > div.content > ul > li ", func(_ int, el *colly.HTMLElement) {
			label := el.ChildText("span.label")
			label = removeComma(label)
			value := strings.ReplaceAll(el.Text, label, "")
			value = setNull(value)
			switch label {
			case "房屋户型":
				house.Type = value
			case "所在楼层":
				house.Floor = value
			case "建筑面积":
				grossAreaString := strings.ReplaceAll(value, "㎡", "")
				grossArea, err := strconv.ParseFloat(grossAreaString, 2)
				if err != nil {
					log.Info("Can't parse the gross area.")
				}
				house.GrossArea = float32(grossArea)

			case "户型结构":
				house.Structure = value
			case "套内面积":
				netAreaString := strings.ReplaceAll(value, "㎡", "")
				netArea, err := strconv.ParseFloat(netAreaString, 2)
				if err != nil {
					log.Info("Can't parse the net area.")
				}
				house.NetArea = float32(netArea)
			case "建筑类型":
				house.BuildingType = value
			case "房屋朝向":
				house.Orientation = value
			case "建筑结构":
				house.BuildingStructure = value
			case "装修情况":
				house.Decoration = value
			case "梯户比例":
				house.ElevatorNum = value
			case "供暖方式":
				house.HeatingMode = value
			case "配备电梯":
				house.Elevator = value
			}
		})

		// transaction information
		e.ForEach("div.transaction > div.content > ul > li ", func(_ int, el *colly.HTMLElement) {
			// get content from element li>span
			content := el.ChildText("span")
			// remove all space and new lines
			content = removeSpace(content)

			// get label from element li>span.label
			label := el.ChildText("span.label")

			// get value by removing label from content
			value := strings.ReplaceAll(content, label, "")

			switch label {
			case "挂牌时间":
				house.ListingTime = value
			case "交易权属":
				house.Transaction = value
			case "上次交易":
				house.LastTransactionTime = value
			case "房屋用途":
				house.Usage = value
			case "房屋年限":
				house.Year = value
			case "产权所属":
				house.Property = value
			case "抵押信息":
				house.Mortgage = value
			case "房本备件":
				house.PropertyCert = value
			}
		})

		// append into houses slice
		w.Write(house.toStringSlice())
		log.Info("Adding house [", houseCount, "]: ", house)
		houseCount++
	})

	// Start scraping ershoufang information
	startT := time.Now()

	areaCollector.Visit(urlPrefix + "/ershoufang/")
	areaQueue.Run(subAreaCollector)
	subAreaQueue.Run(pageCollector)
	pageQueue.Run(detailCollector)
	detailQueue.Run(houseCollector)

	endT := time.Now()
	totalT := endT.Sub(startT)

	log.Info("areaCount: ", areaCount)
	log.Info("subAreaCount: ", subAreaCount)
	log.Info("pageCount: ", pageCount)
	log.Info("detailCount: ", detailCount)
	log.Info("total process time: ", totalT)
}

func removeComma(old string) (new string) {
	old = strings.ReplaceAll(old, ",", " ")
	new = strings.ReplaceAll(old, "，", " ")
	return new
}

func removeSpace(old string) (new string) {
	old = strings.ReplaceAll(old, " ", "")
	new = strings.ReplaceAll(old, "\n", "")
	return new
}

func setNull(s string) string {
	if s == "暂无数据" {
		s = ""
	}
	return s
}
