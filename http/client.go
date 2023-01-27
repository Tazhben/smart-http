package http

import (
	"crypto/x509"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var defaultTransport = &http.Transport{
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var defaultTimeout = time.Millisecond * 300

var defaultRetry = &RetryRoundTripper{
	Retries:    2,
	RetryDelay: time.Millisecond * 100,
	StatusCodes: map[int]bool{
		http.StatusInternalServerError: true,
		http.StatusBadGateway:          true,
		http.StatusServiceUnavailable:  true,
		http.StatusGatewayTimeout:      true,
	},
}

func defaultClient() *Client {
	client := Client{}

	client.http.Transport = defaultTransport
	client.timeout = defaultTimeout

	// по дефолту во всех сервисах используем openTelemetry
	client.http.Transport = otelhttp.NewTransport(client.http.Transport)

	// по дефолту 2 попытки
	client.http = NewRetryClient(client.http, defaultRetry)

	return &client
}

type TLSConfig struct {
	IsSecure bool
	CaCert   *x509.CertPool
}

type RetryRoundTripper struct {
	Transport http.RoundTripper

	Retries     int
	RetryDelay  time.Duration
	StatusCodes map[int]bool
}

func NewRetryRoundTripper(retries int, retryDelay time.Duration, statusCodes map[int]bool) *RetryRoundTripper {
	return &RetryRoundTripper{
		Retries:     retries,
		RetryDelay:  retryDelay,
		StatusCodes: statusCodes,
	}
}

type Client struct {
	tlsConfig TLSConfig
	timeout   time.Duration

	http *http.Client
}

func NewRetryClient(inner *http.Client, retry *RetryRoundTripper) *http.Client {

	retry.Transport = inner.Transport

	return &http.Client{
		Transport: retry,
		Timeout:   inner.Timeout,
	}
}

// NewClient создание клиента
func NewClient(options ...func(*Client)) *http.Client {
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

// WithRetry настройка попытки повторного запроса
func WithRetry(retries int, retryDelay time.Duration, statusCodes []int) func(client *Client) {
	return func(client *Client) {

		statusCodesInMap := make(map[int]bool)
		for _, status := range statusCodes {
			if _, ok := statusCodesInMap[status]; ok {
				continue
			}
			statusCodesInMap[status] = true
		}

		retry := NewRetryRoundTripper(retries, retryDelay, statusCodesInMap)
		client.http = NewRetryClient(client.http, retry)
	}
}

// RoundTrip написал обвертку на request Response для retry
func (r *RetryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < r.Retries; i++ {

		resp, err = r.Transport.RoundTrip(req)
		if err != nil {
			continue
		}

		// проверка на статусы
		if _, ok := r.StatusCodes[resp.StatusCode]; !ok {
			return resp, err
		}

		time.Sleep(r.RetryDelay)
	}
	return resp, err
}
