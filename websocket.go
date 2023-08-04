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
	conn        *websocket.Conn                    // The websocket connection
	channelList *ChannelMap[chan<- string, string] // The key is the websocket channel that are subscribed to ie. "ticker"
	mutex       *sync.RWMutex                      // Mutex for synchronizing access to subscriptions
	productIDs  []string                           // The product IDs for this websocket connection
	shutdown    chan struct{}                      // The channel to close the dispatcher goroutine
}

// ChannelParams represents the parameters for a websocket channel subscription.
type ChannelParams struct {
	ChannelName string        // the name of the websocket channel
	Output      chan<- string // the output go channel for this websocket channel
}

// Channel represents a channel subscription for an application routine.
type Channel struct {
	ChannelName string                          // the name of the websocket channel
	mux         *sync.RWMutex                   // the mutex for this websocket channel
	Outputs     map[chan<- string]chan<- string // the output go channels for this websocket channel
}

// TODO: Rewrite the dispatcher to allow multiple subscriptions to the same go channel.
//
//	This will require a change to the Channel struct to allow multiple output channels.
//	Also allow certain channels to be filtered from the output channel. (ie. heartbeat)
//
// Dispatcher reads messages from the websocket connection and sends them to the appropriate output channels.
// The function runs in a separate goroutine and returns when the shutdown channel is closed.
// The output channels are NOT closed when the function returns.
// The messages are filtered by websocket channel and sent to the appropriate output channels.
// Multiple dispatchers may not be registered to the same output channel. (not enforced, caller beware)
// Parameters:
//
//	ws - a pointer to a WebSocketConnection object representing the websocket connection
func Dispatcher(ws *WebSocketConnection) {
	for {
		select {
		case <-ws.shutdown:
			return

		default:
			// read the message
			_, message, err := ws.conn.ReadMessage()
			if err != nil {
				// log the error instead of returning it
				log.Printf("Error reading message: %v", err)
				continue
			}
			// get the channel name from the message
			var channelName string
			if err := json.Unmarshal(message, &channelName); err != nil {
				// log the error instead of returning it
				log.Printf("Error unmarshalling message: %v", err)
				continue
			}
			log.Printf("Channel: %v", channelName)
			// get the output channels for the websocket channel
			outputs := ws.channelList.GetKeysForValue(channelName)
			// send the message to the output channels
			for _, output := range outputs {
				output <- string(message)
			}
			// log.Printf("Message: %v", string(message))
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
	var wsKey string = ""
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
		// no, create a new websocket connection
		wsConnect, _, err := websocket.DefaultDialer.Dial(c.wsURL, nil)
		if err != nil {
			return nil, err
		}
		// Create the WebSocketConnection object
		wsConnection = &WebSocketConnection{
			conn:        wsConnect,                              // the websocket connection
			channelList: NewChannelMap[chan<- string, string](), // the channels for this websocket connection
			mutex:       new(sync.RWMutex),                      // the mutex for this websocket connection
			productIDs:  productIDs,                             // the product IDs for this websocket connection
			shutdown:    make(chan struct{}),                    // the shutdown channel
		}
		// Save the WebSocketConnection object
		c.wsList[wsKey] = wsConnection
	}
	// Is this websocket channel already subscribed to?
	channels := wsConnection.channelList.GetKeysForValue(sub.ChannelName)
	if len(channels) == 0 {
		// _, ok = wsConnection.channelList[sub.ChannelName]
		// if !ok {
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
		// subscribe to the heartbeat channel
		// create the subscription message
		subMsg = map[string]interface{}{
			"type":        "subscribe",
			"channel":     "heartbeat",
			"api_key":     c.apiKey,
			"product_ids": productIDs,
		}
		// sign the subscription message
		subMsg = c.timestampAndSign(subMsg, "heartbeat", productIDs)
		// send the subscription message
		err = wsConnection.conn.WriteJSON(subMsg)
		if err != nil {
			return nil, err
		}

		// create the channel subscription object
		// wsConnection.mutex.Lock() // lock the websocket connection for writing
		// wsConnection.channelList[sub.ChannelName] = &Channel{
		// 	ChannelName: sub.ChannelName,                       // the name of the websocket channel
		// 	mux:         new(sync.RWMutex),                     // the mutex for this websocket channel
		// 	Outputs:     make(map[chan<- string]chan<- string), // the output go channels for this websocket channel
		// }
		// wsConnection.mutex.Unlock() // unlock the websocket connection
	}
	// add the output channel to the channel's output list
	wsConnection.channelList.Add(sub.Output, sub.ChannelName)
	// wsConnection.channelList[sub.ChannelName].mux.Lock()                       // lock the channel for writing
	// wsConnection.channelList[sub.ChannelName].Outputs[sub.Output] = sub.Output // add the output channel to the channel's output list
	// wsConnection.channelList[sub.ChannelName].mux.Unlock()                     // unlock the channel

	// start the dispatcher
	go Dispatcher(wsConnection)

	return wsConnection, nil
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
		// delete(wsConnection.channelList, channelKeys[0])
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

func (cm *ChannelMap[K, V]) GetKeysForValue(value V) []K {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if keys, found := cm.reverseMap[value]; found {
		return keys
	}

	return nil
}

func (cm *ChannelMap[K, V]) GetValuesForKey(key K) []V {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if values, exists := cm.normalMap[key]; exists {
		return values
	}

	return nil
}
