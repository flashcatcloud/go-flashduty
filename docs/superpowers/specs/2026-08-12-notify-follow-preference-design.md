# Notification Follow Preference SDK Fix

## Problem

`POST /incident/create` accepts `assigned_to.notify.follow_preference` as a
tri-state value in the backend:

- omitted: use each responder's personal notification preference;
- `true`: explicitly use personal preference;
- `false`: use `personal_channels` from the request.

The OpenAPI schema currently models the field as an optional, non-nullable
boolean. The SDK generator therefore emits:

```go
FollowPreference bool `json:"follow_preference,omitempty"`
```

Go's JSON encoder omits `false`, so SDK callers cannot send the value required
to force `personal_channels`.

## Design

Model `follow_preference` as a nullable boolean in the OpenAPI 3.1 source:

```json
"type": ["boolean", "null"]
```

The existing generator already maps nullable request scalars to pointers. The
generated SDK field becomes:

```go
FollowPreference *bool `json:"follow_preference,omitempty"`
```

Callers use `flashduty.Bool(false)` to send an explicit false value. A nil
pointer remains omitted.

The same request contract is used by incident creation and responder addition,
so both request schemas must be corrected. The incident assignment schema must
also expose `assigned_to.notify`, which the backend accepts through the shared
assignment structure.

## Source And Generation Flow

1. Correct the English and Chinese on-call OpenAPI source in
   `flashduty-docs`.
2. Regenerate the consolidated bilingual OpenAPI artifacts there.
3. Sync those artifacts into `go-flashduty`.
4. Run the SDK generator; do not edit `models_gen.go` directly.

## Compatibility

Changing `FollowPreference` from `bool` to `*bool` is a source-level change for
callers that initialize this field. Release it as the next v0 minor version,
not as a patch release.

Removing `omitempty` is rejected because it would silently send `false` for a
notification override that only specifies `template_id`, changing that request
from personal-preference delivery to an empty explicit channel override.

## Verification

- Generator test: an optional nullable request boolean generates `*bool` with
  `omitempty`.
- Wire test: `flashduty.Bool(false)` serializes as
  `"follow_preference":false`.
- Wire test: nil omits `follow_preference`.
- Cover both incident creation and responder addition.
- Regenerate twice and confirm no diff on the second run.
- Run `make check` in `go-flashduty`.

## Success Criteria

An SDK request with `PersonalChannels: []string{"sms"}` and
`FollowPreference: flashduty.Bool(false)` sends both fields, allowing the
backend to select SMS explicitly.
