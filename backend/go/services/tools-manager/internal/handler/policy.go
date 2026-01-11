package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"

	"tools-manager/internal/audit"
	"tools-manager/internal/repository"
)

// PolicyHandler manages policy rule endpoints.
type PolicyHandler struct {
	log  *zap.Logger
	repo *repository.Repository
}

func NewPolicyHandler(log *zap.Logger, repo *repository.Repository) *PolicyHandler {
	return &PolicyHandler{log: log, repo: repo}
}

// createPolicyRuleRequest defines admin payload for policy rules.
type createPolicyRuleRequest struct {
	Effect  string   `json:"effect" binding:"required"`
	ToolID  *string  `json:"tool_id"`
	AgentID *string  `json:"agent_id"`
	Env     *string  `json:"env"`
	Scopes  []string `json:"scopes"`
	Roles   []string `json:"roles"`
	Weight  int      `json:"weight"`
}

func (h *PolicyHandler) List(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	rules, err := h.repo.ListPolicyRules(c.Request.Context(), claims.TenantID)
	if err != nil {
		h.log.Error("list policy rules failed", zap.Error(err))
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to list policy rules", nil)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, rules)
}

func (h *PolicyHandler) Create(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	var req createPolicyRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	if err := validatePolicyReq(req); err != nil {
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

	effect := strings.ToLower(req.Effect)
	creatorID, _ := uuid.Parse(claims.UserID)

	rule, err := h.repo.CreatePolicyRule(c.Request.Context(), claims.TenantID, repository.CreatePolicyRuleParams{
		TenantID:  tenantUUID,
		ToolID:    toolIDPtr,
		AgentID:   req.AgentID,
		Env:       req.Env,
		Effect:    effect,
		Scopes:    req.Scopes,
		Roles:     req.Roles,
		Weight:    req.Weight,
		CreatedBy: creatorID,
	})
	if err != nil {
		h.log.Error("create policy rule failed", zap.Error(err))
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to create policy rule", nil)
		return
	}

	audit.Emit(h.log, audit.Event{
		Category: "policy_rule",
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

func validatePolicyReq(req createPolicyRuleRequest) error {
	effect := strings.ToLower(req.Effect)
	switch effect {
	case "allow", "deny":
	default:
		return errors.New("effect must be allow or deny")
	}
	if req.Weight < 0 {
		return errors.New("weight must be non-negative")
	}
	return nil
}
