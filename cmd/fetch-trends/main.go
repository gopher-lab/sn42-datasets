package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gopher-lab/gopher-client/client"
	"github.com/joho/godotenv"
	"github.com/masa-finance/tee-worker/v2/api/args/twitter"
	"github.com/masa-finance/tee-worker/v2/api/types"
)

const (
	dataDir         = "data"
	defaultAmount   = 10000
	minLikesFilter  = " min_faves:100"
	apiMaxResults   = 100 // Maximum results per API request
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: failed to load .env file: %v", err)
	}

	// Initialize gopher-client
	c, err := client.NewClientFromConfig()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if c.Token == "" {
		log.Fatal("GOPHER_CLIENT_TOKEN is not set")
	}

	fmt.Println("Fetching Twitter trends...")

	// Get trends using the client
	trends, err := getTrends(c)
	if err != nil {
		log.Fatalf("Failed to fetch trends: %v", err)
	}

	fmt.Printf("Found %d trending topics:\n", len(trends))
	for i, trend := range trends {
		fmt.Printf("%d. %s\n", i+1, trend)
	}

	// Get target tweet count from env
	targetTweets := defaultAmount
	if amountStr := os.Getenv("AMOUNT"); amountStr != "" {
		amount, err := strconv.Atoi(amountStr)
		if err != nil {
			log.Fatalf("Invalid AMOUNT: %s", amountStr)
		}
		targetTweets = amount
	}

	// Process each trend
	for _, trend := range trends {
		fmt.Printf("\n=== Processing trend: %s ===\n", trend)
		
		// Sanitize trend for filename
		sanitizedTrend := sanitizeTrend(trend)
		if sanitizedTrend == "" {
			fmt.Printf("Skipping trend (empty after sanitization): %s\n", trend)
			continue
		}

		// Create query: trend + min likes filter
		query := fmt.Sprintf(`"%s"%s`, trend, minLikesFilter)
		outputFile := generateOutputFilename(sanitizedTrend, targetTweets)

		fmt.Printf("Query: %s\n", query)
		fmt.Printf("Output file: %s\n", outputFile)
		fmt.Printf("Target tweets: %d\n", targetTweets)

		// Fetch tweets for this trend
		tweets, err := fetchTrendTweets(c, query, targetTweets)
		if err != nil {
			fmt.Printf("Error fetching tweets for trend '%s': %v\n", trend, err)
			continue
		}

		// Save to file
		if err := saveTrendTweets(tweets, trend, query, outputFile); err != nil {
			fmt.Printf("Error saving tweets for trend '%s': %v\n", trend, err)
			continue
		}

		fmt.Printf("✅ Successfully saved %d tweets for trend '%s'\n", len(tweets), trend)
	}

	fmt.Println("\n✅ All trends processed!")
}

// getTrends fetches trending topics using the gopher client.
// It submits a GetTrends job via SearchTwitterWithArgsAsync with Type=CapGetTrends,
// waits for completion, then extracts trend strings from the returned documents.
func getTrends(c *client.Client) ([]string, error) {
	args := twitter.NewSearchArguments()
	args.Type = types.CapGetTrends

	resp, err := c.SearchTwitterWithArgsAsync(args)
	if err != nil {
		return nil, fmt.Errorf("failed to submit get trends job: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("get trends job error: %s", resp.Error)
	}
	if resp.UUID == "" {
		return nil, fmt.Errorf("get trends job returned no job ID")
	}

	fmt.Printf("Get trends job submitted, waiting for completion (job ID: %s)...\n", resp.UUID)
	docs, err := c.WaitForJobCompletion(resp.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for trends job: %w", err)
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("no trends returned")
	}

	trends := make([]string, 0, len(docs))
	for _, d := range docs {
		// tee-indexer getDocsFromTrends uses Id and Content as the trend string
		s := d.Id
		if s == "" {
			s = d.Content
		}
		s = strings.TrimSpace(s)
		if s != "" {
			trends = append(trends, s)
		}
	}
	return trends, nil
}

// fetchTrendTweets fetches tweets for a specific trend query
func fetchTrendTweets(c *client.Client, query string, targetCount int) ([]types.Document, error) {
	var allTweets []types.Document
	currentQuery := query
	maxResults := apiMaxResults
	
	if targetCount < maxResults {
		maxResults = targetCount
	}

	for len(allTweets) < targetCount {
		fmt.Printf("Fetching batch... (current: %d/%d tweets)\n", len(allTweets), targetCount)

		// Create search arguments
		args := twitter.NewSearchArguments()
		args.Query = currentQuery
		args.MaxResults = maxResults
		args.Type = types.CapSearchByQuery

		// Search for tweets
		results, err := c.SearchTwitterWithArgs(args)
		if err != nil {
			fmt.Printf("Error searching tweets: %v\n", err)
			// Don't fail entirely, just return what we have
			break
		}

		if len(results) == 0 {
			fmt.Println("No more results available.")
			break
		}

		allTweets = append(allTweets, results...)
		fmt.Printf("Fetched %d tweets. Total: %d/%d\n", len(results), len(allTweets), targetCount)

		if len(allTweets) >= targetCount {
			break
		}

		// Get last tweet ID for pagination
		lastTweetID, err := getLastTweetID(results)
		if err != nil {
			fmt.Printf("Error getting last tweet ID: %v\n", err)
			break
		}

		// Update query with max_id for pagination
		currentQuery = fmt.Sprintf("%s max_id:%d", query, lastTweetID)
	}

	return allTweets, nil
}

// getLastTweetID extracts the tweet ID from the last document
func getLastTweetID(results []types.Document) (int64, error) {
	if len(results) == 0 {
		return 0, fmt.Errorf("no results")
	}

	lastDoc := results[len(results)-1]
	if metadata := lastDoc.Metadata; metadata != nil {
		if tweetID, ok := metadata["tweet_id"]; ok {
			switch v := tweetID.(type) {
			case int64:
				return v, nil
			case float64:
				return int64(v), nil
			case string:
				id, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return 0, err
				}
				return id, nil
			}
		}
	}

	if lastDoc.Id != "" {
		id, err := strconv.ParseInt(lastDoc.Id, 10, 64)
		if err == nil {
			return id, nil
		}
	}

	return 0, fmt.Errorf("could not extract tweet_id")
}

// sanitizeTrend sanitizes a trend string for use in filenames
func sanitizeTrend(trend string) string {
	// Convert to lowercase
	sanitized := strings.ToLower(trend)
	
	// Replace spaces with underscores
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	
	// Remove special characters (keep alphanumeric and underscore)
	reg := regexp.MustCompile(`[^a-z0-9_]`)
	sanitized = reg.ReplaceAllString(sanitized, "")
	
	// Remove multiple consecutive underscores
	reg = regexp.MustCompile(`_+`)
	sanitized = reg.ReplaceAllString(sanitized, "_")
	
	// Trim leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")
	
	return sanitized
}

// generateOutputFilename creates a filename for trend tweets
func generateOutputFilename(trend string, targetCount int) string {
	// Ensure data directory exists
	os.MkdirAll(dataDir, 0755)
	
	filename := fmt.Sprintf("trend_%s_%d.json", trend, targetCount)
	return filepath.Join(dataDir, filename)
}

// saveTrendTweets saves tweets to a JSON file
func saveTrendTweets(tweets []types.Document, trend, query, filename string) error {
	output := struct {
		TotalTweets int              `json:"total_tweets"`
		Trend       string           `json:"trend"`
		Query       string           `json:"query"`
		CollectedAt string           `json:"collected_at"`
		Tweets      []types.Document `json:"tweets"`
	}{
		TotalTweets: len(tweets),
		Trend:       trend,
		Query:       query,
		CollectedAt: time.Now().UTC().Format(time.RFC3339),
		Tweets:      tweets,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
