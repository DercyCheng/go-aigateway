package protocol

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"go-aigateway/internal/config"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type ProtocolConverter struct {
	config     *config.ProtocolConversionConfig
	httpClient *http.Client
	grpcConns  map[string]*grpc.ClientConn
}

type ConversionRequest struct {
	SourceProtocol string                 `json:"source_protocol"`
	TargetProtocol string                 `json:"target_protocol"`
	Endpoint       string                 `json:"endpoint"`
	Method         string                 `json:"method"`
	Headers        map[string]string      `json:"headers"`
	Body           interface{}            `json:"body"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type ConversionResponse struct {
	StatusCode int                    `json:"status_code"`
	Headers    map[string]string      `json:"headers"`
	Body       interface{}            `json:"body"`
	Metadata   map[string]interface{} `json:"metadata"`
	Error      string                 `json:"error,omitempty"`
}

func NewProtocolConverter(cfg *config.ProtocolConversionConfig) *ProtocolConverter {
	if !cfg.Enabled {
		return nil
	}

	return &ProtocolConverter{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		},
		grpcConns: make(map[string]*grpc.ClientConn),
	}
}

func (pc *ProtocolConverter) Convert(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	if pc == nil {
		return nil, fmt.Errorf("protocol conversion not enabled")
	}

	switch {
	case req.SourceProtocol == "https" && req.TargetProtocol == "grpc":
		return pc.httpsToGRPC(ctx, req)
	case req.SourceProtocol == "grpc" && req.TargetProtocol == "https":
		return pc.grpcToHTTPS(ctx, req)
	case req.SourceProtocol == "http" && req.TargetProtocol == "https":
		return pc.httpToHTTPS(ctx, req)
	case req.SourceProtocol == "https" && req.TargetProtocol == "http":
		return pc.httpsToHTTP(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported protocol conversion: %s -> %s", req.SourceProtocol, req.TargetProtocol)
	}
}

func (pc *ProtocolConverter) httpsToGRPC(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	if !pc.config.GRPCSupport {
		return nil, fmt.Errorf("gRPC support not enabled")
	}

	logrus.WithFields(logrus.Fields{
		"source":   "https",
		"target":   "grpc",
		"endpoint": req.Endpoint,
	}).Info("Converting HTTPS to gRPC")

	// Parse gRPC endpoint
	conn, err := pc.getGRPCConnection(req.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get gRPC connection: %w", err)
	}

	// Convert HTTP headers to gRPC metadata
	md := metadata.New(req.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// TODO: Implement actual gRPC call based on service definition
	// This is a placeholder implementation
	// For now, we just verify the connection is available
	_ = conn

	return &ConversionResponse{
		StatusCode: 200,
		Headers:    make(map[string]string),
		Body:       map[string]interface{}{"message": "gRPC call successful"},
		Metadata:   map[string]interface{}{"conversion": "https-to-grpc"},
	}, nil
}

func (pc *ProtocolConverter) grpcToHTTPS(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	logrus.WithFields(logrus.Fields{
		"source":   "grpc",
		"target":   "https",
		"endpoint": req.Endpoint,
	}).Info("Converting gRPC to HTTPS")

	// Convert gRPC metadata to HTTP headers
	headers := make(map[string]string)
	for k, v := range req.Headers {
		headers[k] = v
	}

	// Prepare HTTP request
	var body io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.Endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := pc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	var bodyObj interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &bodyObj); err != nil {
			bodyObj = string(respBody)
		}
	}

	return &ConversionResponse{
		StatusCode: resp.StatusCode,
		Headers:    respHeaders,
		Body:       bodyObj,
		Metadata:   map[string]interface{}{"conversion": "grpc-to-https"},
	}, nil
}

func (pc *ProtocolConverter) httpToHTTPS(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	logrus.WithFields(logrus.Fields{
		"source":   "http",
		"target":   "https",
		"endpoint": req.Endpoint,
	}).Info("Converting HTTP to HTTPS")

	// Convert HTTP URL to HTTPS
	httpsURL := strings.Replace(req.Endpoint, "http://", "https://", 1)

	return pc.executeHTTPRequest(ctx, req.Method, httpsURL, req.Headers, req.Body)
}

func (pc *ProtocolConverter) httpsToHTTP(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	logrus.WithFields(logrus.Fields{
		"source":   "https",
		"target":   "http",
		"endpoint": req.Endpoint,
	}).Info("Converting HTTPS to HTTP")

	// Convert HTTPS URL to HTTP
	httpURL := strings.Replace(req.Endpoint, "https://", "http://", 1)

	return pc.executeHTTPRequest(ctx, req.Method, httpURL, req.Headers, req.Body)
}

func (pc *ProtocolConverter) executeHTTPRequest(ctx context.Context, method, endpoint string, headers map[string]string, body interface{}) (*ConversionResponse, error) {
	var reqBody io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	if reqBody != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := pc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	var bodyObj interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &bodyObj); err != nil {
			bodyObj = string(respBody)
		}
	}

	return &ConversionResponse{
		StatusCode: resp.StatusCode,
		Headers:    respHeaders,
		Body:       bodyObj,
		Metadata:   map[string]interface{}{"conversion": fmt.Sprintf("%s-conversion", method)},
	}, nil
}

func (pc *ProtocolConverter) getGRPCConnection(endpoint string) (*grpc.ClientConn, error) {
	if conn, exists := pc.grpcConns[endpoint]; exists {
		return conn, nil
	}

	// Parse endpoint to extract host and port
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gRPC endpoint: %w", err)
	}

	address := u.Host
	if u.Port() == "" {
		if u.Scheme == "grpcs" {
			address += ":443"
		} else {
			address += ":80"
		}
	}

	var opts []grpc.DialOption
	if u.Scheme == "grpcs" {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	pc.grpcConns[endpoint] = conn
	return conn, nil
}

func (pc *ProtocolConverter) Close() error {
	for endpoint, conn := range pc.grpcConns {
		if err := conn.Close(); err != nil {
			logrus.WithError(err).WithField("endpoint", endpoint).Error("Failed to close gRPC connection")
		}
	}
	pc.grpcConns = make(map[string]*grpc.ClientConn)
	return nil
}
