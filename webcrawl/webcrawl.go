package webcrawl

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"
)

type CrawlResult struct {
	Content      string
	CrawledURLs  []string
	PagesCrawled int
	PageErrors   map[string]string
	Links        Links
}

type CrawlOptions struct {
	Timeout          time.Duration
	UserAgent        string
	RemoveNavigation bool
	RemoveFooter     bool
	RemoveHeader     bool
	RemovePopups     bool
	ExtractMainOnly  bool
	FollowRedirects  bool
}

type LinkData struct {
	Href       string `json:"href"`
	Text       string `json:"text"`
	BaseDomain string `json:"base_domain"`
}

type Links struct {
	Internal []LinkData `json:"internal"`
	External []LinkData `json:"external"`
}

func DefaultCrawlOptions() *CrawlOptions {
	return &CrawlOptions{
		Timeout:          30 * time.Second,
		UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		RemoveNavigation: true,
		RemoveFooter:     true,
		RemoveHeader:     true,
		RemovePopups:     true,
		ExtractMainOnly:  true,
		FollowRedirects:  true,
	}
}

func CrawlWebsite(targetURL string, options *CrawlOptions) (*CrawlResult, error) {
	if options == nil {
		options = DefaultCrawlOptions()
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: options.Timeout,
	}

	// Create request
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", options.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK status code: %d", resp.StatusCode)
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Clean the document
	if options.RemovePopups {
		removePopupsAndOverlays(doc)
	}
	if options.RemoveNavigation {
		removeNavigationElements(doc)
	}
	if options.RemoveHeader {
		removeHeaderElements(doc)
	}
	if options.RemoveFooter {
		removeFooterElements(doc)
	}

	// Remove other unwanted elements
	removeUnwantedElements(doc)

	// Extract content
	var content string
	var extractedLinks Links

	if options.ExtractMainOnly {
		// Use go-readability for main content extraction
		content, extractedLinks, err = extractMainContentWithReadability(doc, targetURL)
		if err != nil {
			// Fallback to manual extraction if readability fails
			content, extractedLinks = extractContentManually(doc, targetURL)
		}
	} else {
		content, extractedLinks = extractContentManually(doc, targetURL)
	}

	result := &CrawlResult{
		Content:      content,
		CrawledURLs:  []string{targetURL},
		PagesCrawled: 1,
		PageErrors:   make(map[string]string),
		Links:        extractedLinks,
	}

	return result, nil
}

func removePopupsAndOverlays(doc *goquery.Document) {
	// Common popup and overlay selectors
	popupSelectors := []string{
		// Cookie consent banners
		"[class*='cookie']", "[id*='cookie']",
		"[class*='consent']", "[id*='consent']",
		"[class*='gdpr']", "[id*='gdpr']",
		"[class*='privacy']", "[id*='privacy']",

		// Modal overlays
		"[class*='modal']", "[id*='modal']",
		"[class*='overlay']", "[id*='overlay']",
		"[class*='popup']", "[id*='popup']",
		"[class*='lightbox']", "[id*='lightbox']",

		// Newsletter signups
		"[class*='newsletter']", "[id*='newsletter']",
		"[class*='subscribe']", "[id*='subscribe']",
		"[class*='signup']", "[id*='signup']",

		// Social sharing overlays
		"[class*='share-overlay']", "[id*='share-overlay']",
		"[class*='social-overlay']", "[id*='social-overlay']",

		// Common overlay patterns
		".overlay", ".modal", ".popup", ".lightbox",
		"#overlay", "#modal", "#popup", "#lightbox",
		".fixed", "[style*='position: fixed']",
		"[style*='z-index']",
	}

	for _, selector := range popupSelectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			// Check if element has high z-index (likely overlay)
			style, exists := s.Attr("style")
			if exists && strings.Contains(style, "z-index") {
				s.Remove()
				return
			}

			// Check classes/IDs for popup indicators
			class, _ := s.Attr("class")
			id, _ := s.Attr("id")
			text := strings.ToLower(s.Text())

			if isPopupElement(class, id, text) {
				s.Remove()
			}
		})
	}
}

func removeNavigationElements(doc *goquery.Document) {
	navSelectors := []string{
		"nav", "navigation", ".nav", ".navigation",
		"[role='navigation']", "[class*='nav']", "[id*='nav']",
		".menu", "[class*='menu']", "[id*='menu']",
		".sidebar", "[class*='sidebar']", "[id*='sidebar']",
		".breadcrumb", "[class*='breadcrumb']", "[id*='breadcrumb']",
	}

	for _, selector := range navSelectors {
		doc.Find(selector).Remove()
	}
}

func removeHeaderElements(doc *goquery.Document) {
	headerSelectors := []string{
		"header", ".header", "#header",
		"[class*='header']", "[id*='header']",
		".top-bar", ".topbar", "[class*='top-bar']",
		".site-header", "[class*='site-header']",
	}

	for _, selector := range headerSelectors {
		doc.Find(selector).Remove()
	}
}

func removeFooterElements(doc *goquery.Document) {
	footerSelectors := []string{
		"footer", ".footer", "#footer",
		"[class*='footer']", "[id*='footer']",
		".site-footer", "[class*='site-footer']",
		".bottom", "[class*='bottom']",
	}

	for _, selector := range footerSelectors {
		doc.Find(selector).Remove()
	}
}

func removeUnwantedElements(doc *goquery.Document) {
	unwantedSelectors := []string{
		// Scripts and styles
		"script", "style", "noscript",

		// Ads and tracking
		".ad", ".ads", "[class*='advertisement']",
		"[class*='google-ad']", "[class*='adsense']",
		"iframe[src*='doubleclick']", "iframe[src*='googlesyndication']",

		// Social widgets
		".social-widget", "[class*='social-share']",
		".facebook-widget", ".twitter-widget",

		// Comments (often not main content)
		".comments", "[class*='comment']", "[id*='comment']",
		".disqus", "[class*='disqus']",

		// Related/recommended content
		".related", "[class*='related']",
		".recommended", "[class*='recommended']",
		".suggestions", "[class*='suggestions']",
	}

	for _, selector := range unwantedSelectors {
		doc.Find(selector).Remove()
	}
}

func isPopupElement(class, id, text string) bool {
	popupKeywords := []string{
		"cookie", "consent", "gdpr", "privacy", "modal", "overlay",
		"popup", "newsletter", "subscribe", "signup", "lightbox",
	}

	combined := strings.ToLower(class + " " + id + " " + text)
	for _, keyword := range popupKeywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}
	return false
}

func extractMainContentWithReadability(doc *goquery.Document, targetURL string) (string, Links, error) {
	// Convert goquery document back to HTML string for readability
	html, err := doc.Html()
	if err != nil {
		return "", Links{}, err
	}

	// Use go-readability to extract main content
	article, err := readability.FromReader(strings.NewReader(html), &url.URL{})
	if err != nil {
		return "", Links{}, err
	}

	// Parse the extracted content to get links
	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		return "", Links{}, err
	}

	links := extractLinks(contentDoc.Selection, targetURL)

	// Convert HTML to clean text/markdown-like format
	cleanContent := htmlToCleanText(contentDoc.Selection)

	return cleanContent, links, nil
}

func extractContentManually(doc *goquery.Document, targetURL string) (string, Links) {
	// Try to find main content area
	mainSelectors := []string{
		"main", "[role='main']", ".main", "#main",
		"article", ".article", "#article",
		".content", "#content", ".post", "#post",
		".entry", "#entry", ".page-content",
		"[class*='main-content']", "[class*='page-content']",
	}

	var contentSelection *goquery.Selection
	for _, selector := range mainSelectors {
		if selection := doc.Find(selector).First(); selection.Length() > 0 {
			contentSelection = selection
			break
		}
	}

	// If no main content found, use body but remove unwanted elements
	if contentSelection == nil {
		contentSelection = doc.Find("body")
	}

	links := extractLinks(contentSelection, targetURL)
	content := htmlToCleanText(contentSelection)

	return content, links
}

func extractLinks(selection *goquery.Selection, baseURL string) Links {
	var internal, external []LinkData

	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		return Links{Internal: internal, External: external}
	}

	selection.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		text := strings.TrimSpace(s.Text())
		if text == "" {
			text = href
		}

		// Resolve relative URLs
		linkURL, err := url.Parse(href)
		if err != nil {
			return
		}

		resolvedURL := baseURLParsed.ResolveReference(linkURL)

		linkData := LinkData{
			Href:       resolvedURL.String(),
			Text:       text,
			BaseDomain: resolvedURL.Host,
		}

		// Determine if internal or external
		if resolvedURL.Host == baseURLParsed.Host {
			internal = append(internal, linkData)
		} else {
			external = append(external, linkData)
		}
	})

	return Links{Internal: internal, External: external}
}

func htmlToCleanText(selection *goquery.Selection) string {
	var result strings.Builder

	selection.Contents().Each(func(i int, s *goquery.Selection) {
		if goquery.NodeName(s) == "#text" {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				result.WriteString(text)
				result.WriteString(" ")
			}
		} else {
			// Handle different HTML elements
			tagName := goquery.NodeName(s)
			switch tagName {
			case "h1", "h2", "h3", "h4", "h5", "h6":
				level := tagName[1:] // Extract number
				prefix := strings.Repeat("#", parseInt(level))
				result.WriteString(fmt.Sprintf("\n\n%s %s\n\n", prefix, strings.TrimSpace(s.Text())))
			case "p":
				result.WriteString(fmt.Sprintf("\n%s\n", strings.TrimSpace(s.Text())))
			case "br":
				result.WriteString("\n")
			case "li":
				result.WriteString(fmt.Sprintf("- %s\n", strings.TrimSpace(s.Text())))
			case "blockquote":
				result.WriteString(fmt.Sprintf("\n> %s\n", strings.TrimSpace(s.Text())))
			case "code":
				result.WriteString(fmt.Sprintf("`%s`", strings.TrimSpace(s.Text())))
			case "pre":
				result.WriteString(fmt.Sprintf("\n```\n%s\n```\n", strings.TrimSpace(s.Text())))
			default:
				// For other elements, just extract text
				if text := strings.TrimSpace(s.Text()); text != "" {
					result.WriteString(text)
					result.WriteString(" ")
				}
			}
		}
	})

	// Clean up the result
	content := result.String()

	// Remove excessive whitespace
	multipleSpaces := regexp.MustCompile(`\s+`)
	content = multipleSpaces.ReplaceAllString(content, " ")

	// Remove excessive newlines
	multipleNewlines := regexp.MustCompile(`\n\s*\n\s*\n`)
	content = multipleNewlines.ReplaceAllString(content, "\n\n")

	return strings.TrimSpace(content)
}

func parseInt(s string) int {
	switch s {
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "5":
		return 5
	case "6":
		return 6
	default:
		return 1
	}
}
