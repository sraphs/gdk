// Package useragent includes constants and utilitiesfor setting the User-Agent
// for Go CDK connections to GCP.
package useragent

import (
	"fmt"
	"net/http"

	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

const (
	prefix  = "go-cloud"
	version = "0.1.0"
)

// ClientOption returns an option.ClientOption that sets a Go CDK User-Agent.
func ClientOption(api string) option.ClientOption {
	return option.WithUserAgent(userAgentString(api))
}

// GRPCDialOption returns a grpc.DialOption that sets a Go CDK User-Agent.
func GRPCDialOption(api string) grpc.DialOption {
	return grpc.WithUserAgent(userAgentString(api))
}

// AzureUserAgentPrefix returns a prefix that is used to set Azure SDK User-Agent to help with diagnostics.
func AzureUserAgentPrefix(api string) string {
	return userAgentString(api)
}

func userAgentString(api string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, api, version)
}

// userAgentTransport wraps an http.RoundTripper, adding a User-Agent header
// to each request.
type userAgentTransport struct {
	base http.RoundTripper
	api  string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid mutating it.
	newReq := *req
	newReq.Header = make(http.Header)
	for k, vv := range req.Header {
		newReq.Header[k] = vv
	}
	// Append to the User-Agent string to preserve other information.
	newReq.Header.Set("User-Agent", req.UserAgent()+" "+userAgentString(t.api))
	return t.base.RoundTrip(&newReq)
}

// HTTPClient wraps client and appends a Go CDK string to the User-Agent
// header for all requests.
func HTTPClient(client *http.Client, api string) *http.Client {
	c := *client
	c.Transport = &userAgentTransport{base: c.Transport, api: api}
	return &c
}
