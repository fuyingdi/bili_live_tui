package api

import (
	"net/http"
)

// NewHTTPClient 创建一个带有默认配置的HTTP客户端
func NewHTTPClient() *http.Client {
	client := &http.Client{}
	client.Transport = &userAgentTransport{
		base: http.DefaultTransport,
	}
	return client
}

type userAgentTransport struct {
	base http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", DefaultUserAgent)
	return t.base.RoundTrip(req)
}
