package zerodha

import (
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"github.com/zerodha/gokiteconnect/v4/models"
)

// TickerEventHandler defines the interface for WebSocket event handling
type TickerEventHandler interface {
	// setupEventHandlers configures all WebSocket event callbacks
	setupEventHandlers()

	// Event handler methods
	onConnect()
	onError(err error)
	onClose(code int, reason string)
	onReconnect(attempt int, delay time.Duration)
	onNoReconnect(attempt int)
	onTick(tick models.Tick)
	onOrderUpdate(order kiteconnect.Order)
}
