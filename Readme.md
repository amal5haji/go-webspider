# Go Web Spider

A Go library and command-line tool to crawl websites and extract clean, textual content. It recursively crawls pages starting from a given URL, removes common clutter (navigation, ads, popups), extracts the main article text, and provides the aggregated content.

## Features

*   **Recursive Crawling:** Crawls multiple pages up to a specified depth and page limit.
*   **Content Cleaning:** Removes navigation bars, footers, headers, popups, ads, and other non-essential elements.
*   **Main Content Extraction:** Uses readability logic to focus on the primary article text.
*   **Flexible Output:** Get results directly in code or output to a file via the CLI.
*   **Configurable:** Control crawl depth, number of pages, concurrency, delays, and timeouts.
*   **Link Discovery:** Reports internal links found and identifies downloadable files (PDFs, Docs, etc.).

## Installation

```bash
go get github.com/amal5haji/go-webspider
```
*(Replace `amal5haji` with your actual GitHub username)*

Ensure you have Go installed (version 1.16 or later is recommended).

## Usage

### 1. As a Command-Line Tool (CLI Mode)

Build the CLI tool:

```bash
# Clone the repo if you haven't
# git clone https://github.com/amal5haji/go-webspider.git
# cd go-webspider

# Initialize module (if needed)
go mod tidy

# Build the binary
go build -o go-webspider .
```

Run the tool:

```bash
./go-webspider -url <TARGET_URL> [OPTIONS]
```

**Options:**

*   `-url string`: The starting URL for the crawl. **(Required)**
*   `-max-pages int`: Maximum number of pages to crawl. (Default 100)
*   `-max-depth int`: Maximum depth to crawl from the starting URL. (Default 3)
*   `-timeout duration`: Timeout for fetching a single page (e.g., 30s, 1m). (Default 30s)
*   `-concurrency int`: Number of concurrent crawlers. (Default 5)
*   `-delay duration`: Delay between requests made by each crawler (e.g., 1s, 500ms). (Default 1s)
*   `-output string`: Path to the output text file. If not provided, output is written to standard output (stdout).

**CLI Example:**

Crawl `https://blog.golang.org` up to 20 pages and 2 levels deep, saving the text content to `golang_blog.txt`:

```bash
./go-webspider -url https://blog.golang.org -max-pages 20 -max-depth 2 -output golang_blog.txt
```

### 2. As a Go Library (Package Mode)

You can integrate the crawling functionality directly into your Go application.

**Import the package:**

```go
import "github.com/amal5haji/go-webspider/webspider"
```

**Example Code:**

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/amal5haji/go-webspider/webspider" 
)

func main() {
	// Define the URL to crawl
	targetURL := "https://example.com"

	// (Optional) Configure crawl options
	options := &webspider.SpiderOptions{
		MaxPages:       5,          // Crawl up to 5 pages
		MaxDepth:       2,          // Go 2 levels deep
		CrawlSubDomain: true,       // Crawl subdomains too (if applicable)
		Timeout:        20 * time.Second, // Timeout per page
		Concurrency:    3,          // Use 3 concurrent workers
		DelayBetween:   500 * time.Millisecond, // Wait 0.5s between requests per worker
	}

	// Perform the crawl
	fmt.Printf("Starting crawl of %s...\n", targetURL)
	result, err := webspider.SpiderWebsite(targetURL, options)
	if err != nil {
		log.Fatalf("Error crawling website: %v", err)
	}

	// Access the results
	fmt.Printf("Crawl completed successfully!\n")
	fmt.Printf("Total Pages Processed: %d\n", result.TotalPages)
	fmt.Printf("Successful Pages: %d\n", result.SuccessfulPages)
	fmt.Printf("Processing Time: %v\n", result.ProcessingTime)
	fmt.Println("--- Extracted Content ---")
	fmt.Println(result.Content) // The aggregated text content

	// Optional: Inspect other data
	// fmt.Printf("Crawled URLs: %v\n", result.CrawledURLs)
	// fmt.Printf("Failed Pages: %v\n", result.FailedPages)
	// fmt.Printf("Detected File URLs: %v\n", result.DetectedFileUrls)
}
```

**Running the Example:**

Save the code above as `example_usage.go` in your project or another Go module.

Make sure your `go.mod` includes the dependency:

```bash
go mod init your-example-project # Or your preferred module name
go get github.com/amal5haji/go-webspider # Replace with your actual path
go mod tidy
go run example_usage.go
```

Replace `amal5haji` with your actual GitHub username where the `go-webspider` repository is hosted.

## How It Works

1.  The `webspider` package manages the multi-page crawl logic, respecting depth and page limits.
2.  For each page, it calls the internal `webcrawl` package.
3.  The `webcrawl` package fetches the HTML, removes clutter (navigation, ads, popups), and attempts to extract the main textual content using readability techniques.
4.  Links found on each page are checked against the crawl rules (domain, depth) and added to the queue if appropriate.
5.  Content from all successfully crawled pages is aggregated.
6.  The final result includes the combined text, statistics (pages crawled, failed), and lists of crawled/failed URLs and detected files.

## Dependencies

Managed using Go modules (`go.mod`):

*   `github.com/PuerkitoBio/goquery`: HTML parsing and manipulation.
*   `github.com/go-shiori/go-readability`: Main content extraction from HTML.
*   `go.uber.org/zap`: (Used in the original code for logging; might be simplified or removed in the standalone version).

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## License

Apache 2.0