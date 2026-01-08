package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
)

type stubValidator struct{}

func (stubValidator) ValidateInput(_ interface{}, _ json.RawMessage) error  { return nil }
func (stubValidator) ValidateOutput(_ interface{}, _ json.RawMessage) error { return nil }
func (stubValidator) IsValidSchema(_ json.RawMessage) error                 { return nil }

type stubToolRepo struct {
	tools map[uuid.UUID]*entity.Tool
}

func newStubToolRepo() *stubToolRepo {
	return &stubToolRepo{tools: make(map[uuid.UUID]*entity.Tool)}
}

func (r *stubToolRepo) Create(_ context.Context, tool *entity.Tool) error {
	if tool.ID == uuid.Nil {
		tool.ID = uuid.New()
	}
	r.tools[tool.ID] = tool
	return nil
}

func (r *stubToolRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.Tool, error) {
	tool, ok := r.tools[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return tool, nil
}

func (r *stubToolRepo) FindByName(_ context.Context, name string) (*entity.Tool, error) {
	for _, tool := range r.tools {
		if tool.Name == name {
			return tool, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *stubToolRepo) FindAll(_ context.Context, _ repository.ToolFilters) ([]*entity.Tool, int64, error) {
	list := make([]*entity.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		list = append(list, tool)
	}
	return list, int64(len(list)), nil
}

func (r *stubToolRepo) Update(_ context.Context, tool *entity.Tool) error {
	if _, ok := r.tools[tool.ID]; !ok {
		return errors.New("not found")
	}
	r.tools[tool.ID] = tool
	return nil
}

func (r *stubToolRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.tools, id)
	return nil
}

func (r *stubToolRepo) FindPublicTools(_ context.Context) ([]*entity.Tool, error) {
	return nil, nil
}

func (r *stubToolRepo) FindByCategory(_ context.Context, _ string) ([]*entity.Tool, error) {
	return nil, nil
}

func TestCreateToolEnforcesHTTPSAndAllowedHosts(t *testing.T) {
	repo := newStubToolRepo()
	svc := NewToolService(repo, stubValidator{}, ExecutionPolicy{AllowedHosts: []string{"api.example.com"}})

	baseTool := &entity.Tool{
		Name:         "test-tool",
		DisplayName:  "Test Tool",
		Description:  "",
		Category:     entity.CategoryOther,
		Method:       entity.HTTPMethodPOST,
		BaseURL:      "",
		EndpointPath: "/v1/run",
		AuthType:     entity.AuthTypeNone.String(),
		InputSchema:  []byte("{}"),
		OutputSchema: []byte("{}"),
	}

	t.Run("rejects non-https scheme", func(t *testing.T) {
		tool := *baseTool
		tool.BaseURL = "http://api.example.com"
		if err := svc.CreateTool(context.Background(), &tool); err == nil {
			t.Fatalf("expected error for http base_url")
		}
	})

	t.Run("rejects host not in allowlist", func(t *testing.T) {
		tool := *baseTool
		tool.BaseURL = "https://evil.example.com"
		if err := svc.CreateTool(context.Background(), &tool); err == nil {
			t.Fatalf("expected error for disallowed host")
		}
	})

	t.Run("allows https host in allowlist", func(t *testing.T) {
		tool := *baseTool
		tool.BaseURL = "https://api.example.com"
		if err := svc.CreateTool(context.Background(), &tool); err != nil {
			t.Fatalf("expected create to succeed, got %v", err)
		}
	})
}

func TestUpdateToolEnforcesAllowedHosts(t *testing.T) {
	repo := newStubToolRepo()
	svc := NewToolService(repo, stubValidator{}, ExecutionPolicy{AllowedHosts: []string{"api.example.com"}})

	original := &entity.Tool{
		ID:           uuid.New(),
		Name:         "test-tool",
		DisplayName:  "Test Tool",
		Description:  "",
		Category:     entity.CategoryOther,
		Method:       entity.HTTPMethodPOST,
		BaseURL:      "https://api.example.com",
		EndpointPath: "/v1/run",
		AuthType:     entity.AuthTypeNone.String(),
		InputSchema:  []byte("{}"),
		OutputSchema: []byte("{}"),
	}

	if err := repo.Create(context.Background(), original); err != nil {
		t.Fatalf("failed to seed tool: %v", err)
	}

	updated := *original
	updated.BaseURL = "https://evil.example.com"

	if err := svc.UpdateTool(context.Background(), &updated); err == nil {
		t.Fatalf("expected error when updating to disallowed host")
	}
}
