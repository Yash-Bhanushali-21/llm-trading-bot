# News Sentiment Analysis Feature

## Overview

The LLM Trading Bot now includes a comprehensive **News Sentiment Analysis** feature that scrapes financial news from multiple sources, analyzes sentiment using LLM, and integrates the analysis into trading decisions.

## Features

### 1. Multi-Source News Scraping
- Scrapes news from multiple Indian financial news websites:
  - **MoneyControl** - Leading financial news portal
  - **Economic Times** - Business and finance news
  - **Business Standard** - Market and company news
- Fallback to **Google News** if primary sources fail
- Configurable article limits and timeouts
- Rate-limited scraping to avoid blocking

### 2. Intelligent Sentiment Analysis
- Uses the same LLM (OpenAI or Claude) configured for trading decisions
- Analyzes articles across multiple dimensions:
  - **Overall Sentiment**: POSITIVE, NEGATIVE, NEUTRAL, or MIXED
  - **Business Outlook**: Company's future prospects (-1.0 to 1.0)
  - **Management Quality**: Leadership decisions and competence (-1.0 to 1.0)
  - **Investment Attractiveness**: Impact on investment appeal (-1.0 to 1.0)
- Aggregates sentiment from multiple articles for robust analysis
- Confidence scoring based on article count and sentiment consistency

### 3. Smart Caching
- Caches sentiment results to avoid repeated scraping
- Configurable cache duration (default: 1 hour)
- Automatic cleanup of expired entries
- Cache statistics available

### 4. Seamless Integration
- Sentiment data automatically included in LLM decision context
- LLM considers both technical indicators and news sentiment
- Minimum confidence threshold to filter unreliable sentiment
- Can be enabled/disabled without code changes

## Configuration

Add the following to your `config.yaml`:

```yaml
# ───────────────────────────────
# 📰  NEWS SENTIMENT ANALYSIS
# ───────────────────────────────
news_sentiment:
  enabled: true               # enable/disable news sentiment analysis
  max_articles: 15            # maximum articles to analyze per symbol
  cache_duration_hours: 1     # cache sentiment for this many hours
  scraper_timeout_seconds: 30 # timeout for web scraping operations

  # When to use sentiment in decisions
  use_for_decisions: true     # include sentiment in trading decisions
  min_confidence: 0.4         # minimum confidence to use sentiment (0.0-1.0)
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable/disable the entire sentiment analysis feature |
| `max_articles` | integer | `15` | Maximum number of articles to scrape per symbol |
| `cache_duration_hours` | integer | `1` | How long to cache sentiment results (in hours) |
| `scraper_timeout_seconds` | integer | `30` | Timeout for web scraping operations (in seconds) |
| `use_for_decisions` | boolean | `true` | Whether to include sentiment in trading decisions |
| `min_confidence` | float | `0.4` | Minimum confidence level (0.0-1.0) to use sentiment |

## How It Works

### 1. News Collection
When the trading bot evaluates a symbol:
1. Checks if sentiment is cached (and still valid)
2. If not cached, scrapes news from configured sources
3. Extracts article titles, URLs, content, and publication dates
4. Fetches full article content if initial scrape only got summaries

### 2. Sentiment Analysis
For each article:
1. Sends article content to the configured LLM
2. LLM analyzes sentiment across multiple factors
3. Returns structured sentiment data with scores and reasoning

### 3. Aggregation
Combines multiple article sentiments:
- Calculates average sentiment scores
- Determines overall sentiment (POSITIVE/NEGATIVE/NEUTRAL/MIXED)
- Generates investment recommendation
- Calculates confidence based on data quality

### 4. Decision Integration
When making trading decisions:
1. Sentiment data is added to the LLM context
2. LLM receives both technical indicators AND sentiment
3. LLM weighs sentiment based on:
   - Sentiment score and confidence
   - Business outlook
   - Management quality
   - Article count and consistency

## Sentiment Data Structure

### Individual Article Sentiment
```json
{
  "article_title": "Reliance Industries announces major expansion",
  "url": "https://...",
  "sentiment": "POSITIVE",
  "score": 0.75,
  "reasoning": "Strong business expansion with positive outlook",
  "factors": {
    "business_outlook": 0.8,
    "management": 0.7,
    "investments": 0.75
  }
}
```

### Aggregated Sentiment
```json
{
  "symbol": "RELIANCE",
  "overall_sentiment": "POSITIVE",
  "overall_score": 0.65,
  "article_count": 12,
  "summary": "Analyzed 12 articles. Sentiment breakdown: 8 positive, 2 negative, 2 neutral.",
  "recommendation": "BUY: Generally positive sentiment, consider buying",
  "confidence": 0.85,
  "timestamp": 1699564800
}
```

## LLM Integration

The system prompt has been updated to guide the LLM on using sentiment:

```
When news sentiment data is provided in context, factor it into your decision:
- POSITIVE sentiment (score > 0.3) with good business outlook → favor BUY
- NEGATIVE sentiment (score < -0.3) with poor outlook → favor SELL
- MIXED/NEUTRAL or low confidence sentiment → rely more on technical indicators
- Consider sentiment confidence level when weighting news vs. technical signals
```

## Usage Examples

### Basic Usage
The feature works automatically once enabled in config. No code changes needed:

```bash
# Just run the bot as usual
./bot
```

### Logs Output
```
INFO  News sentiment service initialized max_articles=15 cache_duration_hours=1
INFO  Fetching fresh news sentiment symbol=RELIANCE
INFO  Starting news scraping symbol=RELIANCE sources=3
INFO  News scraping completed symbol=RELIANCE articles=12
INFO  Analyzing sentiment for multiple articles symbol=RELIANCE count=12
INFO  Sentiment analysis completed symbol=RELIANCE overall=POSITIVE score=0.65
INFO  Including news sentiment in decision symbol=RELIANCE sentiment=POSITIVE score=0.65 confidence=0.85
```

### Disable Feature
To disable sentiment analysis:

```yaml
news_sentiment:
  enabled: false
```

Or to disable just the decision integration:

```yaml
news_sentiment:
  enabled: true
  use_for_decisions: false  # Still scrapes/analyzes but doesn't affect decisions
```

## Architecture

### Components

1. **Scraper** (`internal/news/scraper.go`)
   - Web scraping using Colly framework
   - Multi-source support with fallback
   - Rate limiting and timeout handling

2. **Analyzer** (`internal/news/analyzer.go`)
   - LLM-based sentiment analysis
   - Support for OpenAI and Claude
   - Structured output parsing

3. **Service** (`internal/news/service.go`)
   - High-level service interface
   - Caching layer
   - Error handling and fallbacks

4. **Integration** (`internal/engine/engine.go`)
   - Seamless integration into trading engine
   - Context data enrichment
   - Confidence-based filtering

### Data Flow

```
Symbol → Check Cache → [MISS] → Scrape News → Analyze Sentiment → Aggregate → Cache
                    ↓                                                           ↓
                [HIT] ────────────────────────────────────────────────────────→ Return

Sentiment → Add to Context → LLM Decide → Trading Decision
```

## Performance Considerations

### Caching
- Default 1-hour cache prevents excessive scraping
- Reduces API calls to LLM for sentiment analysis
- Configurable cache duration based on needs

### Rate Limiting
- 2-second delay between news sources
- 500ms delay between article fetches
- 1-second delay between article analyses
- Prevents being blocked by news sites

### Timeouts
- 30-second default timeout for scraping
- Prevents hanging on slow/unresponsive sites
- Configurable per deployment

### Concurrent Scraping
- Sources scraped sequentially (not parallel) to avoid blocks
- Articles enriched sequentially for the same reason
- LLM analysis done sequentially to respect rate limits

## Error Handling

The system is designed to be resilient:

1. **Scraping Failures**: Falls back to Google News or returns neutral sentiment
2. **LLM Failures**: Returns neutral sentiment, allows trading to continue
3. **Low Confidence**: Sentiment ignored if below threshold
4. **Cache Misses**: Fresh data fetched automatically
5. **Network Issues**: Errors logged but trading continues

## Testing

Run the test suite:

```bash
# Test news package
go test ./internal/news -v

# Test entire project
go test ./... -v
```

## Environment Variables

No additional environment variables needed. The feature uses the same LLM credentials:
- `OPENAI_API_KEY` (if using OpenAI)
- `ANTHROPIC_API_KEY` (if using Claude)

## Limitations

1. **Language**: Currently optimized for English news sources
2. **Geography**: Focused on Indian financial news websites
3. **Rate Limits**: Subject to news website rate limiting
4. **LLM Costs**: Each article analysis requires an LLM API call
5. **Accuracy**: Sentiment accuracy depends on LLM quality and article content

## Future Enhancements

Potential improvements:
- [ ] Support for more news sources (international sites)
- [ ] RSS feed integration for faster updates
- [ ] Sentiment trend tracking over time
- [ ] Historical sentiment database
- [ ] Machine learning classifier to reduce LLM costs
- [ ] Multi-language support
- [ ] Real-time news alerts for significant sentiment changes
- [ ] Sentiment visualization dashboard

## Troubleshooting

### No articles found
- Check if news websites have changed their HTML structure
- Try Google News fallback
- Verify network connectivity

### LLM errors
- Check API key is set correctly
- Verify LLM provider configuration
- Check API rate limits

### Low confidence
- Increase `max_articles` to get more data
- Check article quality from sources
- Adjust `min_confidence` threshold

### Cache issues
- Clear cache by restarting the bot
- Adjust `cache_duration_hours` if data is stale
- Check system memory if cache grows too large

## Support

For issues or questions:
1. Check logs for error messages
2. Review configuration settings
3. Test with sentiment analysis disabled
4. Open an issue in the repository

## License

Part of the LLM Trading Bot project. See main LICENSE file.
