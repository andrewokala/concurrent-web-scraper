package main

import (
	"fmt"

	"github.com/gocolly/colly"
)

func main() {

	scrapeURl := "https://quotes.toscrape.com/"

	c := colly.NewCollector(colly.AllowedDomains("www.quotes.toscrape.com", "quotes.toscrape.com"))

	c.OnRequest(func(r *colly.Request) {
		fmt.Println(fmt.Sprintf("Visiting %s", r.URl))
	})

	c.OnError(func(r *colly.Response, e error) {
		fmt.Printf("Error while scraping %s\n", e.Error())
	})

	c.OnHTML("h1.col-md-8", func(h *colly.HTMLElement) {
		fmt.Println(h.Text)
	})

	c.Visit(scrapeURl)
}