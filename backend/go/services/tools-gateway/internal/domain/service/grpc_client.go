// Package service contains domain services.
package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// GRPCClient handles gRPC API requests.
type GRPCClient interface {
	// Call invokes a gRPC method.
	Call(
		ctx context.Context,
		integration *entity.Integration,
		method string,
		request interface{},
		response interface{},
		token string,
	) error

	// CallWithMetadata invokes a gRPC method with custom metadata.
	CallWithMetadata(
		ctx context.Context,
		integration *entity.Integration,
		method string,
		request interface{},
		response interface{},
		md map[string]string,
	) error

	// GetConnection gets a gRPC connection for the integration.
	GetConnection(ctx context.Context, integration *entity.Integration) (*grpc.ClientConn, error)
}

// grpcClientImpl implements GRPCClient.
type grpcClientImpl struct {
	connections map[string]*grpc.ClientConn // Cache connections by integration ID
}

// NewGRPCClient creates a new GRPCClient.
func NewGRPCClient() GRPCClient {
	return &grpcClientImpl{
		connections: make(map[string]*grpc.ClientConn),
	}
}

// Call invokes a gRPC method.
func (c *grpcClientImpl) Call(
	ctx context.Context,
	integration *entity.Integration,
	method string,
	request interface{},
	response interface{},
	token string,
) error {
	if integration.GRPCConfig == nil {
		return fmt.Errorf("gRPC config is required")
	}

	// Add authorization to context
	md := make(map[string]string)
	if token != "" {
		md["authorization"] = fmt.Sprintf("Bearer %s", token)
	}

	// Add default metadata
	if integration.GRPCConfig.Metadata != nil {
		for key, value := range integration.GRPCConfig.Metadata {
			md[key] = value
		}
	}

	return c.CallWithMetadata(ctx, integration, method, request, response, md)
}

// CallWithMetadata invokes a gRPC method with custom metadata.
func (c *grpcClientImpl) CallWithMetadata(
	ctx context.Context,
	integration *entity.Integration,
	method string,
	request interface{},
	response interface{},
	md map[string]string,
) error {
	if integration.GRPCConfig == nil {
		return fmt.Errorf("gRPC config is required")
	}

	// Get or create connection
	conn, err := c.GetConnection(ctx, integration)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	// Build full method name
	fullMethod := fmt.Sprintf("/%s/%s", integration.GRPCConfig.ServiceName, method)

	// Add metadata to context
	if len(md) > 0 {
		mdPairs := make([]string, 0, len(md)*2)
		for key, value := range md {
			mdPairs = append(mdPairs, key, value)
		}
		ctx = metadata.AppendToOutgoingContext(ctx, mdPairs...)
	}

	// Apply timeout if configured
	if integration.GRPCConfig.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(integration.GRPCConfig.Timeout)*time.Second)
		defer cancel()
	}

	// Invoke the method
	err = conn.Invoke(ctx, fullMethod, request, response)
	if err != nil {
		return fmt.Errorf("gRPC call failed: %w", err)
	}

	return nil
}

// GetConnection gets or creates a gRPC connection.
func (c *grpcClientImpl) GetConnection(ctx context.Context, integration *entity.Integration) (*grpc.ClientConn, error) {
	if integration.GRPCConfig == nil {
		return nil, fmt.Errorf("gRPC config is required")
	}

	// Check if connection exists and is ready
	integrationKey := integration.ID.String()
	if conn, exists := c.connections[integrationKey]; exists {
		state := conn.GetState()
		if state != connectivity.Shutdown {
			return conn, nil
		}
		// Connection is shutdown, remove from cache
		delete(c.connections, integrationKey)
	}

	// Create new connection
	conn, err := c.createConnection(ctx, integration)
	if err != nil {
		return nil, err
	}

	// Cache the connection
	c.connections[integrationKey] = conn

	return conn, nil
}

// createConnection creates a new gRPC connection.
func (c *grpcClientImpl) createConnection(ctx context.Context, integration *entity.Integration) (*grpc.ClientConn, error) {
	config := integration.GRPCConfig

	// Build dial options
	opts := []grpc.DialOption{}

	// TLS configuration
	if config.UseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.ServerName == "", // Skip if no server name provided
		}
		if config.ServerName != "" {
			tlsConfig.ServerName = config.ServerName
		}
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Max message size
	if config.MaxMessageSize > 0 {
		opts = append(opts,
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(config.MaxMessageSize),
				grpc.MaxCallSendMsgSize(config.MaxMessageSize),
			),
		)
	}

	// Keep-alive
	if config.KeepAlive {
		// Default keep-alive settings
		opts = append(opts, grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                10 * time.Second,
				Timeout:             3 * time.Second,
				PermitWithoutStream: true,
			},
		))
	}

	// Create connection
	conn, err := grpc.DialContext(ctx, integration.BaseURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	return conn, nil
}

// CloseConnection closes a connection for an integration.
func (c *grpcClientImpl) CloseConnection(integrationID string) error {
	if conn, exists := c.connections[integrationID]; exists {
		delete(c.connections, integrationID)
		return conn.Close()
	}
	return nil
}

// CloseAllConnections closes all cached connections.
func (c *grpcClientImpl) CloseAllConnections() error {
	for key, conn := range c.connections {
		if err := conn.Close(); err != nil {
			// Log error but continue closing others
			fmt.Printf("error closing connection %s: %v\n", key, err)
		}
		delete(c.connections, key)
	}
	return nil
}
