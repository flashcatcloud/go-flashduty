// Package e2e contains live end-to-end tests for the Flashduty SDK. They are
// gated behind the "e2e" build tag and run only when an app key is configured
// via FLASHDUTY_E2E_APP_KEY (or FLASHDUTY_APP_KEY). Run them with: make e2e.
//
// Tests that create resources name them with a "gofd-e2e-" prefix and delete
// them again on cleanup; they never mutate pre-existing data.
package e2e
