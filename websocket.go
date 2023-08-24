package coinbase_adv_go

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSockets
// Subscribe() - Subscribe to a websocket channel for a given set of product IDs and streams data to the caller's channel.
//   Each caller must have its own channel.
// Each ProductID slice has its own websocket connection. If multiple subscriptions are made to the same ProductID set, the
//   dispatcher is shared between them.
// Each websocket channel has its own dispatcher. If multiple subscriptions are made to the same channel, the dispatcher is
// shared between them.

// Unsubscribe() - Unsubscribe from a websocket channel for a given set of product IDs.

// WebSocketConnection represents a websocket connection to the Coinbase Advanced API.
type WebSocketConnection struct {
	Cba         *CoinbaseAdvanced                       // The CoinbaseAdvanced object that created this websocket connection
	conn        *websocket.Conn                         // The websocket connection
	channelList *ChannelMap[chan<- interface{}, string] // The key is the go channel, the string is the websocketchannel that is subscribed to ie. "ticker"
	mutex       *sync.RWMutex                           // Mutex for synchronizing access to subscriptions
	productIDs  []string                                // The product IDs for this websocket connection
	shutdown    chan struct{}                           // The channel to close the dispatcher goroutine
}

// ChannelParams represents the parameters for a websocket channel subscription.
type ChannelParams struct {
	ChannelName string             // the name of the websocket channel
	Output      chan<- interface{} // the output go channel for this websocket channel
}

// Channel represents a channel subscription for an application routine.
// type Channel struct {
// 	ChannelName string                          // the name of the websocket channel
// 	mux         *sync.RWMutex                   // the mutex for this websocket channel
// 	Outputs     map[chan<- string]chan<- string // the output go channels for this websocket channel
// }

// TODO: Rewrite the dispatcher to allow multiple subscriptions to the same go channel.
//
//	This will require a change to the Channel struct to allow multiple output channels.
//	Also allow certain channels to be filtered from the output channel. (ie. heartbeat)
//
// dispatcher reads messages from the websocket connection and sends them to the appropriate output channels.
// The function runs in a separate goroutine and returns when the shutdown channel is closed.
// The output channels are NOT closed when the function returns.
// The messages are filtered by websocket channel and sent to the appropriate output channels.
// Multiple dispatchers may not be registered to the same output channel. (not enforced, caller beware)
// Parameters:
//
//	ws - a pointer to a WebSocketConnection object representing the websocket connection
func dispatcher(ws *WebSocketConnection) {
	for {
		select {
		case <-ws.shutdown:
			return

		default:
			// log.Printf("Dispatcher: Product IDs: %v", ws.productIDs)
			// read the message
			_, message, err := ws.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Printf("Dispatcher: %v Normal closure: %v", ws.productIDs, err)
					return
				} else if websocket.IsCloseError(err, websocket.CloseGoingAway) {
					log.Printf("Dispatcher: %v Going away: %v", ws.productIDs, err)
					if err = ws.restartWS(); err != nil {
						log.Printf("Dispatcher: Error restarting websocket connection: %v", err)
						return
					}
					continue
				} else if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
					log.Printf("Dispatcher: %v Abnormal closure: %v", ws.productIDs, err)
					if err = ws.restartWS(); err != nil {
						log.Printf("Dispatcher: Error restarting websocket connection: %v", err)
						return
					}
					continue
				} else if websocket.IsCloseError(err, websocket.CloseNoStatusReceived) {
					log.Printf("Dispatcher: %v No status received: %v", ws.productIDs, err)
					if err = ws.restartWS(); err != nil {
						log.Printf("Dispatcher: Error restarting websocket connection: %v", err)
						return
					}
					continue
				}
				// TODO: Currently, just log the error and continue.
				// Need to restart the websocket connection and resubscribe to the channels.
				log.Printf("Dispatcher: Error reading message: %v", err)
				if err = ws.restartWS(); err != nil {
					log.Printf("Dispatcher: Error restarting websocket connection: %v", err)
					return
				}
				continue
			}

			// unmarshal the message into a map
			var data map[string]interface{}
			if err := json.Unmarshal(message, &data); err != nil {
				log.Printf("Dispatcher: Error unmarshalling message: %v", err)
				continue
			}

			// get the channel name from the message
			channelName, ok := data["channel"].(string)
			if !ok || len(channelName) == 0 {
				log.Printf("Dispatcher: Error: invalid channel name: %v", channelName)
				log.Printf("Dispatcher: Message: %v", string(message))
				continue
			}

			// log.Printf("Dispatcher: Channel: %v", channelName)

			// unmarshal the message into the appropriate struct based on the channel name
			var msg interface{}
			switch channelName {
			case "heartbeats":
				msg = new(WsHeartbeatsMessage)
			case "ticker", "ticker_batch":
				msg = new(WsTickerMessage)
			case "level2":
				msg = new(WsLevel2Message)
			case "candles":
				msg = new(WsCandlesMessage)
			case "status":
				msg = new(WsStatusMessage)
			case "user":
				msg = new(WsUserMessage)
			case "market_trades":
				msg = new(WsMarketTradesMessage)
			case "subscriptions":
				msg = new(WsSubscriptionMessage)
			default:
				log.Printf("Dispatcher: Error: unknown channel name: %v", channelName)
				log.Printf("Dispatcher: Message: %v", string(message))
				continue
			}

			// log.Printf("Dispatcher: Message type is: %s", reflect.TypeOf(msg))
			// unmarshal the message into the appropriate struct
			if err := json.Unmarshal(message, msg); err != nil {
				log.Printf("Dispatcher: Error unmarshalling message: %v", err)
				log.Printf("Dispatcher: Message: %v", string(message))
				continue
			}
			// log.Printf("Dispatcher: Message: %v", msg)

			// get the output channels for the websocket channel
			outputs := ws.channelList.GetKeysForValue(channelName)

			// log.Printf("Dispatcher: Channel Name: %v", channelName)
			// send the message to the output channels
			for _, output := range outputs {
				// log.Printf("Dispatcher: Output: %v", output)
				output <- msg
			}
		}
	}
}

// Subscribe subscribes to a websocket channel for a given set of product IDs and streams data to the caller's channel.
// Each caller must have its own channel.
// Each ProductID slice has its own websocket connection. If multiple subscriptions are made to the same ProductID set, the
//   dispatcher is shared between them.
// Each websocket channel has its own dispatcher. If multiple subscriptions are made to the same channel, the dispatcher is
// shared between them.
// Note: On the creation of a new websocket connection, the heartbeat channel is automatically subscribed to.
// Parameters:
//   ctx        - a context.Context object
//   sub        - a pointer to a ChannelParams object representing the websocket channel subscription
//   productIDs - a slice of strings representing the product IDs to subscribe to
// Returns:
//   webSocket - a pointer to a WebSocketConnection object representing the websocket connection
//   err       - an error object

func (c *CoinbaseAdvanced) Subscribe(ctx context.Context, sub *ChannelParams, productIDs ...string) (webSocket *WebSocketConnection, err error) {
	var (
		wsKey        string = ""
		newWebSocket bool   = false
	)
	// validate parameters
	if sub == nil {
		return nil, errors.New("subscription is nil")
	}
	if len(productIDs) == 0 {
		wsKey = "ALL"
	} else if len(productIDs) == 1 {
		wsKey = productIDs[0]
	} else {
		wsKey = strings.Join(productIDs, ",")
	}
	// does the websocket connection already exist for this product ID set?
	wsConnection, ok := c.wsList[wsKey]
	if !ok {
		newWebSocket = true
		// no, create a new websocket connection
		wsConnect, _, err := websocket.DefaultDialer.Dial(c.wsURL, nil)
		if err != nil {
			return nil, err
		}
		// Create the WebSocketConnection object
		wsConnection = &WebSocketConnection{
			Cba:         c,                                           // the CoinbaseAdvanced object that created this websocket connection
			conn:        wsConnect,                                   // the websocket connection
			channelList: NewChannelMap[chan<- interface{}, string](), // the channels for this websocket connection
			mutex:       new(sync.RWMutex),                           // the mutex for this websocket connection
			productIDs:  productIDs,                                  // the product IDs for this websocket connection
			shutdown:    make(chan struct{}),                         // the shutdown channel
		}
		// Save the WebSocketConnection object
		c.wsList[wsKey] = wsConnection
	}
	// Is this websocket channel already subscribed to?
	channels := wsConnection.channelList.GetKeysForValue(sub.ChannelName)
	if len(channels) == 0 {
		// no, subscribe to the websocket channel
		// create the subscription message
		subMsg := map[string]interface{}{
			"type":        "subscribe",
			"channel":     sub.ChannelName,
			"api_key":     c.apiKey,
			"product_ids": productIDs,
		}
		// sign the subscription message
		subMsg = c.timestampAndSign(subMsg, sub.ChannelName, productIDs)
		// send the subscription message
		err = wsConnection.conn.WriteJSON(subMsg)
		if err != nil {
			return nil, err
		}
		if newWebSocket {
			// subscribe to the heartbeat channel
			// create the subscription message
			subMsg = map[string]interface{}{
				"type":        "subscribe",
				"channel":     "heartbeats",
				"api_key":     c.apiKey,
				"product_ids": productIDs,
			}
			// sign the subscription message
			subMsg = c.timestampAndSign(subMsg, "heartbeats", productIDs)
			// send the subscription message
			err = wsConnection.conn.WriteJSON(subMsg)
			if err != nil {
				return nil, err
			}
			// add the heartbeats channel to the channel list
			// wsConnection.channelList.Add(sub.Output, "heartbeats")
		}
	}
	// add the output channel to the channel's output list
	wsConnection.channelList.Add(sub.Output, sub.ChannelName)

	// start the dispatcher
	go dispatcher(wsConnection)

	return wsConnection, nil
}

// restartWS restarts a websocket connection and resubscribes to the channels.
// Parameters:
// Returns:
//
//	err       - an error object
func (ws *WebSocketConnection) restartWS() error {
	log.Printf("restartWS: Restarting websocket connection: %v", ws.productIDs)
	// close the websocket connection
	if err := ws.conn.Close(); err != nil {
		log.Printf("restartWS: Error closing websocket connection: %v", err)
		return err
	}
	// reopen the websocket connection
	wsConnect, _, err := websocket.DefaultDialer.Dial(ws.Cba.wsURL, nil)
	if err != nil {
		log.Printf("restartWS: Error opening websocket connection: %v", err)
		return err
	}
	// update the websocket connection
	ws.conn = wsConnect
	// subscribe to the heartbeat channel
	// create the subscription message
	subMsg := map[string]interface{}{
		"type":        "subscribe",
		"channel":     "heartbeats",
		"api_key":     ws.Cba.apiKey,
		"product_ids": ws.productIDs,
	}
	// sign the subscription message
	subMsg = ws.Cba.timestampAndSign(subMsg, "heartbeats", ws.productIDs)
	// send the subscription message
	if err := ws.conn.WriteJSON(subMsg); err != nil {
		log.Printf("restartWS: Error sending heartbeat subscription message: %v", err)
		return err
	}
	// resubscribe to the channels
	// need a list of websocket channels to resubscribe to
	wsChannels := ws.channelList.GetValues()
	for _, channel := range wsChannels {
		// create the subscription message
		subMsg := map[string]interface{}{
			"type":        "subscribe",
			"channel":     channel,
			"api_key":     ws.Cba.apiKey,
			"product_ids": ws.productIDs,
		}
		// sign the subscription message
		subMsg = ws.Cba.timestampAndSign(subMsg, channel, ws.productIDs)
		// send the subscription message
		if err := ws.conn.WriteJSON(subMsg); err != nil {
			log.Printf("restartWS: Error sending subscription message: %v", err)
			return err
		}
	}
	return nil
}

// Unsubscribe unsubscribes from a websocket channel for a given set of product IDs.
// If the channel's output list is empty, the websocket channel is unsubscribed from.
// If the websocket connection's channel list is empty, the websocket connection is closed.
// The output channel is NOT closed when the function returns. This is the responsibility of the caller.
// Parameters:
//   ctx          - a context.Context object
//   wsConnection - a pointer to a WebSocketConnection object representing the websocket connection
//                    to unsubscribe from. This is the object returned by the Subscribe() function.
//   sub          - a pointer to a ChannelParams object representing the websocket channel subscription
// Returns:
//   err - an error object

// TODO: Change isEmpty function to a numeric return value indicating the number of subscriptions to the websocket channel.
//
//	This is needed to allow the hidden heartbeat channel to be unsubscribed from.
//	 As an alternative, do not register the heartbeat channel with the dispatcher, and leave
//	 the isEmpty function as is. but add an unregister call to Unsubscribe() to remove the
//	 heartbeat channel from the websocket when the last subscription is removed.
func (c *CoinbaseAdvanced) Unsubscribe(ctx context.Context, wsConnection *WebSocketConnection, sub *ChannelParams) error {
	// Validate parameters
	if wsConnection == nil || sub == nil || sub.ChannelName == "" || sub.Output == nil {
		return errors.New("invalid subscription parameters")
	}

	// Check if the websocket channel is subscribed to
	channelKeys := wsConnection.channelList.GetKeysForValue(sub.ChannelName)
	if len(channelKeys) == 0 {
		return errors.New("websocket channel is not subscribed to")
	}

	// Remove the output channel from the channel's output list
	channel := wsConnection.channelList
	isEmpty := channel.Remove(sub.Output)

	// Unsubscribe from the websocket channel if the output list is empty
	if isEmpty {
		// Create the subscription message
		subMsg := map[string]interface{}{
			"type":    "unsubscribe",
			"channel": sub.ChannelName,
		}
		// Sign the subscription message
		subMsg = c.timestampAndSign(subMsg, sub.ChannelName, wsConnection.productIDs)
		// Send the subscription message
		if err := wsConnection.conn.WriteJSON(subMsg); err != nil {
			return err
		}
		// Close the shutdown channel
		close(wsConnection.shutdown)
		// Delete the websocket from the CoinbaseAdvanced object
		keys := strings.Join(wsConnection.productIDs, ",")
		delete(c.wsList, keys)
	}
	return nil
}

// ChannelMap is a generic map-like structure with duplicate keys.
// It associates multiple values with each key and allows concurrent access.
type ChannelMap[K comparable, V comparable] struct {
	normalMap  map[K][]V
	reverseMap map[V][]K
	mutex      sync.Mutex
}

// NewChannelMap creates a new instance of ChannelMap with the specified data types for keys and values.
func NewChannelMap[K comparable, V comparable]() *ChannelMap[K, V] {
	return &ChannelMap[K, V]{
		normalMap:  make(map[K][]V),
		reverseMap: make(map[V][]K),
	}
}

func (cm *ChannelMap[K, V]) Add(key K, values ...V) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.normalMap[key] = append(cm.normalMap[key], values...)
	for _, val := range values {
		cm.reverseMap[val] = append(cm.reverseMap[val], key)
	}
}

// Remove removes the specified values from the slice associated with the specified key.
// If the slice is empty after removing the values, the key is removed from the map.
// If the map is empty after removing the key, the isEmpty return value is true.

func (cm *ChannelMap[K, V]) Remove(key K, values ...V) (isEmpty bool) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	isEmpty = false

	if vals, exists := cm.normalMap[key]; exists {
		for _, val := range values {
			for i, v := range vals {
				if val == v {
					// Remove the value from the slice
					cm.normalMap[key] = append(vals[:i], vals[i+1:]...)
					break
				}
			}
			// Remove the key from the reverse map
			if keys, found := cm.reverseMap[val]; found {
				for i, k := range keys {
					if k == key {
						cm.reverseMap[val] = append(keys[:i], keys[i+1:]...)
						break
					}
				}
				if len(cm.reverseMap[val]) == 0 {
					delete(cm.reverseMap, val)                  // Delete the reverseMap entry if empty
					isEmpty = isEmpty || len(cm.normalMap) == 0 // Check if the ChannelMap is empty
				}
			}
		}

		// Remove the key from the normal map if the slice is empty
		if len(cm.normalMap[key]) == 0 {
			delete(cm.normalMap, key)
			isEmpty = isEmpty || len(cm.normalMap) == 0 // Check if the ChannelMap is empty
		}
	}

	return isEmpty
}

// GetKeysForValue returns the keys associated with the specified value.
// If the value is not found, nil is returned.
func (cm *ChannelMap[K, V]) GetKeysForValue(value V) []K {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if keys, found := cm.reverseMap[value]; found {
		return keys
	}

	return nil
}

// GetValuesForKey returns the values associated with the specified key.
// If the key is not found, nil is returned.
func (cm *ChannelMap[K, V]) GetValuesForKey(key K) []V {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if values, exists := cm.normalMap[key]; exists {
		return values
	}

	return nil
}

// IsEmpty checks if both the normal map and the reverse map are empty.
func (cm *ChannelMap[K, V]) IsEmpty() bool {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	return len(cm.normalMap) == 0 && len(cm.reverseMap) == 0
}

// GetKeys returns a slice of keys in the ChannelMap.
// The keys are unique.
func (cm *ChannelMap[K, V]) GetKeys() []K {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	keys := make([]K, 0, len(cm.normalMap))
	seen := make(map[K]bool)

	for key := range cm.normalMap {
		if !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}

	return keys
}

func (cm *ChannelMap[K, V]) GetValues() []V {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	values := make([]V, 0, len(cm.reverseMap))
	seen := make(map[V]bool)

	for value := range cm.reverseMap {
		if !seen[value] {
			values = append(values, value)
			seen[value] = true
		}
	}

	return values
}
