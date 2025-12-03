// Package tenant contains the application layer for tenant use cases.
package tenant

import (
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// CreateTenantCommand represents the command to create a tenant.
type CreateTenantCommand struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone,omitempty"`
	Plan         string `json:"plan"`
	BillingEmail string `json:"billing_email,omitempty"`
	Industry     string `json:"industry,omitempty"`
	CompanySize  string `json:"company_size,omitempty"`
	Website      string `json:"website,omitempty"`
}

// Validate validates the create tenant command.
func (cmd CreateTenantCommand) Validate() error {
	if strings.TrimSpace(cmd.Name) == "" {
		return errors.New("name is required")
	}
	if len(cmd.Name) < 2 || len(cmd.Name) > 100 {
		return errors.New("name must be between 2 and 100 characters")
	}

	if strings.TrimSpace(cmd.Email) == "" {
		return errors.New("email is required")
	}
	if !isValidEmail(cmd.Email) {
		return errors.New("invalid email format")
	}

	if cmd.Plan == "" {
		return errors.New("plan is required")
	}
	if !isValidPlan(cmd.Plan) {
		return errors.New("invalid plan, must be one of: starter, professional, enterprise")
	}

	if cmd.BillingEmail != "" && !isValidEmail(cmd.BillingEmail) {
		return errors.New("invalid billing email format")
	}

	return nil
}

// UpdateTenantCommand represents the command to update a tenant.
type UpdateTenantCommand struct {
	ID           uuid.UUID `json:"id"`
	Name         *string   `json:"name,omitempty"`
	Email        *string   `json:"email,omitempty"`
	Phone        *string   `json:"phone,omitempty"`
	BillingEmail *string   `json:"billing_email,omitempty"`
}

// Validate validates the update tenant command.
func (cmd UpdateTenantCommand) Validate() error {
	if cmd.ID == uuid.Nil {
		return errors.New("id is required")
	}

	if cmd.Name != nil {
		name := strings.TrimSpace(*cmd.Name)
		if name == "" {
			return errors.New("name cannot be empty")
		}
		if len(name) < 2 || len(name) > 100 {
			return errors.New("name must be between 2 and 100 characters")
		}
	}

	if cmd.Email != nil {
		if !isValidEmail(*cmd.Email) {
			return errors.New("invalid email format")
		}
	}

	if cmd.BillingEmail != nil && *cmd.BillingEmail != "" {
		if !isValidEmail(*cmd.BillingEmail) {
			return errors.New("invalid billing email format")
		}
	}

	return nil
}

// Helper functions

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func isValidPlan(plan string) bool {
	validPlans := map[string]bool{
		"starter":      true,
		"professional": true,
		"enterprise":   true,
	}
	return validPlans[strings.ToLower(plan)]
}
