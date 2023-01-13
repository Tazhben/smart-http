package http

import (
	"context"
	"crypto/x509"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}

var defaultTransport = &http.Transport{
	DialContext: defaultTransportDialContext(&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}),
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var defaultTimeout = time.Millisecond * 300

func defaultClient() *Client {
	client := Client{}

	client.http.Transport = defaultTransport
	client.timeout = defaultTimeout

	// по дефолту во всех сервисах используем openTelemetry
	client.http.Transport = otelhttp.NewTransport(client.http.Transport)
	return &client
}

type TLSConfig struct {
	IsSecure bool
	CaCert   *x509.CertPool
}

type Client struct {
	tlsConfig TLSConfig
	timeout   time.Duration

	http http.Client
}

// NewClient создание клиента
func NewClient(options ...func(*Client)) http.Client {
	client := defaultClient()

	for _, o := range options {
		o(client)
	}

	return client.http
}

// WithCertificate для отправки tls запросов
func WithCertificate(caCert []byte) func(client *Client) {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := TLSConfig{
		CaCert:   caCertPool,
		IsSecure: true,
	}

	return func(client *Client) {
		client.tlsConfig = tlsConfig
	}
}

// WithTimeout настройка timeout всех запросов
func WithTimeout(timeout time.Duration) func(client *Client) {
	return func(client *Client) {
		client.http.Timeout = timeout
	}
}
