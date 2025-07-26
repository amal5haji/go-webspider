package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amal5haji/go-webspider/webspider"
)

func main() {
	var targetURL string
	var maxPages int
	var maxDepth int
	var timeout time.Duration
	var concurrency int
	var delay time.Duration
	var outputFile string

	flag.StringVar(&targetURL, "url", "", "Target URL to start crawling from")
	flag.IntVar(&maxPages, "max-pages", 100, "Maximum number of pages to crawl")
	flag.IntVar(&maxDepth, "max-depth", 3, "Maximum crawl depth")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "Timeout for individual page requests")
	flag.IntVar(&concurrency, "concurrency", 5, "Number of concurrent crawlers")
	flag.DurationVar(&delay, "delay", 1*time.Second, "Delay between requests per crawler")
	flag.StringVar(&outputFile, "output", "", "Output file path (default: stdout)")

	flag.Parse()

	if targetURL == "" {
		log.Fatal("Please provide a target URL using the -url flag")
	}

	options := &webspider.SpiderOptions{
		MaxPages:       maxPages,
		MaxDepth:       maxDepth,
		Timeout:        timeout,
		Concurrency:    concurrency,
		DelayBetween:   delay,
		CrawlSubDomain: true,
	}

	// Handle graceful shutdown on Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	fmt.Printf("Starting crawl of %s...\n", targetURL)
	startTime := time.Now()

	result, err := webspider.SpiderWebsite(targetURL, options)
	if err != nil {
		log.Fatalf("Crawl failed: %v", err)
	}

	// Check if context was cancelled during crawl
	select {
	case <-ctx.Done():
		log.Println("Crawl was interrupted.")
		os.Exit(1) // Or handle differently
	default:
	}

	duration := time.Since(startTime)
	fmt.Fprintf(os.Stderr, "Crawl completed in %v\n", duration)
	fmt.Fprintf(os.Stderr, "Pages crawled successfully: %d\n", result.SuccessfulPages)
	fmt.Fprintf(os.Stderr, "Pages failed: %d\n", len(result.FailedPages))

	var output *os.File = os.Stdout
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file '%s': %v", outputFile, err)
		}
		defer file.Close()
		output = file
	}

	_, err = fmt.Fprint(output, result.Content)
	if err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	// Optionally log failed pages to stderr or a separate file
	if len(result.FailedPages) > 0 {
		fmt.Fprintf(os.Stderr, "\nFailed Pages:\n")
		for url, err := range result.FailedPages {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", url, err)
		}
	}
	// Optionally log detected files
	if len(result.DetectedFileUrls) > 0 {
		fmt.Fprintf(os.Stderr, "\nDetected File URLs (not crawled):\n")
		for _, fileURL := range result.DetectedFileUrls {
			fmt.Fprintf(os.Stderr, "  %s\n", fileURL)
		}
	}
}
