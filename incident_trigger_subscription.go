package flashduty

import "context"

// IncidentTriggerSubscription represents an Automation subscription to new
// On-call incidents.
type IncidentTriggerSubscription struct {
	AccountID      int64     `json:"account_id" toon:"account_id"`
	ChannelIDs     []int64   `json:"channel_ids" toon:"channel_ids"`
	Consumer       string    `json:"consumer" toon:"consumer"`
	ConsumerRef    string    `json:"consumer_ref" toon:"consumer_ref"`
	CreatedAt      Timestamp `json:"created_at" toon:"created_at"`
	CreatedBy      int64     `json:"created_by" toon:"created_by"`
	DeletedAt      Timestamp `json:"deleted_at" toon:"deleted_at"`
	Enabled        bool      `json:"enabled" toon:"enabled"`
	Severities     []string  `json:"severities" toon:"severities"`
	Source         string    `json:"source" toon:"source"`
	SubscriptionID string    `json:"subscription_id" toon:"subscription_id"`
	UpdatedAt      Timestamp `json:"updated_at" toon:"updated_at"`
	UpdatedBy      int64     `json:"updated_by" toon:"updated_by"`
}

// IncidentTriggerSubscriptionDeleteRequest identifies a subscription to
// remove.
type IncidentTriggerSubscriptionDeleteRequest struct {
	Consumer    string `json:"consumer,omitempty" toon:"consumer,omitempty"`
	ConsumerRef string `json:"consumer_ref,omitempty" toon:"consumer_ref,omitempty"`
	Source      string `json:"source,omitempty" toon:"source,omitempty"`
}

// IncidentTriggerSubscriptionUpsertRequest creates or updates an incident
// trigger subscription. Enabled is pointer-backed so an explicit false is
// distinct from an omitted value.
type IncidentTriggerSubscriptionUpsertRequest struct {
	ChannelIDs     []int64  `json:"channel_ids,omitempty" toon:"channel_ids,omitempty"`
	Consumer       string   `json:"consumer,omitempty" toon:"consumer,omitempty"`
	ConsumerRef    string   `json:"consumer_ref,omitempty" toon:"consumer_ref,omitempty"`
	Enabled        *bool    `json:"enabled,omitempty" toon:"enabled,omitempty"`
	Severities     []string `json:"severities,omitempty" toon:"severities,omitempty"`
	Source         string   `json:"source,omitempty" toon:"source,omitempty"`
	SubscriptionID string   `json:"subscription_id,omitempty" toon:"subscription_id,omitempty"`
}

// TriggerSubscriptionWriteDelete deletes an incident trigger subscription.
//
// API: POST /incident-trigger-subscription/delete
func (s *IncidentsService) TriggerSubscriptionWriteDelete(ctx context.Context, req *IncidentTriggerSubscriptionDeleteRequest) (*Response, error) {
	return s.client.do(ctx, "/incident-trigger-subscription/delete", req, nil)
}

// TriggerSubscriptionWriteUpsert creates or updates an incident trigger
// subscription.
//
// API: POST /incident-trigger-subscription/upsert
func (s *IncidentsService) TriggerSubscriptionWriteUpsert(ctx context.Context, req *IncidentTriggerSubscriptionUpsertRequest) (*IncidentTriggerSubscription, *Response, error) {
	out := new(IncidentTriggerSubscription)
	resp, err := s.client.do(ctx, "/incident-trigger-subscription/upsert", req, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}
