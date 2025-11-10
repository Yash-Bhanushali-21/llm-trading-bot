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
