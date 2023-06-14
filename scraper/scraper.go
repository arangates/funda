package scraper

import (
	"fmt"
	"strings"

	//	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

type House struct {
	Price         int
	Address       string
	PostCode      string
	City          string
	Link          string
	Area          int
	Year          int
	EnergyLabel   string
	Published     string
	Isolation     string
	ExtraPayments string
}

var (
	// html objects from Funda
	fundaHtmlSearchPages = ".search-output-result-count span"

	// regular exprations
	yearRegex, _         = regexp.Compile(`[0-9]{4}`)
	energyLabelRegex, _  = regexp.Compile(`[A-G]{1}[+]*`)
	numberRegex, _       = regexp.Compile(`[0-9\.]+`)
	postCode, _          = regexp.Compile(`[0-9]{4}`)
	postCodeLetters, _   = regexp.Compile(`[0-9]{4} [A-Z]{2}`)
	postCodeCityRegex, _ = regexp.Compile(`([0-9]{4} [A-Z]{2})( )(\w+)`)
	space, _             = regexp.Compile(`\s+`)
)

// check post codes if its in the filter list
func postCodeFilter(pc string, postCodesAllowed *[]string) bool {

	for _, p := range *postCodesAllowed {

		// check for exact match first
		if postCodeLetters.MatchString(p) {
			if pc == p {
				return true
			}
		}

		// check for main code match
		if postCode.MatchString(p) {
			if strings.Split(pc, " ")[0] == p {
				return true
			}
		}

	}

	return false
}

// just make a request and
func ScrapePageContent(url, fakeUserAgent string) (*http.Response, error) {

	log.Infof("Scraping %s\n", url)

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}

	req.Header.Set("User-Agent", fakeUserAgent)

	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}

	//	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
		return nil, nil
	}

	return res, nil
}

// parsing search page
func GetFundaSearchResults(url string, result *[]House, userAgent, searchUrl *string, scrapeDelay *int, postCodes *[]string) {

	res, err := ScrapePageContent(url, *userAgent)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// do parsing search page
	doc.Find(".search-result").Each(func(i int, s *goquery.Selection) {

		var h House

		h.Link, _ = s.Find(".search-result__header a").Attr("href")
		h.Link = "https://www.funda.nl" + h.Link

		// getting post code and apply filter if there is one
		if len(*postCodes) > 0 {
			h.PostCode = postCodeCityRegex.FindStringSubmatch(s.Find(".search-result__header-subtitle").Text())[1]
			if postCodeFilter(h.PostCode, postCodes) {
				*result = append(*result, h)
			}
		} else {
			*result = append(*result, h)
		}

	})
}

func GetHouseDetail(h *House, userAgent, searchUrl *string, scrapeDelay *int) {
	url := h.Link

	res, err := ScrapePageContent(url, *userAgent)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// header parsing
	doc.Find(".object-header__details").Each(func(i int, s *goquery.Selection) {

		h.Address = s.Find(".object-header__title").Text()
		h.Price, _ = strconv.Atoi(strings.Replace(numberRegex.FindString(s.Find(".object-header__price").Text()), ".", "", -1))
		h.PostCode = postCodeCityRegex.FindStringSubmatch(s.Find(".object-header__subtitle").Text())[1]
		h.City = postCodeCityRegex.FindStringSubmatch(s.Find(".object-header__subtitle").Text())[3]

	})

	// apartment properties fields
	doc.Find(".object-kenmerken-list dt").Each(func(i int, s *goquery.Selection) {
		nf := s.NextFiltered("dd")

		key := space.ReplaceAllString(s.Text(), " ")
		value := space.ReplaceAllString(nf.Text(), " ")

		// remove spaces
		value = strings.TrimSpace(value)

		switch key {
		case "Wonen": // square meters
			h.Area, _ = strconv.Atoi(numberRegex.FindString(value))
		case "Energielabel": // energy label
			h.EnergyLabel = energyLabelRegex.FindString(value)
		case "Bouwjaar": // costruction year
			h.Year, _ = strconv.Atoi(yearRegex.FindString(value))
		case "Aangeboden sinds": // publication date
			h.Published = value
		case "Isolatie": // Isolation features like double glass
			h.Isolation = value
		case "Bijdrage VvE": // Extra payments
			h.ExtraPayments = value
		default:
			//
		}

		// debug
		//fmt.Println("KEY: ", key, "VALUE: ", value)

	})

	// wait a bit to not overload Funda
	time.Sleep(time.Duration(*scrapeDelay) * time.Millisecond)
}

func WriteToExcel(results []House) error {
	f := excelize.NewFile()

	// Create a new sheet in the Excel file
	index := f.NewSheet("Sheet1")

	// Set the headers in the first row
	headers := []string{"Price", "Address", "PostCode", "City", "Link", "Area", "Year", "EnergyLabel", "Published", "Isolation", "ExtraPayments"}
	for i, header := range headers {
		colName := string('A' + i)
		f.SetCellValue("Sheet1", colName+"1", header)
	}
	fmt.Println(results)
	// Write the data to the Excel file
	for i, house := range results {
		row := i + 2 // Start from the second row
		fmt.Println(house)
		f.SetCellValue("Sheet1", "A"+strconv.Itoa(row), house.Price)
		f.SetCellValue("Sheet1", "B"+strconv.Itoa(row), house.Address)
		f.SetCellValue("Sheet1", "C"+strconv.Itoa(row), house.PostCode)
		f.SetCellValue("Sheet1", "D"+strconv.Itoa(row), house.City)
		f.SetCellValue("Sheet1", "E"+strconv.Itoa(row), house.Link)
		f.SetCellValue("Sheet1", "F"+strconv.Itoa(row), house.Area)
		f.SetCellValue("Sheet1", "G"+strconv.Itoa(row), house.Year)
		f.SetCellValue("Sheet1", "H"+strconv.Itoa(row), house.EnergyLabel)
		f.SetCellValue("Sheet1", "I"+strconv.Itoa(row), house.Published)
		f.SetCellValue("Sheet1", "J"+strconv.Itoa(row), house.Isolation)
		f.SetCellValue("Sheet1", "K"+strconv.Itoa(row), house.ExtraPayments)
	}

	// Set the active sheet in the Excel file
	f.SetActiveSheet(index)

	// Save the Excel file
	err := f.SaveAs("scraped_data.xlsx")
	if err != nil {
		return err
	}

	return nil
}

func RunScraper(results *[]House, userAgent, searchUrl *string, scrapeDelay *int, postCodes *[]string) {

	res, err := ScrapePageContent(*searchUrl, *userAgent)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)

		if err != nil {
		log.Fatal(err)
	}

	// find amount of elements in search
	numberRegex, _ := regexp.Compile("[0-9]+")
	pages, _ := strconv.Atoi(numberRegex.FindString(doc.Find(fundaHtmlSearchPages).Text()))
	resultsOnPage := 15

	cicles := 0
	if pages%resultsOnPage == 0 {
		cicles = (pages / resultsOnPage)
	} else {
		cicles = (pages / resultsOnPage) + 1
	}

	log.Infof("Found %v results on %v pages\n", pages, cicles)

	log.Info("Collecting all refenrences to the house detail pages")
	for i := 1; i <= cicles; i++ {
		GetFundaSearchResults(fmt.Sprintf(*searchUrl+"p%d/", i), results, userAgent, searchUrl, scrapeDelay, postCodes)
	}

	log.Infof("%v results left after applying filter \n", len(*results))

	log.Info("Collecting data for each particular house")
	for i, _ := range *results {
		GetHouseDetail(&(*results)[i], userAgent, searchUrl, scrapeDelay)
	}
	hel := WriteToExcel(*results)
	if err != nil {
		log.Fatal(hel)
	}

}
