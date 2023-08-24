# Coinbase Advanced Go Client

This is a Go client library for the Coinbase Advanced Trade API. It provides a simple interface for accessing real-time market data and trading on the Coinbase Advanced exchange.

## Installation

To install the library, use the following command:

```bash
go get github.com/DKBrown14/coinbase-advanced-go 
```

## Usage

To use the library, import the github.com/your-username/coinbase-advanced-go package in your Go code.

Here's an example of how to use the library to subscribe to the ticker channel:

```go
package main

import (
    "context"
    "fmt"
    "github.com/DKBrown14/coinbase-advanced-go"
)

func main() {
    // create a new client with your API key and secret
    client, err := coinbase_adv_go.NewClient("your-api-key", "your-api-secret")
    if err != nil {
        fmt.Println("Error creating client:", err)
        return
    }
    // get a list of all products
    productParams := &coinbase_adv_go.ListProductParams{
        product_type: "SPOT",
    }
    products, err := client.ListProducts(context.Background(), productParams)
    if err != nil {
        fmt.Println("Error getting products:", err)
        return
    }
    fmt.Println("Products:")
    for _, product := range products {
        fmt.Println(product.ID)
    }
    
    // Create a new WebSocket connection
    wsConn := coinbase_adv_go.NewWebSocketConnection([]string{"BTC-USD", "ETH-USD"})

    // Subscribe to the ticker channel
    tickerOutput := make(chan []byte)
    tickerParams := &coinbase_adv_go.ChannelParams{
        ChannelName: "ticker",
        Output:      tickerOutput,
    }
    if err := wsConn.Subscribe(context.Background(), tickerParams); err != nil {
        fmt.Println("Error subscribing to ticker channel:", err)
        return
    }

    // Read messages from the ticker output channel
    for msg := range tickerOutput {
        fmt.Println("Received ticker message:", string(msg))
    }

    // Unsubscribe from the ticker channel
    if err := wsConn.Unsubscribe(context.Background(), tickerParams); err != nil {
        fmt.Println("Error unsubscribing from ticker channel:", err)
        return
    }

    // Close the WebSocket connection
    if err := wsConn.Close(); err != nil {
        fmt.Println("Error closing WebSocket connection:", err)
        return
    }
}
```

## API Reference

The library provides the following types:

WebSocketConnection: A WebSocket connection to the Coinbase Advanced Trade API.
ChannelParams: Parameters for subscribing to a WebSocket channel.
The library provides the following functions:

```go
NewWebSocketConnection(productIDs []string) *WebSocketConnection: Creates a new WebSocket connection to the Coinbase 
```

## Advanced Trade API

```go
(*WebSocketConnection) Subscribe(ctx context.Context, sub *ChannelParams) error: Subscribes to a WebSocket channel.
(*WebSocketConnection) Unsubscribe(ctx context.Context, sub *ChannelParams) error: Unsubscribes from a WebSocket channel.
(*WebSocketConnection) Close() error: Closes the WebSocket connection.
```

## License

This library is licensed under the MIT License. See the LICENSE file for details.
