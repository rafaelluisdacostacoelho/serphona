// Package handler provides gRPC handlers.
package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"

	"tenant-manager/internal/application/tenant"
	tenantpb "tenant-manager/proto"
)

// TenantHandler implements gRPC tenant service.
type TenantHandler struct {
	tenantpb.UnimplementedTenantServiceServer
	service *tenant.Service
}

// NewTenantHandler creates a new gRPC tenant handler.
func NewTenantHandler(service *tenant.Service) *TenantHandler {
	return &TenantHandler{
		service: service,
	}
}

// CreateTenant creates a new tenant.
func (h *TenantHandler) CreateTenant(ctx context.Context, req *tenantpb.CreateTenantRequest) (*tenantpb.CreateTenantResponse, error) {
	cmd := tenant.CreateTenantCommand{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Plan:         req.Plan,
		BillingEmail: req.BillingEmail,
	}

	if req.Metadata != nil {
		cmd.Industry = req.Metadata.Industry
		cmd.CompanySize = req.Metadata.CompanySize
		cmd.Website = req.Metadata.Website
	}

	result, err := h.service.CreateTenant(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create tenant: %v", err)
	}

	return &tenantpb.CreateTenantResponse{
		Tenant: toProtoTenant(result),
	}, nil
}

// GetTenant retrieves a tenant by ID.
func (h *TenantHandler) GetTenant(ctx context.Context, req *tenantpb.GetTenantRequest) (*tenantpb.GetTenantResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	result, err := h.service.GetTenant(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "tenant not found: %v", err)
	}

	return &tenantpb.GetTenantResponse{
		Tenant: toProtoTenant(result),
	}, nil
}

// UpdateTenant updates an existing tenant.
func (h *TenantHandler) UpdateTenant(ctx context.Context, req *tenantpb.UpdateTenantRequest) (*tenantpb.UpdateTenantResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	cmd := tenant.UpdateTenantCommand{
		ID:    id,
		Name:  req.Name,
		Email: req.Email,
		Phone: nil,
	}

	result, err := h.service.UpdateTenant(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update tenant: %v", err)
	}

	return &tenantpb.UpdateTenantResponse{
		Tenant: toProtoTenant(result),
	}, nil
}

// DeleteTenant soft-deletes a tenant.
func (h *TenantHandler) DeleteTenant(ctx context.Context, req *tenantpb.DeleteTenantRequest) (*tenantpb.DeleteTenantResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	if err := h.service.DeleteTenant(ctx, id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete tenant: %v", err)
	}

	return &tenantpb.DeleteTenantResponse{
		Success: true,
	}, nil
}

// ListTenants lists tenants with pagination.
func (h *TenantHandler) ListTenants(ctx context.Context, req *tenantpb.ListTenantsRequest) (*tenantpb.ListTenantsResponse, error) {
	query := tenant.ListTenantsQuery{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		Status:   req.Status,
		Search:   req.Search,
	}

	result, err := h.service.ListTenants(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tenants: %v", err)
	}

	tenants := make([]*tenantpb.Tenant, len(result.Tenants))
	for i, t := range result.Tenants {
		tenants[i] = toProtoTenant(t)
	}

	return &tenantpb.ListTenantsResponse{
		Tenants:    tenants,
		Total:      result.Total,
		Page:       int32(result.Page),
		PageSize:   int32(result.PageSize),
		TotalPages: int32(result.TotalPages),
	}, nil
}

// ActivateTenant activates a tenant.
func (h *TenantHandler) ActivateTenant(ctx context.Context, req *tenantpb.ActivateTenantRequest) (*tenantpb.ActivateTenantResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	if err := h.service.ActivateTenant(ctx, id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to activate tenant: %v", err)
	}

	// Get updated tenant
	result, err := h.service.GetTenant(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get tenant: %v", err)
	}

	return &tenantpb.ActivateTenantResponse{
		Tenant: toProtoTenant(result),
	}, nil
}

// SuspendTenant suspends a tenant.
func (h *TenantHandler) SuspendTenant(ctx context.Context, req *tenantpb.SuspendTenantRequest) (*tenantpb.SuspendTenantResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	if err := h.service.SuspendTenant(ctx, id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to suspend tenant: %v", err)
	}

	// Get updated tenant
	result, err := h.service.GetTenant(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get tenant: %v", err)
	}

	return &tenantpb.SuspendTenantResponse{
		Tenant: toProtoTenant(result),
	}, nil
}

// toProtoTenant converts application DTO to proto message.
func toProtoTenant(dto *tenant.TenantDTO) *tenantpb.Tenant {
	// Convert settings (struct to map)
	settings := make(map[string]string)
	// Add settings if needed - dto.Settings is a struct, not a map

	// Convert metadata map
	metadata := make(map[string]string)
	metadata["industry"] = dto.Metadata.Industry
	metadata["company_size"] = dto.Metadata.CompanySize
	metadata["website"] = dto.Metadata.Website

	return &tenantpb.Tenant{
		Id:           dto.ID.String(),
		Name:         dto.Name,
		Slug:         dto.Slug,
		Email:        dto.Email,
		Phone:        dto.Phone,
		Status:       dto.Status,
		Plan:         dto.Plan,
		Settings:     settings,
		Metadata:     metadata,
		CreatedAt:    timestamppb.New(dto.CreatedAt),
		UpdatedAt:    timestamppb.New(dto.UpdatedAt),
		BillingEmail: dto.BillingEmail,
	}
}
