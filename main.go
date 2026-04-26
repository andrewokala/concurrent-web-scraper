package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/gocolly/colly"
)

// ScrapeResult holds the result of scraping a single URL
type ScrapeResult struct {
	URL     string
	Title   string
	Error   error
	Success bool
}

// ScraperConfig holds configuration for the scraper
type ScraperConfig struct {
	MaxConcurrency int
	RequestTimeout time.Duration
	RetryCount     int
}

// Scraper manages concurrent scraping operations
type Scraper struct {
	config     ScraperConfig
	results    []ScrapeResult
	resultsMux sync.Mutex
	wg         sync.WaitGroup
	semaphore  chan struct{}
}

// NewScraper creates a new scraper with the given configuration
func NewScraper(config ScraperConfig) *Scraper {
	return &Scraper{
		config:    config,
		results:   make([]ScrapeResult, 0),
		semaphore: make(chan struct{}, config.MaxConcurrency),
	}
}

// scrapeURL scrapes a single URL and extracts the page title
func (s *Scraper) scrapeURL(url string, retryCount int) ScrapeResult {
	result := ScrapeResult{
		URL:     url,
		Success: false,
	}

	// Create a new collector for each URL
	c := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	)

	// Set timeout
	c.SetRequestTimeout(s.config.RequestTimeout)

	// Handle request
	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("[INFO] Scraping: %s (Attempt %d)\n", r.URL, retryCount+1)
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		result.Error = err
		fmt.Printf("[ERROR] Failed to scrape %s: %v\n", r.Request.URL, err)
	})

	// Extract title - common title selectors
	c.OnHTML("title", func(h *colly.HTMLElement) {
		result.Title = h.Text
		result.Success = true
	})

	// Fallback: try h1 if title not found
	c.OnHTML("h1", func(h *colly.HTMLElement) {
		if result.Title == "" {
			result.Title = h.Text
			result.Success = true
		}
	})

	// Visit the URL
	err := c.Visit(url)
	if err != nil {
		result.Error = err
		return result
	}

	// Wait for async operations to complete
	c.Wait()
	return result
}

// scrapeURLWithRetry attempts to scrape a URL with retry logic
func (s *Scraper) scrapeURLWithRetry(url string) ScrapeResult {
	var result ScrapeResult
	
	for i := 0; i <= s.config.RetryCount; i++ {
		result = s.scrapeURL(url, i)
		
		if result.Success {
			fmt.Printf("[SUCCESS] Scraped %s - Title: %s\n", url, result.Title)
			return result
		}
		
		if i < s.config.RetryCount {
			fmt.Printf("[RETRY] Attempt %d failed for %s, retrying...\n", i+1, url)
			time.Sleep(time.Second * 2) // Wait before retry
		}
	}
	
	fmt.Printf("[FAILED] All retry attempts failed for %s\n", url)
	return result
}

// ScrapeURLs scrapes a list of URLs concurrently with rate limiting
func (s *Scraper) ScrapeURLs(urls []string) []ScrapeResult {
	fmt.Printf("Starting to scrape %d URLs with max concurrency %d\n", len(urls), s.config.MaxConcurrency)
	
	for _, url := range urls {
		s.wg.Add(1)
		
		// Launch goroutine for each URL
		go func(urlToScrape string) {
			defer s.wg.Done()
			
			// Acquire semaphore token (limits concurrency)
			s.semaphore <- struct{}{}
			defer func() { <-s.semaphore }()
			
			// Scrape the URL
			result := s.scrapeURLWithRetry(urlToScrape)
			
			// Store result
			s.resultsMux.Lock()
			s.results = append(s.results, result)
			s.resultsMux.Unlock()
		}(url)
	}
	
	// Wait for all goroutines to complete
	s.wg.Wait()
	
	return s.results
}

// PrintSummary prints a summary of scraping results
func (s *Scraper) PrintSummary() {
	fmt.Println("\n" + "="*50)
	fmt.Println("SCRAPING SUMMARY")
	fmt.Println("="*50)
	
	successCount := 0
	failCount := 0
	
	for _, result := range s.results {
		if result.Success {
			successCount++
			fmt.Printf("✓ %s\n  Title: %s\n", result.URL, result.Title)
		} else {
			failCount++
			fmt.Printf("✗ %s\n  Error: %v\n", result.URL, result.Error)
		}
		fmt.Println()
	}
	
	fmt.Printf("Total URLs: %d\n", len(s.results))
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failCount)
	fmt.Printf("Success Rate: %.2f%%\n", float64(successCount)/float64(len(s.results))*100)
}

func main() {
	// Define the list of URLs to scrape
	urls := []string{
		"https://quotes.toscrape.com/",
		"https://quotes.toscrape.com/page/2/",
		"https://quotes.toscrape.com/page/3/",
		"https://httpbin.org/html", // This will work
		"https://httpbin.org/status/404", // This will fail (404 error)
		"https://httpbin.org/status/500", // This will fail (500 error)
		"https://example.com/",
		"https://example.org/",
	}
	
	// Configure the scraper
	scraperConfig := ScraperConfig{
		MaxConcurrency: 3,           // Maximum 3 concurrent requests
		RequestTimeout: 30 * time.Second, // 30 seconds timeout
		RetryCount:     2,           // Retry failed requests up to 2 times
	}
	
	// Create and run the scraper
	scraper := NewScraper(scraperConfig)
	results := scraper.ScrapeURLs(urls)
	
	// Print results
	_ = results // results already stored in scraper
	scraper.PrintSummary()
	
	// Example of accessing individual results
	fmt.Println("\nDetailed Results:")
	for i, result := range results {
		if result.Success {
			fmt.Printf("%d. [SUCCESS] %s -> \"%s\"\n", i+1, result.URL, result.Title)
		} else {
			fmt.Printf("%d. [FAILED] %s -> Error: %v\n", i+1, result.URL, result.Error)
		}
	}
}

// Helper function to create repeated strings (for the summary separator)
func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}