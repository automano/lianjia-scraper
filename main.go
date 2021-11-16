package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
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
		case reflect.Float64:
			record = append(record, strconv.FormatFloat(val.Field(i).Float(), 'f', 2, 64))
		default:
			return nil
		}
	}
	return record
}

func main() {
	// The API for setting attributes is a little different than the package level
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
		log.Out = logFile
	} else {
		log.Info("Failed to log to file, using default stderr")
	}

	// reporter variables
	var (
		areaCount    int
		subAreaCount int
		pageCount    int
		detailCount  int
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
	w.Write([]string{"房屋页面标题", "房屋页面链接", "总价", "总价单位", "单价", "单价单位", "小区名称", "小区位置", "房屋类型", "所在楼层", "建筑面积", "户型结构", "套内面积", "建筑类型", "房屋朝向", "建筑结构", "装修情况", "配备电梯", "梯户比例", "供暖方式", "挂牌时间", "交易权属", "上次交易", "房屋用途", "房屋年限", "产权所属", "抵押信息", "房本备件"})

	// scaper
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
		Parallelism: 3, // Max parallelism
		Delay:       5 * time.Second,
	})

	subAreaQueue, _ := queue.New(
		3, // Number of consumer threads
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
		Parallelism: 5, // Max parallelism
		Delay:       5 * time.Second,
	})

	// pageQueue is a rate limited queue
	pageQueue, _ := queue.New(
		5, // Number of consumer threads
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
		Parallelism: 5, // Max parallelism
		Delay:       5 * time.Second,
	})

	detailQueue, _ := queue.New(
		5, // Number of consumer threads
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
		Parallelism: 5, // Max parallelism
		Delay:       5 * time.Second,
	})

	// Before making a request print "Visiting ..."
	houseCollector.OnRequest(func(r *colly.Request) {
		log.Info("Visiting ", r.URL.String())
	})

	// Extract details of the house
	houseCollector.OnHTML("body", func(e *colly.HTMLElement) {

		// title
		title := e.ChildText("div.title > h1")

		// total price
		totalPrice, err := strconv.Atoi(e.ChildText("div.price > span.total"))
		if err != nil {
			log.Info("Total price is not integer.")
		}
		totalPriceUnit := e.ChildText("div.price > span.unit")

		// unit price
		unitPriceUnit := e.ChildText("div.unitPrice > span.unitPriceValue > i")

		unitPriceString := e.ChildText("div.unitPrice > span.unitPriceValue")
		unitPriceString = strings.TrimSuffix(unitPriceString, unitPriceUnit)
		unitPrice, err := strconv.Atoi(unitPriceString)
		if err != nil {
			log.Info("Unit Price is not integer.")
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
					log.Info("Can't parse the gross area.")
				}
				house.GrossArea = grossArea

			case "户型结构":
				house.Structure = strings.TrimPrefix(el.Text, label)
			case "套内面积":
				netAreaString := strings.TrimSuffix(el.Text, "㎡")
				netAreaString = strings.TrimPrefix(netAreaString, label)
				netArea, err := strconv.ParseFloat(netAreaString, 2)
				if err != nil {
					log.Info("Can't parse the net area.")
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
		w.Write(house.toStringSlice())
		log.Info("Appending house:", house)
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
