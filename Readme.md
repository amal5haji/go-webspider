# Go Web Spider

A Go library and command-line tool to crawl websites and extract **clean, main textual content**. It recursively crawls pages starting from a given URL, effectively **removes common clutter (navigation, ads, footers, headers, popups, cookie banners)**, and provides the aggregated primary content.

## Features

*   **Recursive Crawling:** Crawls multiple pages up to a specified depth and page limit.
*   **Smart Content Cleaning:**
    *   **Main Content Focus:** Extracts the primary article or text content using readability logic.
    *   **Clutter Removal:** Aggressively removes navigation bars, footers, headers, sidebars, ads, and social widgets.
    *   **Popup/Overlay Handling:** Identifies and removes common elements like cookie consent banners, newsletter signups, modals, and lightboxes.
*   **Flexible Output:** Get results directly in code or output to a file via the CLI.
*   **Configurable:** Control crawl depth, number of pages, concurrency, delays, and timeouts.
*   **Link Discovery:** Reports internal links found and identifies downloadable files (PDFs, Docs, etc.).

## Installation

```bash
go get github.com/amal5haji/go-webspider
```

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
go get github.com/amal5haji/go-webspider
go mod tidy
go run example_usage.go
```

## Architecture & Workflow

This diagram illustrates the core crawling and processing flow:

```
+-------------------+
|   Start URL       |
+-------------------+
          |
          v
+-------------------+     Yes
| URL Queue Empty?  | -----> [End Crawl]
+-------------------+
          | No
          v
+-------------------+     +----------------------+
| Get URL from      | --> | Fetch HTML (webcrawl)|
| Queue             |     +----------------------+
+-------------------+               |
                                    v
                    +---------------------------------+
                    | Clean HTML (Remove popups, ads, |
                    | nav, footers, etc.)             |
                    +---------------------------------+
                                    |
                                    v
                    +---------------------------------+
                    | Extract Main Content (Readability|
                    | or Manual Selection)            |
                    +---------------------------------+
                                    |
                                    v
                    +---------------------------------+
                    | Append Content to Result        |
                    | Add URL to Crawled List         |
                    +---------------------------------+
                                    |
                                    v
                    +---------------------------------+
                    | Extract Links from Cleaned Page |
                    +---------------------------------+
                                    |
                                    v
         +------------------------------------------------------+
         | For Each Link:                                       |
         |   - Is it within crawl depth & page limit?           |
         |   - Is it on the same domain/subdomain (if allowed)? |
         |   Yes --> Add to URL Queue                          |
         |   No  --> Discard / Log (File Links)                |
         +------------------------------------------------------+
                                    |
                                    v
                          [Loop Back to Queue Check]
```

1.  The process starts with a single URL placed in a queue.
2.  Worker goroutines (limited by `Concurrency`) pull URLs from the queue.
3.  Each worker fetches the HTML for its assigned URL using the internal `webcrawl` logic.
4.  The fetched HTML undergoes extensive cleaning to remove unwanted elements, including targeted removal of popups and overlays.
5.  The `webcrawl` package then attempts to extract only the *main* content of the page (e.g., the article body) using readability libraries or manual heuristics.
6.  The extracted text content is appended to the overall result.
7.  Links are extracted from the *cleaned* HTML to find new pages to crawl.
8.  Each discovered link is validated against the crawl rules (depth, domain, max pages). Valid links are added back to the central queue.
9.  The process repeats until the queue is empty or limits are reached.

## Dependencies

Managed using Go modules (`go.mod`):

*   `github.com/PuerkitoBio/goquery`: HTML parsing and manipulation.
*   `github.com/go-shiori/go-readability`: Main content extraction from HTML.
*   `go.uber.org/zap`: (Used in the original code for logging; might be simplified or removed in the standalone version).

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## License

Apache 2.0