package subscription

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	StatusActive            SubscriptionStatus = "active"
	StatusCanceled          SubscriptionStatus = "canceled"
	StatusIncomplete        SubscriptionStatus = "incomplete"
	StatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
	StatusPastDue           SubscriptionStatus = "past_due"
	StatusTrialing          SubscriptionStatus = "trialing"
	StatusUnpaid            SubscriptionStatus = "unpaid"
)

// Subscription represents a customer subscription
type Subscription struct {
	ID                   uuid.UUID              `json:"id" gorm:"type:uuid;primary_key"`
	CustomerID           uuid.UUID              `json:"customer_id" gorm:"type:uuid;not null;index"`
	StripeSubscriptionID string                 `json:"stripe_subscription_id" gorm:"unique;not null"`
	PlanID               string                 `json:"plan_id" gorm:"not null"`
	Status               SubscriptionStatus     `json:"status" gorm:"not null"`
	CurrentPeriodStart   time.Time              `json:"current_period_start"`
	CurrentPeriodEnd     time.Time              `json:"current_period_end"`
	CancelAtPeriodEnd    bool                   `json:"cancel_at_period_end" gorm:"default:false"`
	CanceledAt           *time.Time             `json:"canceled_at"`
	TrialStart           *time.Time             `json:"trial_start"`
	TrialEnd             *time.Time             `json:"trial_end"`
	Metadata             map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	CreatedAt            time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for Subscription
func (Subscription) TableName() string {
	return "subscriptions"
}

// NewSubscription creates a new subscription instance
func NewSubscription(customerID uuid.UUID, planID string) *Subscription {
	return &Subscription{
		ID:                uuid.New(),
		CustomerID:        customerID,
		PlanID:            planID,
		Status:            StatusIncomplete,
		CancelAtPeriodEnd: false,
		Metadata:          make(map[string]interface{}),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// SetStripeSubscriptionID sets the Stripe subscription ID
func (s *Subscription) SetStripeSubscriptionID(stripeID string) {
	s.StripeSubscriptionID = stripeID
	s.UpdatedAt = time.Now()
}

// Activate activates the subscription
func (s *Subscription) Activate(periodStart, periodEnd time.Time) {
	s.Status = StatusActive
	s.CurrentPeriodStart = periodStart
	s.CurrentPeriodEnd = periodEnd
	s.UpdatedAt = time.Now()
}

// Cancel cancels the subscription
func (s *Subscription) Cancel(cancelAtPeriodEnd bool) {
	if cancelAtPeriodEnd {
		s.CancelAtPeriodEnd = true
	} else {
		s.Status = StatusCanceled
		now := time.Now()
		s.CanceledAt = &now
	}
	s.UpdatedAt = time.Now()
}

// UpdateStatus updates the subscription status
func (s *Subscription) UpdateStatus(status SubscriptionStatus) {
	s.Status = status
	s.UpdatedAt = time.Now()
}

// IsActive checks if the subscription is active
func (s *Subscription) IsActive() bool {
	return s.Status == StatusActive || s.Status == StatusTrialing
}

// StartTrial starts a trial period
func (s *Subscription) StartTrial(trialEnd time.Time) {
	s.Status = StatusTrialing
	now := time.Now()
	s.TrialStart = &now
	s.TrialEnd = &trialEnd
	s.UpdatedAt = time.Now()
}
