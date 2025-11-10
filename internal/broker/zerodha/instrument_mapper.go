package zerodha

import (
	"sync"
)

// instrumentMapper manages bidirectional mapping between symbols and tokens
type instrumentMapper struct {
	symbolToToken map[string]uint32
	tokenToSymbol map[uint32]string
	mu            sync.RWMutex
}

// newInstrumentMapper creates a new instrument mapper
func newInstrumentMapper() *instrumentMapper {
	return &instrumentMapper{
		symbolToToken: make(map[string]uint32),
		tokenToSymbol: make(map[uint32]string),
	}
}

// addMapping adds a symbol-token mapping
func (im *instrumentMapper) addMapping(symbol string, token uint32) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.symbolToToken[symbol] = token
	im.tokenToSymbol[token] = symbol
}

// getToken retrieves the token for a symbol
func (im *instrumentMapper) getToken(symbol string) (uint32, bool) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	token, exists := im.symbolToToken[symbol]
	return token, exists
}

// getSymbol retrieves the symbol for a token
func (im *instrumentMapper) getSymbol(token uint32) string {
	im.mu.RLock()
	defer im.mu.RUnlock()

	return im.tokenToSymbol[token]
}

// getAllTokens returns all registered tokens
func (im *instrumentMapper) getAllTokens() []uint32 {
	im.mu.RLock()
	defer im.mu.RUnlock()

	tokens := make([]uint32, 0, len(im.tokenToSymbol))
	for token := range im.tokenToSymbol {
		tokens = append(tokens, token)
	}

	return tokens
}

// clear removes all mappings
func (im *instrumentMapper) clear() {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.symbolToToken = make(map[string]uint32)
	im.tokenToSymbol = make(map[uint32]string)
}
