package forensic

import (
	"fmt"
	"time"

	"llm-trading-bot/internal/forensic/datasource"
	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/store"
)

// CreateDataSource creates the appropriate data source based on configuration
func CreateDataSource(cfg *store.Config) (interfaces.CorporateDataSource, error) {
	if cfg == nil || !cfg.Forensic.Enabled {
		return nil, fmt.Errorf("forensic analysis is not enabled")
	}

	dataSourceType := cfg.Forensic.DataSource
	if dataSourceType == "" {
		dataSourceType = "MOCK" // Default to mock
	}

	switch dataSourceType {
	case "MOCK":
		return NewMockDataSource(), nil

	case "LIVE":
		cacheTTL := time.Duration(cfg.Forensic.CacheTTLHours) * time.Hour
		if cacheTTL == 0 {
			cacheTTL = 24 * time.Hour // Default 24 hours
		}

		liveConfig := &datasource.LiveDataSourceConfig{
			EnableNSE:      cfg.Forensic.EnableNSE,
			EnableBSE:      cfg.Forensic.EnableBSE,
			EnableSEBI:     cfg.Forensic.EnableSEBI,
			EnableScreener: cfg.Forensic.EnableScreener,
			CacheDir:       cfg.Forensic.CacheDir,
			CacheTTL:       cacheTTL,
		}

		// Set defaults if not configured
		if liveConfig.CacheDir == "" {
			liveConfig.CacheDir = "cache/forensic"
		}

		return datasource.NewLiveDataSource(liveConfig), nil

	default:
		return nil, fmt.Errorf("unknown data source type: %s (valid options: MOCK, LIVE)", dataSourceType)
	}
}
