package coinbase_adv_go

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// sign generates a signature for a given string using the HMAC-SHA256 algorithm and a secret key.
//
// Parameters:
//
//	str    - the string to sign
//	secret - the secret key to use for signing
//
// Returns:
//
//	a string representing the signature of the input string
func (c *CoinbaseAdvanced) sign(str, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(str))
	signature := hex.EncodeToString(h.Sum(nil))
	return signature
}

// timestampAndSign generates a signature for a given message using the HMAC-SHA256 algorithm and a secret key.
// The function adds a timestamp and signature to the message and returns the updated message.
//
// Parameters:
//
//	message  - a map[string]interface{} representing the message to sign
//	channel  - a string representing the channel for the message
//	products - a slice of strings representing the products for the message
//
// Returns:
//
//	a map[string]interface{} representing the updated message with the timestamp and signature added
func (c *CoinbaseAdvanced) timestampAndSign(message map[string]interface{}, channel string, products []string) map[string]interface{} {
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	strToSign := timestamp + channel + strings.Join(products, ",")
	sig := c.sign(strToSign, c.apiSecret)
	message["signature"] = sig
	message["timestamp"] = timestamp
	return message
}

func (c *CoinbaseAdvanced) signRequest(req *http.Request) {
	timestamp := time.Now().UTC().Unix()
	message := fmt.Sprintf("%d%s%s", timestamp, req.Method, req.URL.Path)
	signature := c.sign(message, c.apiSecret)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	req.Header.Set("CB-ACCESS-KEY", c.apiKey)
}

// readAndWrite reads messages from a websocket connection and writes them to an output channel.
// If no messages are received for a specified timeout period, the function will shut down the stream.
// The function runs in a separate goroutine and returns when the shutdown channel is closed.
// The output channel is closed when the function returns.
//
// Parameters:
//
//	c        - a pointer to a websocket.Conn object representing the websocket connection
//	output   - a channel of strings to which the received messages will be written
//	shutdown - a channel of struct{} used to signal the function to shut down
//	wg       - a pointer to a sync.WaitGroup object used to synchronize the function with other goroutines
//
// Returns: none
func readAndWrite(c *websocket.Conn, output chan string, shutdown chan struct{}, wg *sync.WaitGroup) {
	// Mark the wait group as done when the function returns
	defer wg.Done()

	var (
		mutex   sync.Mutex // Mutex for synchronizing access to output channel
		timeout = 15       // Time in seconds for stream timeout
	)

	// Create a timer with the specified timeout
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop() // Stop the timer when the function returns

	for {
		select {
		case <-shutdown:
			timer.Stop()   // Stop the timer
			mutex.Unlock() // Release the mutex
			close(output)  // Close the output channel
			return
		case <-timer.C:
			fmt.Println("Timeout: shutting down stream")
			close(output) // Close the output channel
			return
		default:
			// Reset the timer if a message is received
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(time.Duration(timeout) * time.Second)

			// Read the message
			_, message, err := c.ReadMessage()
			if err != nil {
				output <- fmt.Sprintf("read: %v", err)
				return
			}
			s := string(message)
			mutex.Lock()   // Lock the mutex to synchronize access to the output channel
			output <- s    // Write the message to the output channel
			mutex.Unlock() // Unlock the mutex
		}
	}
}

//	func (c *CoinbaseAdapter) signRequest(req *http.Request, timestamp string) {
//		message := timestamp + req.Method + req.URL.Path
//		mac := hmac.New(sha256.New, []byte(c.apiSecret))
//		mac.Write([]byte(message))
//		signature := hex.EncodeToString(mac.Sum(nil))
//		req.Header.Set("CB-ACCESS-SIGN", signature)
//		req.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)
//		req.Header.Set("CB-ACCESS-KEY", c.apiKey)
//	}

func (c *CoinbaseAdvanced) doRequest(ctx context.Context, method, requestPath string, query url.Values, body *strings.Reader) (*http.Response, error) {
	URL, _ := url.Parse(c.apiBaseURL + c.apiEndpoint + requestPath)

	req, err := http.NewRequestWithContext(ctx, method, URL.String(), body)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = query.Encode()

	c.signRequest(req)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	return c.client.Do(req.WithContext(ctx))
}

// Parses the query parameters from the params struct and adds them to the query_params url.Values
// object. The params struct must contain either values or pointers to the values that should be
// added to the query params. The params struct must also contain a "param" tag that specifies the
// name of the query parameter. The tag can also contain a "required" fragment to indicate that the
// parameter is required. If the parameter is required and the value is nil, the function will panic.
// The tag can also contain a "default" fragment to specify a default value for the parameter. If the
// parameter is required and the value is nil, the function will use the default value instead of
// panicking.

func (c *CoinbaseAdvanced) addParamsToQuery(params interface{}, query_params url.Values) error {
	if params != nil && !reflect.ValueOf(params).IsNil() {
		// Get the reflect.Value of the *params struct
		v := reflect.ValueOf(params).Elem()

		// Loop over the fields of the *params struct
		for i := 0; i < v.NumField(); i++ {
			// Get the field value and its corresponding params tag
			var paramName, requiredTag, defaultValue string
			field := v.Field(i)
			// Check if the field is a pointer

			if field.Type().Kind() == reflect.Ptr {
				field = field.Elem()
			}

			tagtype := v.Type().Field(i).Tag.Get("param")
			tags := strings.Split(tagtype, ",")
			l := len(tags)
			if l > 0 {
				paramName = tags[0]
			}
			if len(tags) > 1 {
				requiredTag = tags[1]
			}
			if len(tags) > 2 {
				defaultValue = tags[2]
			}

			// Check if the field value is nil
			fieldName := v.Type().Field(i).Name
			empty, err := isFieldEmpty(params, fieldName)
			if err != nil {
				return err
			}
			if empty {
				if requiredTag == "required" {
					if defaultValue != "" {
						query_params.Add(paramName, defaultValue)
					} else {
						return fmt.Errorf("required field '%s' is nil", fieldName)
					}
				}
				continue
			}

			// Add the field value to the query params
			if requiredTag == "-" {
				continue
			}
			switch field.Type().Kind() {
			case reflect.String:
				query_params.Add(paramName, field.String())
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				query_params.Add(paramName, strconv.FormatInt(field.Int(), 10))
			case reflect.Struct:
				query_params.Add(paramName, field.Interface().(time.Time).Format(time.RFC3339))
			}
		}
	}
	return nil
}

// isFieldEmpty checks whether a field in a struct is empty or not.
//
// Parameters:
//
//	s         - the struct to check
//	fieldname - the name of the field to check
//
// Returns:
//
//	a boolean indicating whether the field is empty or not
//	an error if the field does not exist in the struct
func isFieldEmpty(s interface{}, fieldname string) (bool, error) {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(fieldname)
	if !field.IsValid() {
		return false, fmt.Errorf("no such field: %s in obj", fieldname)
	}
	return reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()), nil
}
