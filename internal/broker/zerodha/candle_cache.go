package zerodha

import (
	"fmt"
	"sync"

	"llm-trading-bot/internal/types"
)

// candleCache manages symbol-specific candle buffers with thread-safe access
type candleCache struct {
	buffers map[string]*candleBuffer
	mu      sync.RWMutex
}

// candleBuffer stores recent candles in a circular buffer
type candleBuffer struct {
	candles []types.Candle
	maxSize int
}

// newCandleCache creates a new candle cache
func newCandleCache() *candleCache {
	return &candleCache{
		buffers: make(map[string]*candleBuffer),
	}
}

// initBuffer initializes a buffer for a symbol
func (cc *candleCache) initBuffer(symbol string, maxSize int) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.buffers[symbol] = &candleBuffer{
		candles: make([]types.Candle, 0, maxSize),
		maxSize: maxSize,
	}
}

// addCandle adds a candle to the symbol's buffer
func (cc *candleCache) addCandle(symbol string, candle types.Candle) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	buffer, exists := cc.buffers[symbol]
	if !exists {
		return
	}

	// Add candle to buffer
	buffer.candles = append(buffer.candles, candle)

	// Maintain circular buffer size
	if len(buffer.candles) > buffer.maxSize {
		buffer.candles = buffer.candles[1:]
	}
}

// getRecent retrieves the last n candles for a symbol
func (cc *candleCache) getRecent(symbol string, n int) ([]types.Candle, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	buffer, exists := cc.buffers[symbol]
	if !exists {
		return nil, fmt.Errorf("no candle data for symbol %s", symbol)
	}

	candles := buffer.candles
	if len(candles) == 0 {
		return nil, fmt.Errorf("no candles available for %s", symbol)
	}

	// Return last n candles
	if len(candles) < n {
		return candles, nil
	}

	return candles[len(candles)-n:], nil
}

// clear removes all candles from all buffers
func (cc *candleCache) clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	for symbol := range cc.buffers {
		cc.buffers[symbol].candles = make([]types.Candle, 0, cc.buffers[symbol].maxSize)
	}
}
