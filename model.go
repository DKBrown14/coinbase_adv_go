package coinbase_adv_go

import (
	"net/http"
	"net/url"
	"time"

	// "backend/adapters"

	// "github.com/alexanderjophus/go-broadcast"

	"golang.org/x/time/rate"
)

/*
	From: https://docs.cloud.coinbase.com/advanced-trade-api/docs/ws-best-practices

	Spread subscriptions over more than one WebSocket client connection.
	For example, do not subscribe to BTC-USD and ETH-USD on the same channel
	if possible. Instead, open up two separate WebSocket connections to help
	load balance those inbound messages across separate connections.
*/
/*
From: https://docs.cloud.coinbase.com/advanced-trade-api/docs/ws-channels

	The Advanced Trade API supports the following channels:
	status: 		Sends all products and currencies on a preset interval
	ticker: 		Real-time price updates every time a match happens
	ticker_batch: 	Real-time price updates every 5000 milli-seconds
	level2:			All updates and easiest way to keep order book snapshot
	user: 			Only sends messages that include the authenticated user
	market_trades: 	Real-time updates every time a market trade happens
*/

type CoinbaseAdvanced struct {
	Name        string
	Description string
	client      *http.Client
	apiEndpoint string
	apiBaseURL  string
	rateLimiter *rate.Limiter
	wsURL       string
	wsList      map[string]*WebSocketConnection // The key is the ProductID
	apiKey      string
	apiSecret   string
	// Exchange    *adapters.ExchangeAdapter
}

func NewCoinbaseAdvanced(key, secret string) *CoinbaseAdvanced {
	u := url.URL{Scheme: "wss", Host: "advanced-trade-ws.coinbase.com", Path: ""}
	return &CoinbaseAdvanced{
		Name:        "Coinbase Advanced Trade",
		Description: "Coinbase Advanced Trade API",
		wsURL:       u.String(),
		wsList:      make(map[string]*WebSocketConnection),
		apiBaseURL:  "https://api.coinbase.com",
		apiEndpoint: "/api/v3/brokerage",
		apiKey:      key,
		apiSecret:   secret,
		rateLimiter: rate.NewLimiter(rate.Every(time.Second/10), 1),
		client:      &http.Client{},
	}
}

type Balance struct {
	Currency  string  `json:"currency"`
	Available float64 `json:"available"`
	Reserved  float64 `json:"reserved"`
}

type Ticker struct {
	TradeID   int64     `json:"trade_id"`
	Price     float64   `json:"price"`
	Size      float64   `json:"size"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

type OrderBook struct {
	Bids [][]float64 `json:"bids"`
	Asks [][]float64 `json:"asks"`
}

type Trade struct {
	TradeID int64     `json:"trade_id"`
	Side    string    `json:"side"`
	Price   float64   `json:"price"`
	Size    float64   `json:"size"`
	Time    time.Time `json:"time"`
}

type Account struct {
	UUID             string `json:"uuid"`     // Unique identifier for the account
	Name             string `json:"name"`     // User-defined name for the account
	Currency         string `json:"currency"` // Currency of the account
	AvailableBalance struct {
		Value    string `json:"value"`    // Available balance in the account
		Currency string `json:"currency"` // Currency of the account
	} `json:"available_balance"`
	Default   bool      `json:"default"`    // True if this is the default account for the currency
	Active    bool      `json:"active"`     // True if the account is active
	CreatedAt time.Time `json:"created_at"` // Timestamp indicating when the account was created
	UpdatedAt time.Time `json:"updated_at"` // Timestamp indicating when the account was last updated
	DeletedAt time.Time `json:"deleted_at"` // Timestamp indicating when the account was deleted
	Type      string    `json:"type"`       // Type of the account
	Ready     bool      `json:"ready"`      // True if the account is ready to transact
	Hold      struct {
		Value    string `json:"value"`    // Amount of funds held in the account
		Currency string `json:"currency"` // Currency of the account
	} `json:"hold"` // Amount of funds held in the account
}

type AccountsList struct {
	Accounts []Account `json:"accounts"`
	HasNext  bool      `json:"has_next"`
	Cursor   string    `json:"cursor"`
	Size     int       `json:"size"`
}

type OrdersResponse struct {
	Orders   Order  `json:"orders"`
	Sequence string `json:"sequence"`
	HasNext  bool   `json:"has_next"`
	Cursor   string `json:"cursor"`
}

type OrderList struct {
	Orders  []Order `json:"orders"`
	HasNext bool    `json:"has_next"`
	Cursor  string  `json:"cursor"`
}

type Order struct {
	OrderID              string             `json:"order_id,omitempty"`
	ProductID            string             `json:"product_id,omitempty"`
	UserID               string             `json:"user_id,omitempty"`
	OrderConfiguration   OrderConfiguration `json:"order_configuration,omitempty"`
	Side                 string             `json:"side,omitempty"`
	ClientOrderID        string             `json:"client_order_id,omitempty"`
	Status               string             `json:"status,omitempty"`
	TimeInForce          string             `json:"time_in_force,omitempty"`
	CreatedTime          time.Time          `json:"created_time,omitempty"`
	CompletionPercentage string             `json:"completion_percentage,omitempty"`
	FilledSize           string             `json:"filled_size,omitempty"`
	AverageFilledPrice   string             `json:"average_filled_price,omitempty"`
	Fee                  string             `json:"fee,omitempty"`
	NumberOfFills        string             `json:"number_of_fills,omitempty"`
	FilledValue          string             `json:"filled_value,omitempty"`
	PendingCancel        bool               `json:"pending_cancel,omitempty"`
	SizeInQuote          bool               `json:"size_in_quote,omitempty"`
	TotalFees            string             `json:"total_fees,omitempty"`
	SizeInclusiveOfFees  bool               `json:"size_inclusive_of_fees,omitempty"`
	TotalValueAfterFees  string             `json:"total_value_after_fees,omitempty"`
	TriggerStatus        string             `json:"trigger_status,omitempty"`
	OrderType            string             `json:"order_type,omitempty"`
	RejectReason         string             `json:"reject_reason,omitempty"`
	Settled              bool               `json:"settled,omitempty"`
	ProductType          string             `json:"product_type,omitempty"`
	RejectMessage        string             `json:"reject_message,omitempty"`
	CancelMessage        string             `json:"cancel_message,omitempty"`
	OrderPlacementSource string             `json:"order_placement_source,omitempty"`
}

type OrderConfiguration struct {
	MarketMarketIOC       MarketMarketIOC       `json:"market_market_ioc,omitempty" param:"market_market_ioc"`
	LimitLimitGTC         LimitLimitGTC         `json:"limit_limit_gtc,omitempty" param:"limit_limit_gtc"`
	LimitLimitGTD         LimitLimitGTD         `json:"limit_limit_gtd,omitempty" param:"limit_limit_gtd"`
	StopLimitStopLimitGTC StopLimitStopLimitGTC `json:"stop_limit_stop_limit_gtc,omitempty" param:"stop_limit_stop_limit_gtc"`
	StopLimitStopLimitGTD StopLimitStopLimitGTD `json:"stop_limit_stop_limit_gtd,omitempty" param:"stop_limit_stop_limit_gtd"`
}

type MarketMarketIOC struct {
	QuoteSize string `json:"quote_size" param:"quote_size"`
	BaseSize  string `json:"base_size" param:"base_size"`
}

type LimitLimitGTC struct {
	BaseSize   string `json:"base_size" param:"base_size"`
	LimitPrice string `json:"limit_price" param:"limit_price"`
	PostOnly   bool   `json:"post_only" param:"post_only"`
}

type LimitLimitGTD struct {
	BaseSize   string    `json:"base_size" param:"base_size"`
	LimitPrice string    `json:"limit_price" param:"limit_price"`
	EndTime    time.Time `json:"end_time" param:"end_time"`
	PostOnly   bool      `json:"post_only" param:"post_only"`
}

type StopLimitStopLimitGTC struct {
	BaseSize      string `json:"base_size" param:"base_size"`
	LimitPrice    string `json:"limit_price" param:"limit_price"`
	StopPrice     string `json:"stop_price" param:"stop_price"`
	StopDirection string `json:"stop_direction" param:"stop_direction"`
}

type StopLimitStopLimitGTD struct {
	BaseSize      string    `json:"base_size" param:"base_size"`
	LimitPrice    string    `json:"limit_price" param:"limit_price"`
	StopPrice     string    `json:"stop_price" param:"stop_price"`
	EndTime       time.Time `json:"end_time" param:"end_time"`
	StopDirection string    `json:"stop_direction" param:"stop_direction"`
}

type Product struct {
	Product_id                   string `json:"product_id"`
	Price                        string `json:"price"`
	Price_percentage_change_24h  string `json:"price_percentage_change_24h"`
	Volume_24h                   string `json:"volume_24h"`
	Volume_percentage_change_24h string `json:"volume_percentage_change_24h"`
	Base_increment               string `json:"base_increment"`
	Quote_increment              string `json:"quote_increment"`
	Quote_min_size               string `json:"quote_min_size"`
	Quote_max_size               string `json:"quote_max_size"`
	Base_min_size                string `json:"base_min_size"`
	Base_max_size                string `json:"base_max_size"`
	Base_name                    string `json:"base_name"`
	Quote_name                   string `json:"quote_name"`
	Watched                      bool   `json:"watched"`
	Is_disabled                  bool   `json:"is_disabled"`
	New                          bool   `json:"new"`
	Status                       string `json:"status"`
	Cancel_only                  bool   `json:"cancel_only"`
	Limit_only                   bool   `json:"limit_only"`
	Post_only                    bool   `json:"post_only"`
	Trading_disabled             bool   `json:"trading_disabled"`
	Auction_mode                 bool   `json:"auction_mode"`
	Product_type                 string `json:"product_type"`
	Quote_currency_id            string `json:"quote_currency_id"`
	Base_currency_id             string `json:"base_currency_id"`
	Mid_market_price             string `json:"mid_market_price"`
	Base_display_symbol          string `json:"base_display_symbol"`
	Quote_display_symbol         string `json:"quote_display_symbol"`
}

type ProductList struct {
	Products     []Product `json:"products"`
	Num_products int       `json:"num_products"`
}

type Market_market_ioc struct {
	QuoteSize string `json:"quote_size"`
	BaseSize  string `json:"base_size"`
}

type Limit_limit_gtc struct {
	BaseSize   string `json:"base_size"`
	LimitPrice string `json:"limit_price"`
	PostOnly   bool   `json:"post_only"`
}

type Limit_limit_gtd struct {
	BaseSize   string    `json:"base_size"`
	LimitPrice string    `json:"limit_price"`
	EndTime    time.Time `json:"end_time"`
	PostOnly   bool      `json:"post_only"`
}

type Stop_limit_stop_limit_gtc struct {
	BaseSize      string `json:"base_size"`
	LimitPrice    string `json:"limit_price"`
	StopPrice     string `json:"stop_price"`
	StopDirection string `json:"stop_direction"`
}

type Stop_limit_stop_limit_gtd struct {
	BaseSize      string    `json:"base_size"`
	LimitPrice    string    `json:"limit_price"`
	StopPrice     string    `json:"stop_price"`
	EndTime       time.Time `json:"end_time"`
	StopDirection string    `json:"stop_direction"`
}

type Fill struct {
	Entry_id            string    `json:"entry_id"`
	Trade_id            string    `json:"trade_id"`
	Order_id            string    `json:"order_id"`
	Trade_time          time.Time `json:"trade_time"`
	Trade_type          string    `json:"trade_type"`
	Price               string    `json:"price"`
	Size                string    `json:"size"`
	Commission          string    `json:"commission"`
	Product_id          string    `json:"product_id"`
	Sequence_timestamp  time.Time `json:"sequence_timestamp"`
	Liquidity_indicator string    `json:"liquidity_indicator"`
	Size_in_quote       bool      `json:"size_in_quote"`
	User_id             string    `json:"user_id"`
	Side                string    `json:"side"`
}

type FillList struct {
	Fills  []Fill `json:"fills"`
	Cursor string `json:"cursor"`
}

// func (f FillList) GetFills() []Fill {
// 	return f.Fills
// }

// func (f FillList) GetCursor() string {
// 	return f.Cursor
// }

type FillLister interface {
	GetFills() []Fill
	GetCursor() string
}

type Candles struct {
	Start  string `json:"start"`
	Low    string `json:"low"`
	High   string `json:"high"`
	Open   string `json:"open"`
	Close  string `json:"close"`
	Volume string `json:"volume"`
}

type CandleList struct {
	Candles []Candles `json:"candles"`
}

// Defines the granularity of the candles in minutes
// probably needs a better way to determine if the granularity is valid
// for Coinbase Advanced API.

var Granularity = map[string]time.Duration{
	"UNKNOWN_GRANULARITY": 0 * time.Minute,
	"ONE_MINUTE":          1 * time.Minute,
	"FIVE_MINUTE":         5 * time.Minute,
	"FIFTEEN_MINUTE":      15 * time.Minute,
	"THIRTY_MINUTE":       30 * time.Minute,
	"ONE_HOUR":            60 * time.Minute,
	"TWO_HOUR":            120 * time.Minute,
	"FOUR_HOUR":           240 * time.Minute,
	"SIX_HOUR":            360 * time.Minute,
	"TWELVE_HOUR":         720 * time.Minute,
	"ONE_DAY":             1440 * time.Minute,
	"ONE_WEEK":            10080 * time.Minute,
	"ONE_MONTH":           43200 * time.Minute,
}

var CoinbaseAdvancesValidGranularity = map[string]bool{
	"UNKNOWN_GRANULARITY": true,
	"ONE_MINUTE":          true,
	"FIVE_MINUTE":         true,
	"FIFTEEN_MINUTE":      true,
	"THIRTY_MINUTE":       true,
	"ONE_HOUR":            true,
	"TWO_HOUR":            true,
	"FOUR_HOUR":           false,
	"SIX_HOUR":            true,
	"TWELVE_HOUR":         false,
	"ONE_DAY":             true,
	"ONE_WEEK":            false,
	"ONE_MONTH":           false,
}

type MarketTradeList struct {
	Trades   []MarketTrade `json:"trades"`
	Best_bid string        `json:"best_bid"`
	Best_ask string        `json:"best_ask"`
}

type MarketTrade struct {
	Trade_id   string    `json:"trade_id"`
	Product_id string    `json:"product_id"`
	Price      string    `json:"price"`
	Size       string    `json:"size"`
	Time       time.Time `json:"time"`
	Side       string    `json:"side"`
	Bid        string    `json:"bid"`
	Ask        string    `json:"ask"`
}

type TransactionsSummary struct {
	Total_volume               float64             `json:"total_volume"`
	Total_fees                 float64             `json:"total_fees"`
	Fee_tier                   FeeTier             `json:"fee_tier,omitempty"`
	Margin_rate                MarginRate          `json:"margin_rate,omitempty"`            // only for margin accounts
	Goods_and_services_tax     GoodsAndServicesTax `json:"goods_and_services_tax,omitempty"` // only for margin accounts
	Advanced_trade_only_volume float64             `json:"advanced_trade_only_volume"`
	Advanced_trade_only_fees   float64             `json:"advanced_trade_only_fees"`
	Coinbase_pro_volume        float64             `json:"coinbase_pro_volume"`
	Coinbase_pro_fees          float64             `json:"coinbase_pro_fees"`
}

type FeeTier struct {
	Pricing_tier   string `json:"pricing_tier"`
	Usd_from       string `json:"usd_from"`
	Usd_to         string `json:"usd_to"`
	Taker_fee_rate string `json:"taker_fee_rate"`
	Maker_fee_rate string `json:"maker_fee_rate"`
}

type MarginRate struct {
	Value string `json:"value"`
}

type GoodsAndServicesTax struct {
	Rate string `json:"rate"`
	Type string `json:"type"`
}

// Websocket user channel message
type UserMessage struct {
	Channel     string      `json:"channel"`
	ClientID    string      `json:"client_id"`
	Timestamp   time.Time   `json:"timestamp"`
	SequenceNum int         `json:"sequence_num"`
	Events      []UserEvent `json:"events"`
}

type UserEvent struct {
	Type   string       `json:"type"`
	Orders []UserOrders `json:"orders"`
}

type UserOrders struct {
	OrderID            string    `json:"order_id"`
	ClientOrderIO      string    `json:"client_order_id"`
	CumulativeQuantity string    `json:"cumulative_quantity"`
	LeavesQuantity     string    `json:"leaves_quantity"`
	AverageFilledPrice string    `json:"average_filled_price"`
	TotalFees          string    `json:"total_fees"`
	Status             string    `json:"status"`
	ProductID          string    `json:"product_id"`
	CreationTime       time.Time `json:"creation_time"`
	OrderType          string    `json:"order_type"`
	OrderSide          string    `json:"order_side"`
}

// Following are the structs for the websocket messages
type WsHeartbeatsMessage struct {
	Channel     string              `json:"channel"`
	ClientID    string              `json:"client_id"`
	Timestamp   string              `json:"timestamp"`
	SequenceNum int                 `json:"sequence_num"`
	Events      []WsHeartbeatsEvent `json:"events"`
}

type WsHeartbeatsEvent struct {
	CurrentTime      string `json:"current_time"`
	HeartbeatCounter int    `json:"heartbeat_counter"`
}

type WsCandlesMessage struct {
	Channel     string           `json:"channel"`
	ClientID    string           `json:"client_id"`
	Timestamp   string           `json:"timestamp"`
	SequenceNum int              `json:"sequence_num"`
	Events      []WsCandlesEvent `json:"events"`
}

type WsCandlesEvent struct {
	Type    string     `json:"type"`
	Candles []WsCandle `json:"candles"`
}

type WsCandle struct {
	Start     string `json:"start"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Open      string `json:"open"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
	ProductID string `json:"product_id"`
}

type WsMarketTradesMessage struct {
	Channel     string                `json:"channel"`
	ClientID    string                `json:"client_id"`
	Timestamp   string                `json:"timestamp"`
	SequenceNum int                   `json:"sequence_num"`
	Events      []WsMarketTradesEvent `json:"events"`
}

type WsMarketTradesEvent struct {
	Type   string    `json:"type"`
	Trades []WsTrade `json:"trades"`
}

type WsTrade struct {
	TradeID   string `json:"trade_id"`
	ProductID string `json:"product_id"`
	Price     string `json:"price"`
	Size      string `json:"size"`
	Side      string `json:"side"`
	Time      string `json:"time"`
}

type WsStatusMessage struct {
	Channel     string          `json:"channel"`
	ClientID    string          `json:"client_id"`
	Timestamp   string          `json:"timestamp"`
	SequenceNum int             `json:"sequence_num"`
	Events      []WsStatusEvent `json:"events"`
}

type WsStatusEvent struct {
	Type     string      `json:"type"`
	Products []WsProduct `json:"products"`
}

type WsProduct struct {
	ProductType    string `json:"product_type"`
	ID             string `json:"id"`
	BaseCurrency   string `json:"base_currency"`
	QuoteCurrency  string `json:"quote_currency"`
	BaseIncrement  string `json:"base_increment"`
	QuoteIncrement string `json:"quote_increment"`
	DisplayName    string `json:"display_name"`
	Status         string `json:"status"`
	StatusMessage  string `json:"status_message"`
	MinMarketFunds string `json:"min_market_funds"`
}

type WsTickerMessage struct {
	Channel     string          `json:"channel"`
	ClientID    string          `json:"client_id"`
	Timestamp   string          `json:"timestamp"`
	SequenceNum int             `json:"sequence_num"`
	Events      []WsTickerEvent `json:"events"`
}

type WsTickerEvent struct {
	Type    string     `json:"type"`
	Tickers []WsTicker `json:"tickers"`
}

type WsTicker struct {
	Type               string `json:"type"`
	ProductID          string `json:"product_id"`
	Price              string `json:"price"`
	Volume24H          string `json:"volume_24_h"`
	Low24H             string `json:"low_24_h"`
	High24H            string `json:"high_24_h"`
	Low52W             string `json:"low_52_w"`
	High52W            string `json:"high_52_w"`
	PricePercentChg24H string `json:"price_percent_chg_24_h"`
}

type WsLevel2Message struct {
	Channel     string          `json:"channel"`
	ClientID    string          `json:"client_id"`
	Timestamp   string          `json:"timestamp"`
	SequenceNum int             `json:"sequence_num"`
	Events      []WsLevel2Event `json:"events"`
}

type WsLevel2Event struct {
	Type      string           `json:"type"`
	ProductID string           `json:"product_id"`
	Updates   []WsLevel2Update `json:"updates"`
}

type WsLevel2Update struct {
	Side        string `json:"side"`
	EventTime   string `json:"event_time"`
	PriceLevel  string `json:"price_level"`
	NewQuantity string `json:"new_quantity"`
}

type WsUserMessage struct {
	Channel     string        `json:"channel"`
	ClientID    string        `json:"client_id"`
	Timestamp   string        `json:"timestamp"`
	SequenceNum int           `json:"sequence_num"`
	Events      []WsUserEvent `json:"events"`
}

type WsUserEvent struct {
	Type   string    `json:"type"`
	Orders []WsOrder `json:"orders"`
}

type WsOrder struct {
	OrderID            string `json:"order_id"`
	ClientOrderID      string `json:"client_order_id"`
	CumulativeQuantity string `json:"cumulative_quantity"`
	LeavesQuantity     string `json:"leaves_quantity"`
	AvgPrice           string `json:"avg_price"`
	TotalFees          string `json:"total_fees"`
	Status             string `json:"status"`
	ProductID          string `json:"product_id"`
	CreationTime       string `json:"creation_time"`
	OrderSide          string `json:"order_side"`
	OrderType          string `json:"order_type"`
}

type WsSubscriptionMessage struct {
	Channel     string                `json:"channel"`
	ClientID    string                `json:"client_id"`
	Timestamp   string                `json:"timestamp"`
	SequenceNum int                   `json:"sequence_num"`
	Events      []WsSubscriptionEvent `json:"events"`
}

type WsSubscriptionEvent struct {
	Subscriptions WsSubscriptions `json:"subscriptions"`
}

type WsSubscriptions struct {
	User []string `json:"user"`
}
