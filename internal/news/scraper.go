package news

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/types"
)

// Scraper handles scraping news from multiple sources
type Scraper struct {
	sources []NewsSource
	timeout time.Duration
}

// NewsSource defines a news source configuration
type NewsSource struct {
	Name        string
	BaseURL     string
	SearchPath  string // e.g., "/search?q={symbol}"
	Selectors   ArticleSelectors
	RateLimit   time.Duration
}

// ArticleSelectors defines CSS selectors for extracting article data
type ArticleSelectors struct {
	ArticleContainer string
	Title            string
	URL              string
	Content          string
	PublishedAt      string
}

// NewScraper creates a new news scraper with default sources
func NewScraper(timeout time.Duration) *Scraper {
	return &Scraper{
		sources: getDefaultSources(),
		timeout: timeout,
	}
}

// getDefaultSources returns a list of financial news sources to scrape
func getDefaultSources() []NewsSource {
	return []NewsSource{
		{
			Name:       "MoneyControl",
			BaseURL:    "https://www.moneycontrol.com",
			SearchPath: "/news/tags/{symbol}.html",
			Selectors: ArticleSelectors{
				ArticleContainer: "li.clearfix",
				Title:            "h2 a, h3 a",
				URL:              "h2 a, h3 a",
				Content:          "p",
				PublishedAt:      "span.ago",
			},
			RateLimit: 2 * time.Second,
		},
		{
			Name:       "EconomicTimes",
			BaseURL:    "https://economictimes.indiatimes.com",
			SearchPath: "/topic/{symbol}",
			Selectors: ArticleSelectors{
				ArticleContainer: "div.story-box",
				Title:            "a",
				URL:              "a",
				Content:          "p",
				PublishedAt:      "time",
			},
			RateLimit: 2 * time.Second,
		},
		{
			Name:       "BusinessStandard",
			BaseURL:    "https://www.business-standard.com",
			SearchPath: "/search?q={symbol}",
			Selectors: ArticleSelectors{
				ArticleContainer: "div.listing-txt",
				Title:            "a.Hdng",
				URL:              "a.Hdng",
				Content:          "p",
				PublishedAt:      "span.listing-date",
			},
			RateLimit: 2 * time.Second,
		},
	}
}

// ScrapeNews fetches news articles for a given symbol from all sources
func (s *Scraper) ScrapeNews(ctx context.Context, symbol string, maxArticles int) ([]types.NewsArticle, error) {
	logger.Info(ctx, "Starting news scraping", "symbol", symbol, "sources", len(s.sources))

	allArticles := []types.NewsArticle{}
	articlesPerSource := maxArticles / len(s.sources)
	if articlesPerSource < 1 {
		articlesPerSource = 1
	}

	for _, source := range s.sources {
		articles, err := s.scrapeSource(ctx, source, symbol, articlesPerSource)
		if err != nil {
			logger.ErrorWithErr(ctx, "Failed to scrape source", err, "source", source.Name, "symbol", symbol)
			continue
		}
		allArticles = append(allArticles, articles...)

		// Rate limiting between sources
		time.Sleep(source.RateLimit)
	}

	logger.Info(ctx, "News scraping completed", "symbol", symbol, "articles", len(allArticles))
	return allArticles, nil
}

// scrapeSource scrapes articles from a single news source
func (s *Scraper) scrapeSource(ctx context.Context, source NewsSource, symbol string, maxArticles int) ([]types.NewsArticle, error) {
	articles := []types.NewsArticle{}

	// Create collector with timeout
	c := colly.NewCollector(
		colly.AllowedDomains(getDomain(source.BaseURL)),
		colly.MaxDepth(1),
		colly.Async(false),
	)

	c.SetRequestTimeout(s.timeout)

	// Set user agent to avoid being blocked
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	})

	// Extract articles
	c.OnHTML(source.Selectors.ArticleContainer, func(e *colly.HTMLElement) {
		if len(articles) >= maxArticles {
			return
		}

		title := strings.TrimSpace(e.ChildText(source.Selectors.Title))
		if title == "" {
			return
		}

		articleURL := e.ChildAttr(source.Selectors.URL, "href")
		if articleURL == "" {
			return
		}

		// Make URL absolute
		if !strings.HasPrefix(articleURL, "http") {
			articleURL = source.BaseURL + articleURL
		}

		content := strings.TrimSpace(e.ChildText(source.Selectors.Content))
		publishedAt := strings.TrimSpace(e.ChildText(source.Selectors.PublishedAt))

		articles = append(articles, types.NewsArticle{
			Title:       title,
			URL:         articleURL,
			Content:     content,
			Source:      source.Name,
			PublishedAt: publishedAt,
			Symbol:      symbol,
		})
	})

	c.OnError(func(r *colly.Response, err error) {
		logger.ErrorWithErr(ctx, "Scraping error", err, "source", source.Name, "url", r.Request.URL.String())
	})

	// Build search URL
	searchURL := source.BaseURL + strings.ReplaceAll(source.SearchPath, "{symbol}", strings.ToLower(symbol))

	// Visit the search page
	err := c.Visit(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to visit %s: %w", searchURL, err)
	}

	c.Wait()

	// Fetch full content for each article (if needed)
	articles = s.enrichArticles(ctx, articles, source)

	return articles, nil
}

// enrichArticles fetches full content for articles if the initial scrape only got summaries
func (s *Scraper) enrichArticles(ctx context.Context, articles []types.NewsArticle, source NewsSource) []types.NewsArticle {
	enriched := make([]types.NewsArticle, len(articles))
	copy(enriched, articles)

	for i := range enriched {
		// If content is too short, try to fetch full article
		if len(enriched[i].Content) < 100 {
			fullContent := s.fetchArticleContent(ctx, enriched[i].URL)
			if fullContent != "" {
				enriched[i].Content = fullContent
			}
		}

		// Rate limiting between article fetches
		time.Sleep(500 * time.Millisecond)
	}

	return enriched
}

// fetchArticleContent fetches full content from an article URL
func (s *Scraper) fetchArticleContent(ctx context.Context, articleURL string) string {
	c := colly.NewCollector()
	c.SetRequestTimeout(s.timeout)

	var content string

	c.OnHTML("article, div.article-body, div.content-body, div.story-content", func(e *colly.HTMLElement) {
		// Extract all paragraph text
		paragraphs := []string{}
		e.ForEach("p", func(_ int, el *colly.HTMLElement) {
			text := strings.TrimSpace(el.Text)
			if text != "" && len(text) > 20 {
				paragraphs = append(paragraphs, text)
			}
		})
		content = strings.Join(paragraphs, "\n\n")
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	})

	err := c.Visit(articleURL)
	if err != nil {
		logger.ErrorWithErr(ctx, "Failed to fetch article content", err, "url", articleURL)
		return ""
	}

	return content
}

// getDomain extracts domain from URL
func getDomain(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// ScrapeGoogleNews searches Google News for company news (fallback method)
func (s *Scraper) ScrapeGoogleNews(ctx context.Context, companyName string, maxArticles int) ([]types.NewsArticle, error) {
	articles := []types.NewsArticle{}

	c := colly.NewCollector(
		colly.AllowedDomains("news.google.com", "www.google.com"),
	)

	c.SetRequestTimeout(s.timeout)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	})

	c.OnHTML("article", func(e *colly.HTMLElement) {
		if len(articles) >= maxArticles {
			return
		}

		title := e.ChildText("h3, h4")
		link := e.ChildAttr("a", "href")

		if title != "" && link != "" {
			// Clean up Google News redirect URL
			if strings.HasPrefix(link, "./articles/") {
				link = "https://news.google.com" + link[1:]
			}

			articles = append(articles, types.NewsArticle{
				Title:  title,
				URL:    link,
				Source: "GoogleNews",
				Symbol: companyName,
			})
		}
	})

	searchQuery := url.QueryEscape(companyName + " stock news India")
	searchURL := fmt.Sprintf("https://news.google.com/search?q=%s&hl=en-IN&gl=IN&ceid=IN:en", searchQuery)

	err := c.Visit(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape Google News: %w", err)
	}

	c.Wait()

	logger.Info(ctx, "Google News scraping completed", "company", companyName, "articles", len(articles))
	return articles, nil
}
