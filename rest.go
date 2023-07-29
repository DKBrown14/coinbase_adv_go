package coinbase_adv_go

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ListAccountParams struct {
	Limit  int32  `param:"limit"`
	Cursor string `param:"cursor"`
}

// List of REST API endpoints
func (c *CoinbaseAdvanced) listAccounts(ctx context.Context, params *ListAccountParams) (AccountsList, error) {

	/**
	  https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_getaccounts

	  Get a list of authenticated accounts for the current user.
	  local function to get a paginated list of accounts.

	  Args: (in ListAccountParams)
	  - limit: A pagination limit with default of 49 and maximum of 250.
	  	   If has_next is true, additional orders are available to be fetched
	  	   with pagination and the cursor value in the response can be passed
	  	   as cursor parameter in the subsequent request.

	  - cursor: Cursor used for pagination. When provided, the response returns
	  		  responses after this cursor.

	  Returns:
		{
		"accounts": {
			"uuid": "8bfc20d7-f7c6-4422-bf07-8243ca4169fe",
			"name": "BTC Wallet",
			"currency": "BTC",
			"available_balance": {
				"value": "1.23",
				"currency": "BTC"
			},
			"default": false,
			"active": true,
			"created_at": "2021-05-31T09:59:59Z",
			"updated_at": "2021-05-31T09:59:59Z",
			"deleted_at": "2021-05-31T09:59:59Z",
			"type": "ACCOUNT_TYPE_UNSPECIFIED",
			"ready": true,
			"hold": {
				"value": "1.23",
				"currency": "BTC"
			}
		},
		"has_next": true,
		"cursor": "789100",
		"size": "integer"
		}

		**/

	requestPath := "/accounts"
	query_params := url.Values{}

	fail := c.addParamsToQuery(params, query_params)
	if fail != nil {
		return AccountsList{}, fail
	}

	resp, err := c.doRequest(ctx, "GET", requestPath, query_params, nil)
	if err != nil {
		return AccountsList{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AccountsList{}, err
	}
	var al AccountsList

	err = json.Unmarshal(body, &al)
	if err != nil {
		return AccountsList{}, err
	}

	return al, nil
}

func (c *CoinbaseAdvanced) ListAccounts(ctx context.Context) (AccountsList, error) {
	/**
	  This function overrides the default ListAccounts function
	  to return all accounts in a single call.
	*/
	var al AccountsList
	al.Cursor = ""
	al.HasNext = true
	al.Size = 0

	params := ListAccountParams{
		Limit:  250,
		Cursor: al.Cursor,
	}

	for {
		params.Cursor = al.Cursor
		accounts, err := c.listAccounts(ctx, &params)
		if err != nil {
			return AccountsList{}, err
		}
		// copy accounts to al
		al.Accounts = append(al.Accounts, accounts.Accounts...)
		al.Cursor = accounts.Cursor
		al.Size += accounts.Size
		al.HasNext = accounts.HasNext
		if !accounts.HasNext {
			break
		}
	}
	return al, nil
}

type GetAccountResponse struct {
	Account Account `json:"account"`
}

type GetAccountParams struct {
	AccountID string `param:"account_id,-"`
}

func (c *CoinbaseAdvanced) GetAccount(ctx context.Context, params *GetAccountParams) (Account, error) {
	/**
	  https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_getaccount

	  Get a single account for the current user.

	  Args:
	  - accountID: The account ID

	  Returns:
		{
		"account": {
			"uuid": "8bfc20d7-f7c6-4422-bf07-8243ca4169fe",
			"name": "BTC Wallet",
			"currency": "BTC",
			"available_balance": {
				"value": "1.23",
				"currency": "BTC"
			},
			"default": false,
			"active": true,
			"created_at": "2021-05-31T09:59:59Z",
			"updated_at": "2021-05-31T09:59:59Z",
			"deleted_at": "2021-05-31T09:59:59Z",
			"type": "ACCOUNT_TYPE_UNSPECIFIED",
			"ready": true,
			"hold": {
				"value": "1.23",
				"currency": "BTC"
			}
		}
		}

		**/
	if params.AccountID == "" {
		return Account{}, errors.New("accountID is required")
	}
	requestPath := "/accounts/" + params.AccountID

	resp, err := c.doRequest(ctx, "GET", requestPath, nil, nil)
	if err != nil {
		return Account{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Account{}, err
	}
	var a GetAccountResponse

	err = json.Unmarshal(body, &a)
	if err != nil {
		return Account{}, err
	}

	return a.Account, nil
}

type CreateOrderParams struct {
	ClientOrderId      string             `param:"client_order_id,required"`
	ProductId          string             `param:"product_id,required"`
	Side               string             `param:"side,required"`
	OrderConfiguration OrderConfiguration `param:"order_configuration,required"`
}

// Not fully implemented
func (c *CoinbaseAdvanced) CreateOrder(ctx context.Context, params *CreateOrderParams) (interface{}, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_createorder

	Create a new order.

	Args:
	- client_order_id: The client order ID ***required
	- product_id: The product ID ***required
	- side: The order side
	- order_configuration: The order configuration
	**/
	requestPath := "/orders"
	query_params := url.Values{}
	fail := c.addParamsToQuery(params, query_params)
	if fail != nil {
		return nil, fail
	}

	resp, err := c.doRequest(ctx, "POST", requestPath, nil, strings.NewReader(query_params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var o Order

	err = json.Unmarshal(body, &o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

// Not fully implemented
func (c *CoinbaseAdvanced) CancelOrders(ctx context.Context, order_ids []string) (interface{}, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_cancelorders

	Cancel one or more orders.

	Args:
	- order_ids: The order IDs to cancel
	**/
	requestPath := "/orders/batchcancel"
	query_params := url.Values{}
	if order_ids != nil {
		order_ids_json, err := json.Marshal(order_ids)
		if err != nil {
			return nil, err
		}
		if len(order_ids) == 0 {
			return nil, errors.New("order_ids is required")
		}
		query_params.Add("order_ids", string(order_ids_json))
	}

	resp, err := c.doRequest(ctx, "POST", requestPath, nil, strings.NewReader(query_params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var o Order

	err = json.Unmarshal(body, &o)
	if err != nil {
		return nil, err
	}

	return o, nil

}

type ListOrdersParams struct {
	ProductID            string    `param:"product_id"`
	OrderStatus          string    `param:"order_status"`
	Limit                int       `param:"limit"`
	StartDate            time.Time `param:"start_date"`
	EndDate              time.Time `param:"end_date"`
	UserNativeCurrency   string    `param:"user_native_currency"`
	OrderType            string    `param:"order_type"`
	OrderSide            string    `param:"order_side"`
	Cursor               string    `param:"cursor"`
	ProductType          string    `param:"product_type"`
	OrderPlacementSource string    `param:"order_placement_source"`
}

func (c *CoinbaseAdvanced) listOrders(ctx context.Context, params *ListOrdersParams) (OrderList, error) {
	/**
		https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_gethistoricalorders

		Get a list of orders filtered by optional query parameters (product_id, order_status, etc).

		Args:
		- product_id: Optional string of the product ID.
					  Defaults to null, or fetch for all products.
		- order_status: A list of order statuses.
		- limit: A pagination limit with no default set.
				 If has_next is true, additional orders are available
				 to be fetched with pagination; also the cursor value
				 in the response can be passed as cursor parameter in
				 the subsequent request.
		- start_date: Start date to fetch orders from, inclusive.
		- end_date: An optional end date for the query window, exclusive.
					If provided only orders with creation time before
					this date will be returned.
		- user_native_currency: String of the users native currency. Default is USD.
		- order_type: Type of orders to return. Default is to return all order types.
			- MARKET: A market order
			- LIMIT: A limit order
			- STOP: A stop order is an order that becomes a market order when triggered
			- STOP_LIMIT: A stop order is a limit order that doesn't go on the book until
						  it hits the stop price.
		- order_side: Only orders matching this side are returned. Default is to return all sides.
		- cursor: Cursor used for pagination.
				  When provided, the response returns responses after this cursor.
		- product_type: Only orders matching this product type are returned.
						Default is to return all product types.
		- order_placement_source: String. Only orders matching this placement source are returned.
								  Default is to return RETAIL_ADVANCED placement source.
	**/

	requestPath := "/orders/historical/batch"
	query_params := url.Values{}

	//**** add params to query_params as as helper function
	c.addParamsToQuery(params, query_params)

	resp, err := c.doRequest(ctx, "GET", requestPath, query_params, nil)
	if err != nil {
		return OrderList{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return OrderList{}, err
	}
	var orderList OrderList

	err = json.Unmarshal(body, &orderList)
	if err != nil {
		return OrderList{}, err
	}

	return orderList, nil
}

// Iterate through all orders if HasNext is true
func (c *CoinbaseAdvanced) ListOrders(ctx context.Context, params *ListOrdersParams) (OrderList, error) {
	orderList := OrderList{}
	for {
		response, err := c.listOrders(ctx, params)
		if err != nil {
			return OrderList{}, err
		}
		orderList.Orders = append(orderList.Orders, response.Orders...)
		orderList.Cursor = response.Cursor
		orderList.HasNext = response.HasNext
		params.Cursor = response.Cursor
		if !response.HasNext {
			break
		}
	}
	return orderList, nil
}

type ListFillsParams struct {
	OrderID                string    `param:"order_id"`
	ProductID              string    `param:"product_id"`
	StartSequenceTimestamp time.Time `param:"start_sequence_timestamp,required"`
	EndSequenceTimestamp   time.Time `param:"end_sequence_timestamp,required"`
	Limit                  int64     `param:"limit,,100"`
	Cursor                 string    `param:"cursor"`
}

func (c *CoinbaseAdvanced) listFills(ctx context.Context, params *ListFillsParams) (FillList, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_getfills

	Get a list of fills filtered by optional query parameters (product_id, order_id, etc).
	Possibly a paginated response. Not exported due to an override being available that
	returns all fills for a given order.

	Args:
	- order_id: Optional string of the order ID.
				Defaults to null, or fetch for all orders.
	- product_id: Optional string of the product ID.
				  Defaults to null, or fetch for all products.
	- start_sequence_timestamp: Start date to fetch fills from, inclusive.
	- end_sequence_timestamp: An optional end date for the query window, exclusive.
							  If provided only fills with creation time before
							  this date will be returned.
	- limit: A pagination limit with no default set.
			 If has_next is true, additional fills are available
			 to be fetched with pagination; also the cursor value
			 in the response can be passed as cursor parameter in
			 the subsequent request.
	- cursor: Cursor used for pagination.
			  When provided, the response returns responses after this cursor.
	**/

	requestPath := "/orders/historical/fills"
	query_params := url.Values{}

	c.addParamsToQuery(params, query_params)

	resp, err := c.doRequest(ctx, "GET", requestPath, query_params, nil)
	if err != nil {
		return FillList{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FillList{}, err
	}
	var fillList FillList

	err = json.Unmarshal(body, &fillList)
	if err != nil {
		return FillList{}, err
	}

	return fillList, nil
}

func (c *CoinbaseAdvanced) ListFills(ctx context.Context, params *ListFillsParams) (FillList, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_getfills

	Get a list of fills filtered by optional query parameters (product_id, order_id, etc).
	loop until Cursor is empty

	Args:
	- order_id: Optional string of the order ID.
				Defaults to null, or fetch for all orders.
	- product_id: Optional string of the product ID.
				  Defaults to null, or fetch for all products.
	- start_sequence_timestamp: Start date to fetch fills from, inclusive.
	- end_sequence_timestamp: An optional end date for the query window, exclusive.
							  If provided only fills with creation time before
							  this date will be returned.
	- limit: A pagination limit with no default set.
			 If has_next is true, additional fills are available
			 to be fetched with pagination; also the cursor value
			 in the response can be passed as cursor parameter in
			 the subsequent request.
	- cursor: Cursor used for pagination.
			  When provided, the response returns responses after this cursor.
	**/

	// var fills FillList
	fills := FillList{}
	for {
		response, err := c.listFills(ctx, params)
		if err != nil {
			return FillList{}, err
		}
		fills.Cursor = response.Cursor
		fills.Fills = append(fills.Fills, response.Fills...)
		if response.Cursor == "" {
			break
		}
	}

	return fills, nil
}

type GetOrderResponse struct {
	Order Order `json:"order"`
}

func (c *CoinbaseAdvanced) GetOrder(ctx context.Context, order_id string) (Order, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_getorder

	Get a single order by order ID.

	Args:
	- order_id: The order ID
	**/
	if order_id == "" {
		return Order{}, errors.New("order_id is required")
	}
	requestPath := "/orders/historical/" + order_id

	resp, err := c.doRequest(ctx, "GET", requestPath, nil, nil)
	if err != nil {
		return Order{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Order{}, err
	}
	var o GetOrderResponse

	err = json.Unmarshal(body, &o)
	if err != nil {
		return Order{}, err
	}

	return o.Order, nil
}

type ListProductsParams struct {
	Limit       int32  `param:"limit"`
	Offset      int32  `param:"offset"`
	ProductType string `param:"product_type"`
}

func (c *CoinbaseAdvanced) ListProducts(ctx context.Context, params *ListProductsParams) (ProductList, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_listproducts

	Get a list of products.

	Args:
	- limit: A pagination limit with no default set.
			 If has_next is true, additional products are available
			 to be fetched with pagination; also the cursor value
			 in the response can be passed as cursor parameter in
			 the subsequent request.
	- offset: An optional offset for the query window.
	- product_type: Only products matching this product type are returned.
					Default is to return all product types.
	**/
	requestPath := "/products"
	query_params := url.Values{}

	c.addParamsToQuery(params, query_params)

	resp, err := c.doRequest(ctx, "GET", requestPath, query_params, nil)
	if err != nil {
		return ProductList{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ProductList{}, err
	}
	var productList ProductList

	err = json.Unmarshal(body, &productList)
	if err != nil {
		return ProductList{}, err
	}

	return productList, nil
}

func (c *CoinbaseAdvanced) GetProduct(ctx context.Context, product_id string) (interface{}, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_getproduct

	Get a single product by product ID.

	Args:
	- product_id: The product ID
	**/
	requestPath := "/products/" + product_id
	query_params := url.Values{}
	if product_id != "" {
		query_params.Add("product_id", product_id)
	}

	resp, err := c.doRequest(ctx, "GET", requestPath, query_params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var p Product

	err = json.Unmarshal(body, &p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

type GetProductCandlesParams struct {
	Product_id  string `param:"product_id,-"`
	Start       string `param:"start"`
	End         string `param:"end"`
	Granularity string `param:"granularity"`
}

func (c *CoinbaseAdvanced) getProductCandles(ctx context.Context, params *GetProductCandlesParams) (CandleList, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_getproductcandles

	Get a list of candles for a product.

	Args:
	- product_id: The product ID
	- start string required: Timestamp for starting range of aggregations, in UNIX time.
	- end string required: Timestamp for ending range of aggregations, in UNIX time.
	- granularity string required: The time slice value for each candle.
				   Valid values are {"ONE_MINUTE", "FIVE_MINUTE", "FIFTEEN_MINUTE",
				   "THIRTY_MINUTE", "ONE_HOUR", "TWO_HOUR", "SIX_HOUR", "ONE_DAY" }.
	**/
	var requestPath string
	if params.Product_id != "" {
		requestPath = "/products/" + params.Product_id + "/candles"
	} else {
		return CandleList{}, errors.New("product_id is required")
	}
	query_params := url.Values{}

	c.addParamsToQuery(params, query_params)

	resp, err := c.doRequest(ctx, "GET", requestPath, query_params, nil)
	if err != nil {
		return CandleList{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CandleList{}, err
	}
	var candleList CandleList

	err = json.Unmarshal(body, &candleList)
	if err != nil {
		return CandleList{}, err
	}

	return candleList, nil
}

func (c *CoinbaseAdvanced) GetProductCandles(ctx context.Context, params *GetProductCandlesParams) (CandleList, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_getproductcandles

	Get all requested candles for a product by making multiple requests.
	Requests are normally limited to 300 candles per request, but this function
	will make multiple requests to get all candles in the requested range.
	Args:
	- product_id: The product ID
	- start string required: Timestamp for starting range of aggregations, in UNIX time.
	- end string required: Timestamp for ending range of aggregations, in UNIX time.
	- granularity string required: The time slice value for each candle.
				   Valid values are {"ONE_MINUTE", "FIVE_MINUTE", "FIFTEEN_MINUTE",
				   "THIRTY_MINUTE", "ONE_HOUR", "TWO_HOUR", "SIX_HOUR", "ONE_DAY" }.

	**/
	if params.Product_id == "" {
		return CandleList{}, errors.New("product_id is required")
	}

	stepSizeInMinutes, valid := Granularity[params.Granularity]
	if !valid {
		return CandleList{}, errors.New("invalid granularity")
	}

	// limit to 300 candles per request
	maxCandlesPerRequest := 300

	var rangeInMinutes time.Duration = (stepSizeInMinutes * time.Duration(maxCandlesPerRequest))
	var candleList = CandleList{}

	// run through from most recent to oldest to preserve time order in list
	endDateInt, err := strconv.Atoi(params.End)
	if err != nil {
		return CandleList{}, err
	}
	endDate := time.Unix(int64(endDateInt), 0)

	startDateInt, err := strconv.Atoi(params.Start)
	if err != nil {
		return CandleList{}, err
	}
	start := time.Unix(int64(startDateInt), 0)

	for endDate.After(start) || endDate.Equal(start) {
		startDate := endDate.Add(-rangeInMinutes)
		if startDate.Before(start) {
			startDate = start
		}

		params.Start = strconv.FormatInt(startDate.Unix(), 10)
		params.End = strconv.FormatInt(endDate.Unix(), 10)

		candles, err := c.getProductCandles(ctx, params)
		if err != nil {
			return CandleList{}, err
		}
		candleList.Candles = append(candleList.Candles, candles.Candles...)
		endDate = startDate.Add(-time.Minute)
	}

	return candleList, nil

}

type GetMarketTradesParams struct {
	Product_id string `param:"product_id,-"`
	Limit      int    `param:"limit,required,100"`
}

func (c *CoinbaseAdvanced) GetMarketTrades(ctx context.Context, params *GetMarketTradesParams) (MarketTradeList, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_getmarkettrades

	Get a list of market trades for a product.

	Args:
	- product_id string required: The product ID
	- limit int required: The number of trades to return. Default is 100. Max is 1000.
	**/
	requestPath := "/products/" + params.Product_id + "/ticker"
	query_params := url.Values{}
	c.addParamsToQuery(params, query_params)

	resp, err := c.doRequest(ctx, "GET", requestPath, query_params, nil)
	if err != nil {
		return MarketTradeList{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return MarketTradeList{}, err
	}
	var marketTradeList MarketTradeList

	err = json.Unmarshal(body, &marketTradeList)
	if err != nil {
		return MarketTradeList{}, err
	}

	return marketTradeList, nil
}

type GetTransactionsSummaryParams struct {
	StartDate          time.Time `param:"start_date,required"`
	EndDate            time.Time `param:"end_date,required"`
	UserNativeCurrency string    `param:"user_native_Currency,,USD"`
	ProductType        string    `param:"product_type,required,SPOT"`
}

func (c *CoinbaseAdvanced) GetTransactionsSummary(ctx context.Context, params *GetTransactionsSummaryParams) (TransactionsSummary, error) {
	/**
	https://docs.cloud.coinbase.com/advanced-trade-api/reference/retailbrokerageapi_gettransactionssummary

	Get a summary of transactions for a user.

	Args:
	- start_date string required: Timestamp for starting range of transactions, in UNIX time.
	- end_date string required: Timestamp for ending range of transactions, in UNIX time.
	- user_native_Currency string required: The user's native currency.
	- product_type string required: The product type.
	**/
	requestPath := "/transaction_summary"
	query_params := url.Values{}
	err := c.addParamsToQuery(params, query_params)
	if err != nil {
		return TransactionsSummary{}, err
	}

	resp, err := c.doRequest(ctx, "GET", requestPath, query_params, nil)
	if err != nil {
		return TransactionsSummary{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TransactionsSummary{}, err
	}
	var transactionsSummary TransactionsSummary

	err = json.Unmarshal(body, &transactionsSummary)
	if err != nil {
		return TransactionsSummary{}, err
	}

	return transactionsSummary, nil
}
