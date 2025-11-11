package forensic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/types"
)

// DocumentAnalyzer handles downloading and analyzing company documents
type DocumentAnalyzer struct {
	downloadDir string
	httpClient  *http.Client
	llmAnalyzer *LLMDocumentAnalyzer
}

// DocumentAnalysis represents analysis results from a document
type DocumentAnalysis struct {
	DocumentURL  string                  `json:"document_url"`
	DocumentType string                  `json:"document_type"` // Annual Report, Board Notice, etc.
	AnalyzedAt   time.Time               `json:"analyzed_at"`
	RedFlags     []types.RedFlag         `json:"red_flags"`
	KeyFindings  []string                `json:"key_findings"`
	RiskScore    float64                 `json:"risk_score"`
	Extractions  map[string]interface{}  `json:"extractions"` // Structured data extracted
}

// NewDocumentAnalyzer creates a new document analyzer
func NewDocumentAnalyzer(downloadDir string, llmProvider string) *DocumentAnalyzer {
	if downloadDir == "" {
		downloadDir = "cache/documents"
	}

	// Create download directory
	os.MkdirAll(downloadDir, 0755)

	return &DocumentAnalyzer{
		downloadDir: downloadDir,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		llmAnalyzer: NewLLMDocumentAnalyzer(llmProvider),
	}
}

// AnalyzeDocument downloads and analyzes a company document
func (da *DocumentAnalyzer) AnalyzeDocument(ctx context.Context, documentURL, documentType, symbol string) (*DocumentAnalysis, error) {
	logger.Info(ctx, "Analyzing document", "url", documentURL, "type", documentType)

	// Download document
	localPath, err := da.downloadDocument(ctx, documentURL, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to download document: %w", err)
	}

	// Extract text from document
	text, err := da.extractText(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text: %w", err)
	}

	// Analyze with LLM
	analysis := &DocumentAnalysis{
		DocumentURL:  documentURL,
		DocumentType: documentType,
		AnalyzedAt:   time.Now(),
		RedFlags:     []types.RedFlag{},
		KeyFindings:  []string{},
		Extractions:  make(map[string]interface{}),
	}

	// Perform different analyses based on document type
	switch documentType {
	case "Annual Report":
		da.analyzeAnnualReport(ctx, text, analysis)
	case "Board Notice", "Board Meeting":
		da.analyzeBoardNotice(ctx, text, analysis)
	case "Financial Results":
		da.analyzeFinancialResults(ctx, text, analysis)
	case "Announcement":
		da.analyzeAnnouncement(ctx, text, analysis)
	default:
		da.analyzeGeneral(ctx, text, analysis)
	}

	logger.Info(ctx, "Document analysis complete", "red_flags", len(analysis.RedFlags))
	return analysis, nil
}

func (da *DocumentAnalyzer) downloadDocument(ctx context.Context, url, symbol string) (string, error) {
	// Generate local filename
	filename := fmt.Sprintf("%s_%d_%s", symbol, time.Now().Unix(), filepath.Base(url))
	localPath := filepath.Join(da.downloadDir, filename)

	// Check if already downloaded
	if _, err := os.Stat(localPath); err == nil {
		logger.Info(ctx, "Document already downloaded", "path", localPath)
		return localPath, nil
	}

	// Download
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := da.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Save to file
	out, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	logger.Info(ctx, "Document downloaded", "path", localPath)
	return localPath, nil
}

func (da *DocumentAnalyzer) extractText(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".pdf":
		return da.extractTextFromPDF(filePath)
	case ".html", ".htm":
		return da.extractTextFromHTML(filePath)
	case ".txt":
		data, err := os.ReadFile(filePath)
		return string(data), err
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

func (da *DocumentAnalyzer) extractTextFromPDF(filePath string) (string, error) {
	// For production, use a PDF parsing library like:
	// - github.com/ledongthuc/pdf
	// - github.com/unidoc/unipdf
	// For now, return placeholder
	return fmt.Sprintf("[PDF content from %s - requires PDF parser library]", filePath), nil
}

func (da *DocumentAnalyzer) extractTextFromHTML(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Basic HTML cleaning (strip tags)
	text := string(data)
	// Remove script and style tags
	text = strings.ReplaceAll(text, "<script", "<REMOVESCRIPT")
	text = strings.ReplaceAll(text, "</script>", "</REMOVESCRIPT>")
	text = strings.ReplaceAll(text, "<style", "<REMOVESTYLE")
	text = strings.ReplaceAll(text, "</style>", "</REMOVESTYLE>")

	// For production, use proper HTML parser like:
	// - golang.org/x/net/html
	// - github.com/PuerkitoBio/goquery

	return text, nil
}

func (da *DocumentAnalyzer) analyzeAnnualReport(ctx context.Context, text string, analysis *DocumentAnalysis) {
	logger.Info(ctx, "Analyzing annual report")

	// Check for key forensic indicators in annual reports
	indicators := []string{
		"going concern",
		"material uncertainty",
		"qualified opinion",
		"adverse opinion",
		"related party transaction",
		"contingent liability",
		"legal proceedings",
		"regulatory action",
		"restatement",
		"change in accounting policy",
		"resignation",
		"auditor change",
	}

	for _, indicator := range indicators {
		if containsIgnoreCase(text, indicator) {
			analysis.RedFlags = append(analysis.RedFlags, types.RedFlag{
				Category:    "DOCUMENT_ANALYSIS",
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("Annual report mentions: %s", indicator),
				DetectedAt:  time.Now(),
				Impact:      30.0,
			})
			analysis.KeyFindings = append(analysis.KeyFindings,
				fmt.Sprintf("Found mention of '%s' in annual report", indicator))
		}
	}

	// Use LLM for deeper analysis
	if da.llmAnalyzer != nil {
		llmAnalysis, err := da.llmAnalyzer.AnalyzeAnnualReport(ctx, text)
		if err == nil {
			analysis.RedFlags = append(analysis.RedFlags, llmAnalysis.RedFlags...)
			analysis.KeyFindings = append(analysis.KeyFindings, llmAnalysis.KeyFindings...)
			analysis.Extractions = llmAnalysis.Extractions
		}
	}

	// Calculate risk score
	analysis.RiskScore = float64(len(analysis.RedFlags)) * 10.0
	if analysis.RiskScore > 100 {
		analysis.RiskScore = 100
	}
}

func (da *DocumentAnalyzer) analyzeBoardNotice(ctx context.Context, text string, analysis *DocumentAnalysis) {
	logger.Info(ctx, "Analyzing board notice")

	// Board notices often contain critical governance changes
	keywords := []struct {
		phrase   string
		severity string
		impact   float64
	}{
		{"resignation", "HIGH", 60.0},
		{"removal", "HIGH", 65.0},
		{"appointment", "LOW", 20.0},
		{"related party", "MEDIUM", 45.0},
		{"material transaction", "MEDIUM", 40.0},
		{"loan", "MEDIUM", 35.0},
		{"guarantee", "MEDIUM", 40.0},
		{"auditor", "HIGH", 55.0},
	}

	for _, kw := range keywords {
		if containsIgnoreCase(text, kw.phrase) {
			analysis.RedFlags = append(analysis.RedFlags, types.RedFlag{
				Category:    "BOARD_NOTICE",
				Severity:    kw.severity,
				Description: fmt.Sprintf("Board notice mentions: %s", kw.phrase),
				DetectedAt:  time.Now(),
				Impact:      kw.impact,
			})
		}
	}

	// Calculate risk
	totalImpact := 0.0
	for _, flag := range analysis.RedFlags {
		totalImpact += flag.Impact
	}
	analysis.RiskScore = totalImpact / float64(len(analysis.RedFlags)+1)
}

func (da *DocumentAnalyzer) analyzeFinancialResults(ctx context.Context, text string, analysis *DocumentAnalysis) {
	logger.Info(ctx, "Analyzing financial results")

	// Look for restatements and accounting changes
	indicators := []string{
		"restatement",
		"restated",
		"revision",
		"correction",
		"error",
		"change in accounting",
		"exceptional items",
		"one-time charge",
	}

	for _, indicator := range indicators {
		if containsIgnoreCase(text, indicator) {
			analysis.RedFlags = append(analysis.RedFlags, types.RedFlag{
				Category:    "FINANCIAL_RESULTS",
				Severity:    "HIGH",
				Description: fmt.Sprintf("Financial results mention: %s", indicator),
				DetectedAt:  time.Now(),
				Impact:      50.0,
			})
		}
	}
}

func (da *DocumentAnalyzer) analyzeAnnouncement(ctx context.Context, text string, analysis *DocumentAnalysis) {
	logger.Info(ctx, "Analyzing announcement")

	// General announcement analysis
	redFlagKeywords := []string{
		"penalty",
		"violation",
		"non-compliance",
		"legal notice",
		"investigation",
		"suspension",
		"default",
	}

	for _, keyword := range redFlagKeywords {
		if containsIgnoreCase(text, keyword) {
			analysis.RedFlags = append(analysis.RedFlags, types.RedFlag{
				Category:    "ANNOUNCEMENT",
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("Announcement mentions: %s", keyword),
				DetectedAt:  time.Now(),
				Impact:      40.0,
			})
		}
	}
}

func (da *DocumentAnalyzer) analyzeGeneral(ctx context.Context, text string, analysis *DocumentAnalysis) {
	logger.Info(ctx, "Performing general document analysis")

	// Generic red flag detection
	generalFlags := []string{
		"fraud",
		"misrepresentation",
		"default",
		"bankruptcy",
		"insolvency",
		"winding up",
		"liquidation",
	}

	for _, flag := range generalFlags {
		if containsIgnoreCase(text, flag) {
			analysis.RedFlags = append(analysis.RedFlags, types.RedFlag{
				Category:    "DOCUMENT",
				Severity:    "CRITICAL",
				Description: fmt.Sprintf("Document mentions: %s", flag),
				DetectedAt:  time.Now(),
				Impact:      80.0,
			})
		}
	}
}

func containsIgnoreCase(text, substr string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substr))
}

// CleanupOldDocuments removes documents older than specified days
func (da *DocumentAnalyzer) CleanupOldDocuments(days int) error {
	entries, err := os.ReadDir(da.downloadDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -days)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(da.downloadDir, entry.Name()))
		}
	}

	return nil
}
