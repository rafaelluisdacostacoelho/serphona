package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"

	"tools-manager/internal/audit"
	"tools-manager/internal/repository"
)

// QuotaHandler manages quota rule endpoints.
type QuotaHandler struct {
	log  *zap.Logger
	repo *repository.Repository
}

func NewQuotaHandler(log *zap.Logger, repo *repository.Repository) *QuotaHandler {
	return &QuotaHandler{log: log, repo: repo}
}

// createQuotaRuleRequest defines admin payload for quota rules.
type createQuotaRuleRequest struct {
	ToolID         *string `json:"tool_id"`
	AgentID        *string `json:"agent_id"`
	Env            *string `json:"env"`
	LimitPerMinute *int    `json:"limit_per_minute"`
	LimitPerDay    *int    `json:"limit_per_day"`
	Weight         int     `json:"weight"`
}

func (h *QuotaHandler) List(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	rules, err := h.repo.ListQuotaRules(c.Request.Context(), claims.TenantID)
	if err != nil {
		h.log.Error("list quota rules failed", zap.Error(err))
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to list quota rules", nil)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, rules)
}

func (h *QuotaHandler) Create(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	var req createQuotaRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	if err := validateQuotaReq(req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	var toolIDPtr *uuid.UUID
	if req.ToolID != nil && *req.ToolID != "" {
		id, err := uuid.Parse(*req.ToolID)
		if err != nil {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", "tool_id must be a UUID", nil)
			return
		}
		toolIDPtr = &id
	}

	var tenantUUID *uuid.UUID
	if claims.TenantID != "platform" {
		id, err := uuid.Parse(claims.TenantID)
		if err != nil {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", "tenant_id in claims must be a UUID", nil)
			return
		}
		tenantUUID = &id
	}

	creatorID, _ := uuid.Parse(claims.UserID)

	rule, err := h.repo.CreateQuotaRule(c.Request.Context(), claims.TenantID, repository.CreateQuotaRuleParams{
		TenantID:       tenantUUID,
		ToolID:         toolIDPtr,
		AgentID:        req.AgentID,
		Env:            req.Env,
		LimitPerMinute: req.LimitPerMinute,
		LimitPerDay:    req.LimitPerDay,
		Weight:         req.Weight,
		CreatedBy:      creatorID,
	})
	if err != nil {
		h.log.Error("create quota rule failed", zap.Error(err))
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to create quota rule", nil)
		return
	}

	audit.Emit(h.log, audit.Event{
		Category: "quota_rule",
		Action:   "create",
		Outcome:  "success",
		TenantID: claims.TenantID,
		UserID:   claims.UserID,
		Service:  claims.Service,
		Path:     c.FullPath(),
		RuleID:   rule.ID.String(),
	})

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusCreated, rule)
}

func validateQuotaReq(req createQuotaRuleRequest) error {
	if req.LimitPerMinute != nil && *req.LimitPerMinute < 0 {
		return errors.New("limit_per_minute must be non-negative")
	}
	if req.LimitPerDay != nil && *req.LimitPerDay < 0 {
		return errors.New("limit_per_day must be non-negative")
	}
	if req.LimitPerMinute == nil && req.LimitPerDay == nil {
		return errors.New("at least one of limit_per_minute or limit_per_day is required")
	}
	if req.Weight < 0 {
		return errors.New("weight must be non-negative")
	}
	return nil
}
