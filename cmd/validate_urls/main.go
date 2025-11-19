package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/oliver/lan_smite_teams_bot/internal/smite"
)

// Global HTTP client with connection pooling
var httpClient *http.Client

func init() {
	// Create a transport with optimized settings
	transport := &http.Transport{
		// Connection pooling settings
		MaxIdleConns:        100,              // Maximum number of idle connections across all hosts
		MaxIdleConnsPerHost: 10,               // Maximum number of idle connections per host
		IdleConnTimeout:     90 * time.Second, // How long an idle connection is kept alive
		
		// Connection dialer with timeout
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		
		// Connection timeouts
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		
		// TLS configuration with verification disabled
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		
		// Enable compression and HTTP/2
		DisableCompression: false,
		ForceAttemptHTTP2:  true, // Attempt HTTP/2 on HTTPS connections
	}

	// Create the global HTTP client
	httpClient = &http.Client{
		Timeout:   15 * time.Second, // Overall request timeout
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow redirects, but limit them
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// checkURL performs a HTTP HEAD request to check if a URL exists
func checkURL(url string) (bool, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, err
	}

	// Set a reasonable User-Agent
	req.Header.Set("User-Agent", "Smite2-Discord-Bot/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		// Check if it's a TLS error and try to provide more helpful feedback
		if strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "tls") {
			return false, fmt.Errorf("TLS/SSL certificate issue (may work in production)")
		}
		return false, err
	}
	defer resp.Body.Close()

	// Consider 200-299 as success
	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Println("Usage: validate_urls <gods_json_file>")
		log.Println("Example: validate_urls data/gods.json")
		os.Exit(1)
	}

	godsFile := os.Args[1]

	log.Printf("Loading gods data from %s...", godsFile)

	// Load gods from the specified file
	if err := smite.LoadGodsFromFile(godsFile); err != nil {
		log.Fatalf("Failed to load gods data: %v", err)
	}

	gods := smite.GetAllGods()
	log.Printf("Validating URLs for %d gods...", len(gods))

	var invalidSmiteURLs []string
	var invalidWikiURLs []string
	validCount := 0
	totalChecked := 0

	fmt.Printf("%-20s %-50s %-50s %-8s %-8s\n", "God", "Smite2.com URL", "Wiki URL", "Smite", "Wiki")
	fmt.Println(strings.Repeat("-", 140))

	for _, god := range gods {
		// Generate URLs using smite package functions
		smiteURL := smite.GetGodSmite2ComURL(god.Name)
		wikiURL := smite.GetGodWikiURL(god.Name)

		// Check Smite2.com URL
		smiteValid, err := checkURL(smiteURL)
		if err != nil {
			log.Printf("Error checking Smite URL for %s: %v", god.Name, err)
			smiteValid = false
		}
		if !smiteValid {
			invalidSmiteURLs = append(invalidSmiteURLs, fmt.Sprintf("%s -> %s", god.Name, smiteURL))
		}

		// Check Wiki URL
		wikiValid, err := checkURL(wikiURL)
		if err != nil {
			log.Printf("Error checking Wiki URL for %s: %v", god.Name, err)
			wikiValid = false
		}
		if !wikiValid {
			invalidWikiURLs = append(invalidWikiURLs, fmt.Sprintf("%s -> %s", god.Name, wikiURL))
		}

		// Status indicators
		smiteStatus := "✅"
		if !smiteValid {
			smiteStatus = "❌"
		}
		wikiStatus := "✅"
		if !wikiValid {
			wikiStatus = "❌"
		}

		fmt.Printf("%-20s %-50s %-50s %-8s %-8s\n",
			god.Name,
			smiteURL,
			wikiURL,
			smiteStatus,
			wikiStatus)

		if smiteValid && wikiValid {
			validCount++
		}
		totalChecked++
	}

	fmt.Println(strings.Repeat("-", 140))
	fmt.Printf("Validation Summary:\n")
	fmt.Printf("Total gods checked: %d\n", totalChecked)
	fmt.Printf("Gods with both URLs valid: %d\n", validCount)
	fmt.Printf("Invalid Smite2.com URLs: %d\n", len(invalidSmiteURLs))
	fmt.Printf("Invalid Wiki URLs: %d\n", len(invalidWikiURLs))

	if len(invalidSmiteURLs) > 0 {
		fmt.Printf("\n❌ Invalid Smite2.com URLs:\n")
		for _, url := range invalidSmiteURLs {
			fmt.Printf("  %s\n", url)
		}
	}

	if len(invalidWikiURLs) > 0 {
		fmt.Printf("\n❌ Invalid Wiki URLs:\n")
		for _, url := range invalidWikiURLs {
			fmt.Printf("  %s\n", url)
		}
	}

	if len(invalidSmiteURLs) == 0 && len(invalidWikiURLs) == 0 {
		fmt.Printf("\n🎉 All URLs are valid!\n")
	}
}
