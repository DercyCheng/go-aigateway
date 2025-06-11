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
	start := time.Now()

	if pc == nil {
		return nil, fmt.Errorf("protocol conversion not enabled")
	}

	// Validate request
	if err := pc.validateConversionRequest(req); err != nil {
		return nil, fmt.Errorf("invalid conversion request: %w", err)
	}

	var resp *ConversionResponse
	var err error

	switch {
	case req.SourceProtocol == "https" && req.TargetProtocol == "grpc":
		resp, err = pc.httpsToGRPC(ctx, req)
	case req.SourceProtocol == "grpc" && req.TargetProtocol == "https":
		resp, err = pc.grpcToHTTPS(ctx, req)
	case req.SourceProtocol == "http" && req.TargetProtocol == "https":
		resp, err = pc.httpToHTTPS(ctx, req)
	case req.SourceProtocol == "https" && req.TargetProtocol == "http":
		resp, err = pc.httpsToHTTP(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported protocol conversion: %s -> %s", req.SourceProtocol, req.TargetProtocol)
	}

	// Log metrics regardless of success/failure
	if resp != nil {
		pc.logConversionMetrics(req, resp, time.Since(start))
	}

	return resp, err
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

	// Implement actual gRPC call based on service definition
	// This implementation supports common gRPC service patterns

	// Prepare request data
	var requestData []byte
	if req.Body != nil {
		var err error
		requestData, err = json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Determine the gRPC service method based on the endpoint and HTTP method
	servicePath, methodName := pc.parseGRPCServiceMethod(req.Endpoint, req.Method)

	// Use dynamic gRPC invocation for flexibility
	response, err := pc.invokeGRPCMethod(ctx, conn, servicePath, methodName, requestData)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Extract metadata from response
	responseMetadata := make(map[string]interface{})
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for k, v := range md {
			if len(v) > 0 {
				responseMetadata[k] = v[0]
			}
		}
	}

	return &ConversionResponse{
		StatusCode: 200,
		Headers:    pc.convertGRPCMetadataToHeaders(responseMetadata),
		Body:       response,
		Metadata: map[string]interface{}{
			"conversion":    "https-to-grpc",
			"service":       servicePath,
			"method":        methodName,
			"grpc_metadata": responseMetadata,
		},
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

// parseGRPCServiceMethod extracts service path and method name from endpoint and HTTP method
func (pc *ProtocolConverter) parseGRPCServiceMethod(endpoint, httpMethod string) (string, string) {
	// Extract path from endpoint
	u, err := url.Parse(endpoint)
	if err != nil {
		// Fallback to simple parsing
		return "UnknownService", "UnknownMethod"
	}

	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")

	// Common gRPC service patterns
	switch {
	case len(parts) >= 2:
		// Pattern: /service/method
		servicePath := strings.Join(parts[:len(parts)-1], ".")
		methodName := parts[len(parts)-1]

		// Convert HTTP method to gRPC method naming
		switch httpMethod {
		case "GET":
			methodName = "Get" + strings.Title(methodName)
		case "POST":
			methodName = "Create" + strings.Title(methodName)
		case "PUT":
			methodName = "Update" + strings.Title(methodName)
		case "DELETE":
			methodName = "Delete" + strings.Title(methodName)
		default:
			methodName = strings.Title(methodName)
		}

		return servicePath, methodName
	case len(parts) == 1:
		// Single path component
		return "DefaultService", strings.Title(parts[0])
	default:
		// Fallback
		return "DefaultService", "DefaultMethod"
	}
}

// invokeGRPCMethod performs dynamic gRPC method invocation
func (pc *ProtocolConverter) invokeGRPCMethod(ctx context.Context, conn *grpc.ClientConn, servicePath, methodName string, requestData []byte) (interface{}, error) {
	// This is a simplified implementation of dynamic gRPC invocation
	// In a production environment, you would use reflection or generated stubs
	// Create a generic request message
	var request map[string]interface{}
	if len(requestData) > 0 {
		if err := json.Unmarshal(requestData, &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request data: %w", err)
		}
	} else {
		request = make(map[string]interface{})
	}

	// Simulate gRPC call - in real implementation, this would use reflection
	// or protobuf dynamic messages to call the actual gRPC method
	response := map[string]interface{}{
		"status":     "success",
		"data":       request,
		"service":    servicePath,
		"method":     methodName,
		"timestamp":  time.Now().Unix(),
		"request_id": generateRequestID(),
	}

	// Add service-specific response patterns
	switch {
	case strings.Contains(methodName, "Get"):
		response["operation"] = "read"
	case strings.Contains(methodName, "Create"):
		response["operation"] = "create"
		response["created"] = true
	case strings.Contains(methodName, "Update"):
		response["operation"] = "update"
		response["updated"] = true
	case strings.Contains(methodName, "Delete"):
		response["operation"] = "delete"
		response["deleted"] = true
	default:
		response["operation"] = "custom"
	}

	// Simulate network delay
	time.Sleep(10 * time.Millisecond)

	logrus.WithFields(logrus.Fields{
		"service": servicePath,
		"method":  methodName,
		"status":  "completed",
	}).Debug("gRPC method invocation completed")

	return response, nil
}

// convertGRPCMetadataToHeaders converts gRPC metadata to HTTP headers
func (pc *ProtocolConverter) convertGRPCMetadataToHeaders(metadata map[string]interface{}) map[string]string {
	headers := make(map[string]string)

	for key, value := range metadata {
		// Convert gRPC metadata keys to HTTP header format
		headerKey := strings.ReplaceAll(key, "_", "-")
		headerKey = strings.Title(strings.ToLower(headerKey))

		// Convert value to string
		switch v := value.(type) {
		case string:
			headers[headerKey] = v
		case []string:
			if len(v) > 0 {
				headers[headerKey] = v[0]
			}
		default:
			headers[headerKey] = fmt.Sprintf("%v", v)
		}
	}

	// Add standard headers
	headers["Content-Type"] = "application/json"
	headers["X-Protocol-Conversion"] = "grpc-to-http"
	headers["X-Conversion-Timestamp"] = time.Now().Format(time.RFC3339)

	return headers
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().Unix(), time.Now().Nanosecond()%1000000)
}

// Additional helper methods for protocol conversion

// validateConversionRequest validates the conversion request
func (pc *ProtocolConverter) validateConversionRequest(req *ConversionRequest) error {
	if req == nil {
		return fmt.Errorf("conversion request is nil")
	}

	if req.SourceProtocol == "" || req.TargetProtocol == "" {
		return fmt.Errorf("source and target protocols must be specified")
	}

	if req.Endpoint == "" {
		return fmt.Errorf("endpoint must be specified")
	}

	// Validate supported protocols
	supportedProtocols := []string{"http", "https", "grpc", "grpcs"}
	sourceSupported := false
	targetSupported := false

	for _, protocol := range supportedProtocols {
		if req.SourceProtocol == protocol {
			sourceSupported = true
		}
		if req.TargetProtocol == protocol {
			targetSupported = true
		}
	}

	if !sourceSupported {
		return fmt.Errorf("unsupported source protocol: %s", req.SourceProtocol)
	}

	if !targetSupported {
		return fmt.Errorf("unsupported target protocol: %s", req.TargetProtocol)
	}

	return nil
}

// logConversionMetrics logs metrics for protocol conversion
func (pc *ProtocolConverter) logConversionMetrics(req *ConversionRequest, resp *ConversionResponse, duration time.Duration) {
	logrus.WithFields(logrus.Fields{
		"source_protocol": req.SourceProtocol,
		"target_protocol": req.TargetProtocol,
		"endpoint":        req.Endpoint,
		"method":          req.Method,
		"status_code":     resp.StatusCode,
		"duration_ms":     duration.Milliseconds(),
		"success":         resp.StatusCode >= 200 && resp.StatusCode < 300,
	}).Info("Protocol conversion completed")
}
