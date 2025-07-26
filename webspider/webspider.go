package webspider

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/amal5haji/go-webspider/webcrawl"

	"go.uber.org/zap"
)

type SpiderOptions struct {
	MaxPages       int
	MaxDepth       int
	CrawlSubDomain bool
	Timeout        time.Duration
	Concurrency    int
	DelayBetween   time.Duration
}

type SpiderResult struct {
	Content          string
	CrawledURLs      []string
	DetectedFileUrls []string
	TotalPages       int
	SuccessfulPages  int
	FailedPages      map[string]string
	ProcessingTime   time.Duration
}

type urlJob struct {
	url   string
	depth int
}

func DefaultSpiderOptions() *SpiderOptions {
	return &SpiderOptions{
		MaxPages:       100,
		MaxDepth:       3,
		CrawlSubDomain: true,
		Timeout:        30 * time.Second,
		Concurrency:    5,
		DelayBetween:   1 * time.Second,
	}
}

func SpiderWebsite(targetURL string, options *SpiderOptions) (*SpiderResult, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	logger.Debug("Starting spider for URL: ", zap.String("url", targetURL))
	if options == nil {
		options = DefaultSpiderOptions()
	}
	if options.MaxPages <= 0 {
		options.MaxPages = 1
	}

	startTime := time.Now()

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	result := &SpiderResult{
		Content:          "",
		CrawledURLs:      []string{},
		DetectedFileUrls: []string{},
		FailedPages:      make(map[string]string),
	}

	visitedURLs := make(map[string]bool)
	var mu sync.Mutex

	urlJobs := make(chan urlJob, options.MaxPages*2)
	urlJobs <- urlJob{url: targetURL, depth: 0}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, options.Concurrency)
	activeWorkers := 0
	var workerMu sync.Mutex

	for {
		select {
		case job := <-urlJobs:
			if result.TotalPages >= options.MaxPages || job.depth > options.MaxDepth {
				continue
			}

			mu.Lock()
			if visitedURLs[job.url] {
				mu.Unlock()
				continue
			}
			visitedURLs[job.url] = true
			result.TotalPages++
			mu.Unlock()

			semaphore <- struct{}{}
			wg.Add(1)
			workerMu.Lock()
			activeWorkers++
			workerMu.Unlock()

			go func(currentURL string, currentDepth int) {
				defer wg.Done()
				defer func() { <-semaphore }()
				defer func() {
					workerMu.Lock()
					activeWorkers--
					workerMu.Unlock()
				}()

				logger.Debug("Processing URL",
					zap.String("url", currentURL),
					zap.Int("depth", currentDepth),
				)

				if options.DelayBetween > 0 {
					time.Sleep(options.DelayBetween)
				}

				crawlOptions := &webcrawl.CrawlOptions{
					Timeout: options.Timeout,
				}

				crawlResult, err := webcrawl.CrawlWebsite(currentURL, crawlOptions)
				if err != nil {
					mu.Lock()
					result.FailedPages[currentURL] = err.Error()
					mu.Unlock()
					logger.Debug("Failed to crawl URL",
						zap.String("url", currentURL),
						zap.Error(err),
					)
					return
				}

				mu.Lock()
				// Remove markdown links and keep only the text
				cleanedContent := removeMarkdownLinks(crawlResult.Content)
				result.Content += fmt.Sprintf("\n\n# URL: %s\n\n%s", currentURL, cleanedContent)

				result.CrawledURLs = append(result.CrawledURLs, currentURL)
				result.SuccessfulPages++
				mu.Unlock()

				logger.Debug("Successfully crawled URL",
					zap.String("url", currentURL),
					zap.Int("depth", currentDepth),
				)

				if currentDepth < options.MaxDepth {
					crawlableLinks, fileLinks := extractLinks(crawlResult, currentURL, parsedURL, options.CrawlSubDomain)

					logger.Debug("Extracted links",
						zap.String("url", currentURL),
						zap.Int("depth", currentDepth),
						zap.Int("crawlable_links", len(crawlableLinks)),
						zap.Int("file_links", len(fileLinks)),
					)

					mu.Lock()
					result.DetectedFileUrls = append(result.DetectedFileUrls, fileLinks...)
					mu.Unlock()

					for _, link := range crawlableLinks {
						select {
						case urlJobs <- urlJob{url: link, depth: currentDepth + 1}:
							logger.Debug("Added link to queue",
								zap.String("link", link),
								zap.Int("depth", currentDepth+1),
							)
						default:
							logger.Debug("Channel full, skipping link",
								zap.String("link", link),
							)
						}
					}
				}
			}(job.url, job.depth)

		case <-time.After(2 * time.Second):
			workerMu.Lock()
			currentActiveWorkers := activeWorkers
			workerMu.Unlock()
			if currentActiveWorkers == 0 && len(urlJobs) == 0 {
				logger.Debug("No active workers and no pending jobs, finishing crawl")
				goto done
			}
			continue
		}

		if result.TotalPages >= options.MaxPages {
			logger.Debug("Reached maximum pages limit",
				zap.Int("max_pages", options.MaxPages),
				zap.Int("total_pages", result.TotalPages),
			)
			break
		}
	}

done:
	wg.Wait()

	mu.Lock()
	if len(result.DetectedFileUrls) > 0 {
		uniqueFileUrls := make(map[string]bool)
		for _, fileUrl := range result.DetectedFileUrls {
			uniqueFileUrls[fileUrl] = true
		}

		finalFileList := make([]string, 0, len(uniqueFileUrls))
		for fileUrl := range uniqueFileUrls {
			finalFileList = append(finalFileList, fileUrl)
		}

		logger.Info("Detected file URLs that were not crawled",
			zap.Strings("files", finalFileList),
		)
	}
	mu.Unlock()

	result.ProcessingTime = time.Since(startTime)

	return result, nil
}

func extractLinks(crawlResult *webcrawl.CrawlResult, baseURL string, parsedBaseURL *url.URL, crawlSubDomain bool) (crawlableLinks []string, fileLinks []string) {
	crawlableLinkSet := make(map[string]bool)
	fileLinkSet := make(map[string]bool)

	// Process internal links from the crawl response
	for _, link := range crawlResult.Links.Internal {
		href := strings.TrimSpace(link.Href)
		if href == "" {
			continue
		}

		processLinkFromResponse(href, link.Text, baseURL, parsedBaseURL, crawlSubDomain, crawlableLinkSet, fileLinkSet)
	}

	// Convert sets to slices
	for link := range crawlableLinkSet {
		crawlableLinks = append(crawlableLinks, link)
	}
	for link := range fileLinkSet {
		fileLinks = append(fileLinks, link)
	}

	return crawlableLinks, fileLinks
}

func processLinkFromResponse(href, text, baseURL string, parsedBaseURL *url.URL, crawlSubDomain bool, crawlableLinkSet, fileLinkSet map[string]bool) {
	if href == "" || strings.HasPrefix(href, "#") {
		return
	}

	// Sanitize the URL
	sanitizedLink := sanitizeURL(href)
	if sanitizedLink == "" {
		return
	}

	resolvedURL, err := url.Parse(sanitizedLink)
	if err != nil {
		return
	}

	// Handle relative URLs
	if !resolvedURL.IsAbs() {
		base, err := url.Parse(baseURL)
		if err != nil {
			return
		}
		resolvedURL = base.ResolveReference(resolvedURL)
	}

	// Check if we should crawl this URL
	if shouldCrawlURL(resolvedURL, parsedBaseURL, crawlSubDomain) {
		resolvedURL.Fragment = ""
		cleanURL := resolvedURL.String()

		if isFileURL(resolvedURL) {
			fileLinkSet[cleanURL] = true
		} else {
			crawlableLinkSet[cleanURL] = true
		}
	}
}

var fileExtensions = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true,
	".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	".zip": true, ".rar": true, ".gz": true, ".tar": true,
	".svg": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
}

func isFileURL(u *url.URL) bool {
	// Check for patterns like 'download=1'
	if u.Query().Get("download") == "1" {
		return true
	}

	path := strings.ToLower(u.Path)
	query := strings.ToLower(u.RawQuery)

	// Check file extension in the path
	for ext := range fileExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	// Add heuristics for this specific site based on log analysis
	if strings.Contains(path, "/resource/") {
		return true
	}
	if strings.Contains(query, "item=form") || strings.Contains(query, "item=statute") {
		return true
	}

	return false
}

var sanitizeRegex = regexp.MustCompile(`^https?://[^\s")'\]}]+`)

func sanitizeURL(rawURL string) string {
	// Find the first valid-looking URL part and discard the rest.
	// This handles cases like "...?download=1)(pdf)" by stopping at the '('.
	return sanitizeRegex.FindString(strings.TrimSpace(rawURL))
}

func shouldCrawlURL(targetURL, baseURL *url.URL, crawlSubDomain bool) bool {
	targetHost := strings.ToLower(targetURL.Host)
	baseHost := strings.ToLower(baseURL.Host)

	if targetHost == baseHost {
		return true
	}

	if crawlSubDomain {
		targetHostClean := strings.TrimPrefix(targetHost, "www.")
		baseHostClean := strings.TrimPrefix(baseHost, "www.")

		if strings.HasSuffix(targetHostClean, "."+baseHostClean) ||
			strings.HasSuffix(baseHostClean, "."+targetHostClean) {
			return true
		}
	}

	return false
}

func removeMarkdownLinks(content string) string {
	markdownLinkRegex := regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	return markdownLinkRegex.ReplaceAllString(content, "$1")
}
