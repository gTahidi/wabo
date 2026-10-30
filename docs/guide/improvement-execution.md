# SDK improvement review and execution

Reviewed input: `/home/ambuche/Downloads/wapi_meta_v26_sdk_research.md`, dated
29 October 2026. This implementation addresses confirmed correctness defects
and adds the foundations for a separately verified compatibility release.

## Implemented

- `ClientConfig.APIVersion` propagates through the shared Graph requester to
  messaging, business, media and Flow managers. Empty retains `v24.0`; explicitly
  set `v26.0` when testing that version. Configuration does not certify API support.
- Incoming documents emit `DocumentMessageEvent` with the correct media ID,
  checksum, MIME type, caption and optional filename.
- Incoming contact cards populate `ContactCards` and normalized contact events.
  The old `Contacts` identity-list field remains for source compatibility but is
  excluded from JSON; it was not the correct model for incoming contact cards.
- Each message's sender name is matched by its nonempty `from` and contact `wa_id`.
  Unmatched senders do not inherit another person's name. Direct dispatch callers
  without contact metadata retain their supplied `SenderName` fallback.
- `BaseMessageEvent.RawMessage` retains the complete inbound JSON, including
  identity fields not yet modeled. `Referral` retains the entire optional referral
  object, including future fields and referrals without a click ID.
- Unrecognized change fields, message types and system notification types emit
  `UnknownWebhookEvent` under `events.UnknownEventType`. Unknown system types are
  no longer classified as phone-number changes.
- Changes decode to `json.RawMessage` inside the existing `interface{}` Value
  field, avoiding loss of large numeric identifiers. Consumers that type-asserted
  decoded Values to maps should now decode the raw JSON explicitly.
- Pricing retains optional `type` and `billable`; a `*bool` distinguishes false
  from absence. This is a decoding improvement, not an invoice calculation.
- Template button indices accept 0 through 9; business fetch returns JSON errors.

## Evidence and limitations

Regression tests use synthetic wire fixtures derived from the documented bug
shapes, not claimed official Meta captures. They exercise the HTTP webhook
handler and event subscribers, transport URL construction, decoding failures and
button validation. All outbound HTTP in tests is stubbed.

Direct attempts to read Meta's BSUID documentation and Graph v26 changelog failed
with a rate limit or access error on 30 October 2026. Consequently this change
makes no claim of complete v26 or username compatibility. Live API requests were
not performed.

## Remaining work, in order

1. Obtain accessible official Meta contracts or sanitized account captures for
   BSUID sends, responses, contact matching, identity rotation and pricing analytics.
   Verify exact `recipient` versus `to` routing before adding outbound methods.
2. Add typed identity/status models and normalized identity-change events against
   those fixtures. Raw retention currently protects inbound message data, but
   does not supply a typed BSUID send/reply path or normalized status pricing.
3. Implement pricing analytics with endpoint contract tests; only then reconsider
   the default API version and publish a supported-version policy.
4. Extract independently testable webhook parsing and standard-library HTTP
   handling; define body limits, durable delivery semantics and retries explicitly.
5. Incrementally separate domain responsibilities while retaining old imports.
6. Evaluate optional calling, username-management and advertising capabilities
   against verified contracts and account eligibility. Keep Marketing API work
   separate from ordinary Cloud API messaging.

The subsequent Wabo migration changes the module path to `github.com/gTahidi/wabo`; see [the migration guide](migrating-to-wabo.md). No optional advertising
APIs or package moves are part of this patch.

## Validation results

- `GOCACHE=/tmp/wapi-go-cache go test ./...`: passed, including existing tests.
- `GOCACHE=/tmp/wapi-go-cache go build ./...`: passed.
- `PATH=/tmp/wapi-tools:$PATH GOCACHE=/tmp/wapi-go-cache make docs GOMARKDOS=/tmp/wapi-tools/gomarkdoc`:
  passed using gomarkdoc v1.1.0. Regeneration includes previously stale references.
- `git diff --check`: reports one trailing blank line emitted by the documentation
  generator in `docs/api-reference/pkg/business.mdx`; no source whitespace errors.
