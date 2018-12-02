package ipstack

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	// DefaultClientTimeout is the recommended default for http timeouts, when
	// the external ipstack API is called
	DefaultClientTimeout = 5
)

// Client defines a single http client, which can be used to acces the external
// ipstack API
type Client struct {
	httpClient   http.Client
	httpsEnabled bool
	accessKey    string
}

// NewClient initializes a new ipstack.Client which can access the external
// ipstack API in a typesafe way
func NewClient(accessKey string, httpsEnabled bool, timeout int) *Client {
	// calculate duration for timeout
	duration := time.Duration(timeout) * time.Second

	// create http client for accessing the external api
	netTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: duration,
		}).Dial,
		TLSHandshakeTimeout: duration,
	}
	httpClient := http.Client{
		Timeout:   duration,
		Transport: netTransport,
	}

	// return new ipstack instance
	return &Client{
		httpClient:   httpClient,
		httpsEnabled: httpsEnabled,
		accessKey:    accessKey,
	}
}

// Check performs a single API call to the external ipstack API and returns
// the response object or any occurred errors
func (c *Client) Check(ip string) (r *Response, err error) {
	// Unfortunately ipstack only offers unencrypted http in it's free tier.
	// Therefore we limit the protocol to http by default
	protocol := "http://"
	if c.httpsEnabled {
		protocol = "https://"
	}

	// build url that's used to call the api endpoint
	url := fmt.Sprintf("%sapi.ipstack.com/%s?access_key=%s&hostname=1&language=en&output=json", protocol, ip, c.accessKey)

	// query external api
	buf, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	// free allocated resources
	defer buf.Body.Close()

	// unmarshal json response
	r = &Response{}
	err = json.NewDecoder(buf.Body).Decode(r)
	if err != nil {
		return nil, err
	}

	return r, nil
}
