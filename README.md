# Twitter Bulk Fetcher

A Go script that fetches large batches of tweets from the Gopher AI subnet API using pagination. This tool automatically optimizes batch sizes based on your target amount and saves tweets to a JSON file for analysis.

## Features

- Fetches configurable amounts of tweets using pagination (default: 10,000)
- Automatically optimizes batch size: uses `AMOUNT` if ≤100, otherwise uses API maximum (100)
- Uses `max_id` for proper chronological pagination
- Progress logging during collection
- Saves tweets to JSON with metadata
- Fully configurable via environment variables (QUERY, AMOUNT)

## Prerequisites

- Go 1.24.6 or later
- A Gopher AI API token

## Setup

1. **Clone or navigate to the repository:**
   ```bash
   cd sn42-datasets
   ```

2. **Create a `.env` file in the root directory:**
   ```bash
   GOPHER_CLIENT_TOKEN=your_api_token_here
   QUERY="bitcoin min_faves:1000"
   AMOUNT=10000
   ```

   The script will automatically load this file. You can also set these as environment variables:
   ```bash
   export GOPHER_CLIENT_TOKEN=your_api_token_here
   export QUERY="bitcoin min_faves:1000"
   export AMOUNT=10000
   ```

3. **Install dependencies:**
   ```bash
   go mod tidy
   ```

## Usage

### Basic Usage

Run the script to fetch 10,000 tweets matching your query:

```bash
go run ./cmd/fetch-tweets
```

The query and amount are read from the `QUERY` and `AMOUNT` environment variables in your `.env` file. If not set, they default to: `bitcoin min_faves:1000` and `10000` tweets respectively.

### Customization

#### Custom Query and Amount

Set the `QUERY` and `AMOUNT` environment variables in your `.env` file:

```bash
QUERY="ethereum min_faves:500"
AMOUNT=5000
```

Or set them as environment variables:
```bash
export QUERY="ethereum min_faves:500"
export AMOUNT=5000
go run ./cmd/fetch-tweets
```

**Note**: 
- `AMOUNT` is the total number of tweets to collect
- The batch size (tweets per API request) is automatically set to `min(AMOUNT, 100)`
  - If `AMOUNT` is 50, it will fetch 50 tweets in one request
  - If `AMOUNT` is 5000, it will fetch 100 tweets per request (API maximum) until reaching 5000
- The output filename is automatically generated from your query and amount

## Output

The script automatically generates a filename based on your query and target tweet count. The format is:

```
data/{sanitized_query}_{target_count}.json
```

For example:
- Query: `"bitcoin min_faves:1000"` with 10,000 tweets → `data/bitcoin_min_faves_1000_10000.json`
- Query: `"ethereum min_retweets:50"` with 5,000 tweets → `data/ethereum_min_retweets_50_5000.json`

The output JSON file has the following structure:

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

1. **Initial Request**: Fetches the first batch of tweets matching the query (batch size = `min(AMOUNT, 100)`)
2. **Pagination**: Uses the last tweet's ID as `max_id` for the next request
3. **Collection**: Continues fetching batches until reaching the target count (AMOUNT) or running out of results
4. **Output**: Saves all collected tweets to a JSON file with metadata

**Batch Size Examples:**
- `AMOUNT=50` → 1 request of 50 tweets
- `AMOUNT=500` → 5 requests of 100 tweets each
- `AMOUNT=10000` → 100 requests of 100 tweets each

### Pagination Logic

The script uses Twitter's `max_id` parameter for pagination:
- First request: Uses your `QUERY` as-is (e.g., `bitcoin min_faves:1000`)
- Subsequent requests: Appends `max_id:{last_tweet_id}` to your query (e.g., `bitcoin min_faves:1000 max_id:1234567890`)

This ensures you get older tweets in chronological order.

## Environment Variables

The script uses the following environment variables (loaded from `.env` file):

- `GOPHER_CLIENT_TOKEN`: Your Gopher AI API token (required)
- `QUERY`: Twitter search query (optional, defaults to `"bitcoin min_faves:1000"`)
- `AMOUNT`: Total number of tweets to collect (optional, defaults to `10000`)
- `GOPHER_CLIENT_URL`: API base URL (optional, defaults to `https://data.gopher-ai.com/api`)
- `GOPHER_CLIENT_TIMEOUT`: Request timeout (optional, defaults to `60s`)

**Batch Size Logic**: The script automatically sets the batch size (tweets per API request) to `min(AMOUNT, 100)`. This means:
- If `AMOUNT=50`, it fetches 50 tweets in one request
- If `AMOUNT=5000`, it fetches 100 tweets per request (API max) until reaching 5000

### Query Examples

- `QUERY="bitcoin min_faves:1000"` - Bitcoin tweets with at least 1000 likes
- `QUERY="ethereum min_retweets:50"` - Ethereum tweets with at least 50 retweets
- `QUERY="crypto -filter:retweets"` - Crypto tweets excluding retweets
- `QUERY="from:elonmusk"` - All tweets from a specific user

## Error Handling

The script handles:
- Missing or invalid API tokens
- API errors and rate limiting
- Empty result sets
- File write errors

If an error occurs, the script will log it and exit gracefully.

## Performance

- **Batch Size**: Automatically optimized based on `AMOUNT`:
  - If `AMOUNT ≤ 100`: Single request with that amount
  - If `AMOUNT > 100`: Multiple requests of 100 tweets each (API maximum)
- **Rate Limiting**: 1 second delay between requests
- **Total Time**: Varies based on `AMOUNT`:
  - `AMOUNT=50`: ~5-10 seconds (1 request)
  - `AMOUNT=1000`: ~10-20 seconds (10 requests)
  - `AMOUNT=10000`: ~100-200 seconds (100 requests)

## Troubleshooting

### "Failed to create client from config"

- Ensure your `.env` file exists in the `sn42-datasets/` directory
- Check that `GOPHER_CLIENT_TOKEN` is set correctly
- Verify there are no extra spaces or quotes around the token value

### "QUERY not set in .env, using default"

- This is informational only - the script will use the default query
- To use a custom query, add `QUERY="your query here"` to your `.env` file

### "AMOUNT not set in .env, using default"

- This is informational only - the script will use the default amount (10,000)
- To use a custom amount, add `AMOUNT=5000` (or your desired number) to your `.env` file

### "Invalid AMOUNT value in .env"

- `AMOUNT` must be a positive integer
- Check that there are no quotes, spaces, or non-numeric characters
- Example: `AMOUNT=10000` (correct) vs `AMOUNT="10000"` or `AMOUNT=ten thousand` (incorrect)

### "No more results available"

- The query may not have enough matching tweets
- Try adjusting the query parameters (e.g., lower `min_faves` threshold)
- Check if you've reached the end of available results

### Rate Limiting

If you encounter rate limiting:
- Increase the delay between requests in the code (currently 1 second)
- Reduce the `AMOUNT` value to fetch fewer tweets
- Contact Gopher AI support for API rate limit information

**Note**: The batch size is automatically optimized, so you don't need to manually adjust it. Simply set `AMOUNT` to your desired total number of tweets.

## fetch-trends: Get trends and collect tweets per trend

A second tool, `fetch-trends`, uses the gopher client to **get current Twitter trends**, then for **each trend** collects up to 10,000 tweets with **at least 100 likes** and saves them under `data/`.

### How to run fetch-trends

Same `.env` as fetch-tweets (at least `GOPHER_CLIENT_TOKEN`). Optional: `AMOUNT` (default 10000 tweets per trend).

```bash
go run ./cmd/fetch-trends
```

Or build and run:

```bash
go build -o fetch-trends ./cmd/fetch-trends
./fetch-trends
```

### What it does

1. **Get trends** – Calls the gopher API with a “get trends” job (`CapGetTrends`), waits for completion, and reads the list of trending topic strings.
2. **For each trend** – Builds a query `"{trend}" min_faves:100` and fetches tweets the same way as fetch-tweets (pagination, batch size 100).
3. **Output** – One JSON file per trend in `data/`, e.g. `data/trend_bitcoin_10000.json`, with the same structure as fetch-tweets (metadata, `collected_at`, etc.).

So you get “trends → 10k tweets (min 100 likes) per trend” in one run.

## Building

To build standalone binaries:

```bash
# Tweet fetcher (single query)
go build -o fetch-tweets ./cmd/fetch-tweets

# Trend-based fetcher (trends + 10k tweets per trend with min 100 likes)
go build -o fetch-trends ./cmd/fetch-trends
```

Then run:
```bash
./fetch-tweets
# or
./fetch-trends
```

## License

See the repository license file for details.
