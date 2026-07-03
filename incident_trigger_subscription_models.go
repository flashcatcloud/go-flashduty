package flashduty

// IncidentTriggerSubscriptionUpsertRequest creates or updates an incident trigger subscription.
//
// Enabled is pointer-backed because omitting it defaults to true, while a
// non-nil false explicitly disables the subscription.
type IncidentTriggerSubscriptionUpsertRequest struct {
	// On-call channel IDs whose new incidents should trigger the consumer.
	ChannelIDs []int64 `json:"channel_ids,omitempty" toon:"channel_ids,omitempty"`
	// Consumer system. Use `fc_safari` for AI SRE automation rules.
	Consumer string `json:"consumer,omitempty" toon:"consumer,omitempty"`
	// Consumer-owned reference, such as an Automation rule ID.
	ConsumerRef string `json:"consumer_ref,omitempty" toon:"consumer_ref,omitempty"`
	// Whether the subscription is enabled. Defaults to true when omitted.
	Enabled *bool `json:"enabled,omitempty" toon:"enabled,omitempty"`
	// Incident severities to subscribe to. `Ok` is not valid.
	Severities []string `json:"severities,omitempty" toon:"severities,omitempty"`
	// Subscription source. Use `ai_sre_automation` for AI SRE automation rules.
	Source string `json:"source,omitempty" toon:"source,omitempty"`
	// Existing subscription ID. Omit to create or upsert by source, consumer, and consumer_ref.
	SubscriptionID string `json:"subscription_id,omitempty" toon:"subscription_id,omitempty"`
}
