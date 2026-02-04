package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gopher-lab/gopher-client/client"
	"github.com/masa-finance/tee-worker/v2/api/args/twitter"
	"github.com/masa-finance/tee-worker/v2/api/types"
)

const (
	baseQuery    = "bitcoin min_faves:1000"
	maxResults   = 100
	targetTweets = 10000
	outputFile   = "data/tweets_10000.json"
)

func main() {
	// Initialize gopher-client from .env file
	c, err := client.NewClientFromConfig()
	if err != nil {
		log.Fatalf("Failed to create client from config: %v\nMake sure GOPHER_CLIENT_TOKEN is set in your .env file", err)
	}

	// Verify token is set
	if c.Token == "" {
		log.Fatal("GOPHER_CLIENT_TOKEN is not set. Please set it in your .env file")
	}

	fmt.Println("Starting tweet collection...")
	fmt.Printf("Target: %d tweets\n", targetTweets)
	fmt.Printf("Batch size: %d tweets per request\n\n", maxResults)

	// Initialize tweets array
	var allTweets []types.Document
	query := baseQuery

	// Loop until we have 10,000 tweets or no more results
	for len(allTweets) < targetTweets {
		fmt.Printf("Fetching batch... (current: %d/%d tweets)\n", len(allTweets), targetTweets)

		// Create search arguments
		args := twitter.NewSearchArguments()
		args.Query = query
		args.MaxResults = maxResults
		args.Type = types.CapSearchByQuery // Explicitly set search type

		// Make API request (synchronous - waits for completion)
		results, err := c.SearchTwitterWithArgs(args)
		if err != nil {
			log.Printf("Error fetching tweets: %v", err)
			break
		}

		// Check if we got any results
		if len(results) == 0 {
			fmt.Println("No more results available.")
			break
		}

		// Append results to our collection
		allTweets = append(allTweets, results...)
		fmt.Printf("Fetched %d tweets in this batch. Total: %d/%d\n\n", len(results), len(allTweets), targetTweets)

		// If we've reached our target, break
		if len(allTweets) >= targetTweets {
			break
		}

		// Get the last tweet ID for pagination
		lastTweetID, err := getLastTweetID(results)
		if err != nil {
			log.Printf("Error extracting last tweet ID: %v", err)
			break
		}

		// Update query with max_id for next iteration
		query = fmt.Sprintf("%s max_id:%d", baseQuery, lastTweetID)
	}

	// Save to JSON file
	fmt.Printf("\nSaving %d tweets to %s...\n", len(allTweets), outputFile)
	if err := saveTweetsToFile(allTweets); err != nil {
		log.Fatalf("Failed to save tweets: %v", err)
	}

	fmt.Printf("✅ Successfully collected and saved %d tweets to %s\n", len(allTweets), outputFile)
}

// getLastTweetID extracts the tweet ID from the last document in the results
func getLastTweetID(results []types.Document) (int64, error) {
	if len(results) == 0 {
		return 0, fmt.Errorf("no results to extract tweet ID from")
	}

	// Get the last tweet (oldest in the batch)
	lastDoc := results[len(results)-1]

	// Try to get tweet_id from metadata
	if metadata := lastDoc.Metadata; metadata != nil {
		if tweetID, ok := metadata["tweet_id"]; ok {
			switch v := tweetID.(type) {
			case int64:
				return v, nil
			case float64:
				// JSON numbers are unmarshaled as float64
				return int64(v), nil
			case string:
				id, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse tweet_id string: %w", err)
				}
				return id, nil
			}
		}
	}

	// Fallback: try to parse the Id field
	if lastDoc.Id != "" {
		id, err := strconv.ParseInt(lastDoc.Id, 10, 64)
		if err == nil {
			return id, nil
		}
	}

	return 0, fmt.Errorf("could not extract tweet_id from document")
}

// saveTweetsToFile saves the tweets to a JSON file with proper formatting
func saveTweetsToFile(tweets []types.Document) error {
	// Create output structure with metadata
	output := struct {
		TotalTweets int              `json:"total_tweets"`
		Query       string           `json:"query"`
		CollectedAt string           `json:"collected_at"`
		Tweets      []types.Document `json:"tweets"`
	}{
		TotalTweets: len(tweets),
		Query:       baseQuery,
		CollectedAt: time.Now().UTC().Format(time.RFC3339),
		Tweets:      tweets,
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tweets: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
