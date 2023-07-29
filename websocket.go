package coinbase_adv_go

import (
	"fmt"
	"strings"

	"github.com/alexanderjophus/go-broadcast"
	"github.com/gorilla/websocket"
)

// Subscribe subscribes to a websocket channel for a given set of product IDs and streams data
// to the caller's channel.
// One broadcaster is created per productID set. If multiple subscriptions are made to the same
// productID set, the broadcaster is shared between them. This allows for multiple subscriptions to the
// same productIDs to be handled by a single broadcaster.
//
// If the productIDs are not already subscribed to, a new connection is created and the productIDs are subscribed to.
// A broadcaster is created and the caller's channel is registered to it.
// If the productIDs are already subscribed to, the caller's channel is registered to the existing broadcaster.
//
// Parameters:
//
//	stream     - the caller's channel that data will be streamed to
//	channel    - the name of the websocket channel to subscribe to ie. "ticker"
//	productIDs - a slice of product IDs to subscribe to. ie. []string{"BTC-USD", "ETH-USD"}
//
// If channel is 'user' ProductIDs can be nil, if this is the case, the user's account will be
// subscribed to all products and ProductIDs will internally be "ALL".
//
// Returns:
//
//	stream     - a broadcast channel that data will be streamed to
//	err        - an error if there was a problem subscribing to the channel
func (c *CoinbaseAdvanced) Subscribe(stream chan string, channel string, productIDs []string) (chan string, error) {
	if channel == "" {
		return nil, fmt.Errorf("channel not specified")
	}
	if productIDs == nil || len(productIDs) == 0 {
		productIDs = []string{"ALL"}
	}

	key := strings.Join(productIDs, ",")
	connect, ok := c.wsList[key]
	if !ok {
		conn, _, err := websocket.DefaultDialer.Dial(c.wsURL, nil)
		if err != nil {
			return nil, err
		}
		connect = &webSocketConnection{
			conn:         conn,
			subscribedTo: map[string]*subscribed{},
		}
	}

	sub, ok := connect.subscribedTo[channel]
	if ok {
		sub.broadcaster.Register(stream)
		return stream, nil
	}

	sub.broadcaster = broadcast.NewBroadcaster[string](5)
	connect.subscribedTo[channel] = sub
	sub.broadcaster.Register(stream)

	subscribeMsg := map[string]interface{}{
		"type":        "subscribe",
		"channel":     channel,
		"api_key":     c.apiKey,
		"product_ids": productIDs,
	}
	subscribeMsg = c.timestampAndSign(subscribeMsg, channel, productIDs)
	err := connect.conn.WriteJSON(subscribeMsg)
	if err != nil {
		return nil, err
	}

	go readAndWrite(connect.conn, sub, &subscribers{sub, stream})

	return stream, nil
}

// Unsubscribe unsubscribes from a websocket channel for a given set of product IDs.
// If there are multiple subscriptions to the same channel, the subscription count is decremented.
// If there is only one subscription, the channel is deleted.
// If there are no more subscriptions, the connection is closed and deleted.
func (c *CoinbaseAdvanced) Unsubscribe(channel string, productIDs []string) error {
	key := strings.Join(productIDs, ",")
	if connect, ok := c.wsList[key]; ok {
		if sub, ok := connect.subscribedTo[channel]; ok && sub.channel != nil {
			// Subscription is valid
			// If there are multiple subscriptions to the same channel, decrement the count
			// If there is only one subscription, delete the channel
			if sub.subCount > 1 {
				sub.subCount--
			} else {
				unsubscribeMsg := map[string]interface{}{
					"type":        "unsubscribe",
					"channel":     channel,
					"api_key":     c.apiKey,
					"product_ids": productIDs,
				}
				unsubscribeMsg = c.timestampAndSign(unsubscribeMsg, channel, productIDs)
				err := connect.conn.WriteJSON(unsubscribeMsg)
				if err != nil {
					return err
				}
				delete(connect.subscribedTo, channel)
			}
			// If there are no more subscriptions, close the connection and delete the connection
			if len(connect.subscribedTo) == 0 {
				connect.conn.Close()
				delete(c.wsList, key)
			}
		} else {
			// Subscription is not valid
			return fmt.Errorf("subscription not found")
		}
	} else {
		// Connection not found
		return fmt.Errorf("connection not found")
	}

	return nil
}
