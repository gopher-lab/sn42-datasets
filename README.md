# Twitter Bulk Fetcher

A Go script that fetches large batches of tweets from the Gopher AI subnet API using pagination. This tool collects tweets in batches of 100 and saves them to a JSON file for analysis.

## Features

- Fetches up to 10,000 tweets using pagination
- Uses `max_id` for proper chronological pagination
- Progress logging during collection
- Saves tweets to JSON with metadata
- Configurable via environment variables

## Prerequisites

- Go 1.24.6 or later
- A Gopher AI API token

## Setup

1. **Clone or navigate to the repository:**
   ```bash
   cd sn42
   ```

2. **Create a `.env` file in the root directory:**
   ```bash
   GOPHER_CLIENT_TOKEN=your_api_token_here
   ```

   The script will automatically load this file. You can also set the token as an environment variable:
   ```bash
   export GOPHER_CLIENT_TOKEN=your_api_token_here
   ```

3. **Install dependencies:**
   ```bash
   go mod tidy
   ```

## Usage

### Basic Usage

Run the script to fetch 10,000 tweets matching the default query:

```bash
go run ./cmd/fetch-tweets
```

The default query is: `bitcoin min_faves:1000`

### Customization

You can modify the script constants in `cmd/fetch-tweets/main.go`:

- `baseQuery`: The search query (default: `"bitcoin min_faves:1000"`)
- `maxResults`: Number of tweets per API request (default: `100`, max: `1000`)
- `targetTweets`: Total number of tweets to collect (default: `10000`)
- `outputFile`: Output JSON file path (default: `"data/tweets_10000.json"`)

### Example: Custom Query

To fetch tweets with a different query, modify the `baseQuery` constant:

```go
const (
    baseQuery = "ethereum min_faves:500"  // Your custom query
    // ... other constants
)
```

## Output

The script generates a JSON file in the `data/` directory with the following structure:

```json
{
  "total_tweets": 10000,
  "query": "bitcoin min_faves:1000",
  "collected_at": "2026-02-04T01:22:46Z",
  "tweets": [
    {
      "id": "2018797961606557803",
      "source": "twitter",
      "content": "Tweet content...",
      "metadata": {
        "tweet_id": 2018797961606557700,
        "created_at": "2026-02-03T21:25:16Z",
        "likes": 1715,
        "public_metrics": { ... },
        ...
      }
    },
    ...
  ]
}
```

## How It Works

1. **Initial Request**: Fetches the first 100 tweets matching the query
2. **Pagination**: Uses the last tweet's ID as `max_id` for the next request
3. **Collection**: Continues fetching batches until reaching the target count or running out of results
4. **Output**: Saves all collected tweets to a JSON file with metadata

### Pagination Logic

The script uses Twitter's `max_id` parameter for pagination:
- First request: `bitcoin min_faves:1000`
- Subsequent requests: `bitcoin min_faves:1000 max_id:{last_tweet_id}`

This ensures you get older tweets in chronological order.

## Environment Variables

The script uses the following environment variables (loaded from `.env` file):

- `GOPHER_CLIENT_TOKEN`: Your Gopher AI API token (required)
- `GOPHER_CLIENT_URL`: API base URL (optional, defaults to `https://data.gopher-ai.com/api`)
- `GOPHER_CLIENT_TIMEOUT`: Request timeout (optional, defaults to `60s`)

## Error Handling

The script handles:
- Missing or invalid API tokens
- API errors and rate limiting
- Empty result sets
- File write errors

If an error occurs, the script will log it and exit gracefully.

## Performance

- **Batch Size**: 100 tweets per request (configurable)
- **Rate Limiting**: 1 second delay between requests
- **Total Time**: Approximately 100-200 seconds for 10,000 tweets (depending on API response time)

## Troubleshooting

### "Failed to create client from config"

- Ensure your `.env` file exists in the `sn42/` directory
- Check that `GOPHER_CLIENT_TOKEN` is set correctly
- Verify there are no extra spaces or quotes around the token value

### "No more results available"

- The query may not have enough matching tweets
- Try adjusting the query parameters (e.g., lower `min_faves` threshold)
- Check if you've reached the end of available results

### Rate Limiting

If you encounter rate limiting:
- Increase the delay between requests in the code (currently 1 second)
- Reduce the `maxResults` batch size
- Contact Gopher AI support for API rate limit information

## Building

To build a standalone binary:

```bash
go build -o fetch-tweets ./cmd/fetch-tweets
```

Then run:
```bash
./fetch-tweets
```

## License

See the repository license file for details.
