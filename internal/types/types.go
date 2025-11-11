package types

type Candle struct {
	Ts                          int64
	Open, High, Low, Close, Vol float64
}
type Indicators struct {
	SMA map[int]float64
	RSI float64
	BB  struct{ Middle, Upper, Lower float64 }
	ATR float64
}
type Decision struct {
	Action, Reason string  `json:"action"`
	Confidence     float64 `json:"confidence"`
	Qty            int     `json:"qty,omitempty"`
}

type StepResult struct {
	Symbol   string      `json:"symbol"`
	Decision Decision    `json:"decision"`
	Price    float64     `json:"price"`
	Time     int64       `json:"time"`
	Orders   []OrderResp `json:"orders"`
	Reason   string      `json:"reason"`
}
type OrderReq struct {
	Symbol, Side string
	Qty          int
	Tag          string
}
type OrderResp struct {
	OrderID, Status, Message string `json:"order_id"`
}

// NewsArticle represents a scraped news article
type NewsArticle struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Content     string    `json:"content"`
	Source      string    `json:"source"`
	PublishedAt string    `json:"published_at"`
	Symbol      string    `json:"symbol"`
}

// ArticleSentiment represents sentiment analysis of a single article
type ArticleSentiment struct {
	ArticleTitle string  `json:"article_title"`
	URL          string  `json:"url"`
	Sentiment    string  `json:"sentiment"` // POSITIVE, NEGATIVE, NEUTRAL
	Score        float64 `json:"score"`     // -1.0 to 1.0
	Reasoning    string  `json:"reasoning"`
	Factors      struct {
		BusinessOutlook float64 `json:"business_outlook"` // -1.0 to 1.0
		Management      float64 `json:"management"`        // -1.0 to 1.0
		Investments     float64 `json:"investments"`       // -1.0 to 1.0
	} `json:"factors"`
}

// NewsSentiment represents aggregated sentiment from multiple articles
type NewsSentiment struct {
	Symbol           string             `json:"symbol"`
	OverallSentiment string             `json:"overall_sentiment"` // POSITIVE, NEGATIVE, NEUTRAL, MIXED
	OverallScore     float64            `json:"overall_score"`     // -1.0 to 1.0
	ArticleCount     int                `json:"article_count"`
	Articles         []ArticleSentiment `json:"articles"`
	Summary          string             `json:"summary"`
	Recommendation   string             `json:"recommendation"`
	Confidence       float64            `json:"confidence"` // 0.0 to 1.0
	Timestamp        int64              `json:"timestamp"`
}
