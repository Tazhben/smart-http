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

func defaultClient() Client {
	client := Client{}

	client.http.Transport = defaultTransport

	// по дефолту во всех сервисах используем openTelemetry
	client.http.Transport = otelhttp.NewTransport(client.http.Transport)
	return client
}

type TLSConfig struct {
	IsSecure bool
	CaCert   *x509.CertPool
}

type Client struct {
	tlsConfig TLSConfig

	http http.Client
}

// NewClient создание клиента
func NewClient() http.Client {
	client := defaultClient()

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
