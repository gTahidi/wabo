# Wabo and Wabble: compatibility, availability, and launch plan

Date: 2026-11-01. Status: investigation and implementation plan; not a claim of Meta certification.

## Outcome and release scope

Commercial direction, clarified 2026-11-01: **Feem Digitals is the agency partner
and customer-acquisition front; Nextbird is the customer-facing product; Wabble
is its application and onboarding engine; Wabo is its WhatsApp/Graph SDK.**
Sell locally managed WhatsApp lead-generation and follow-up services, using real
agency engagements to validate the underlying technology. Customer acquisition
cost (CAC) is the working interpretation of the user's "cac" reference.

Launch Wabble as an application powered by Wabo that can onboard a business,
receive leads from click-to-WhatsApp ads, support conversations, retain ad
attribution, and report verified business outcomes. Publish Wabo with a precise
supported-capability matrix and a working Wabble integration example.

Start with customers creating ads in Ads Manager. Add programmatic campaign
management after attribution and operational reliability work. Dedicated Marketing
Messages API and WhatsApp Status placements have separate eligibility and
contract checks; neither follows automatically from Cloud API or Tech Provider access.

The user clarified that **Nextbird is the target organization**. The account
identified by `daniel@ujaotech` is the reported Meta login, not necessarily a
matching Wabble User email. Nextbird takes priority over the earlier exploratory
Diaspora inventory and pilot suggestion.
Tech Provider access is user-reported until corroborated by app/account evidence.
Do not infer provider approval from a Wabble user email, a stored token, or one
successful API request. Meta credentials belong to an app and authorized assets.

## Boundaries and evidence handling

- Immediate work: inspect source and a bounded set of matching database records;
  issue read-only Meta requests; run offline tests; record sanitized findings.
- Do not send customer messages, request syncs, register/deregister numbers,
  change subscriptions, exchange stored historical codes, create ads, or spend.
- Keep DSNs, access tokens, app secrets, webhook secrets, authorization codes,
  customer conversations, and raw credential responses out of this document and Git.
- Prefer the existing local database snapshot. Record its freshness and do not
  silently equate snapshot contents with production configuration.
- Load credentials directly in the checking process. Redact upstream messages
  and URLs, which can contain credentials. Log status, error code/subcode,
  capability names, counts, version headers, and masked identifiers only.
- Database access must use read-only transactions and bounded queries/timeouts.
  Network/sandbox approvals remain necessary when the environment requires them.

## Observed baseline

- Wabo checkout: `30f5ec9`, tag `v0.10.0-beta.3`; Go 1.26.
- Wabble source: `/home/ambuche/repos/wabble/wabble`.
- Wabble `go.mod` pins published `github.com/gTahidi/wabo v0.10.0-beta.3` without
  a local replacement. Local `wapi` import aliases are retained deliberately.
- Wabble `internal/core/meta_signup/client.go` uses Wabo for OAuth, WABA and
  portfolio reads, phone reads, webhook subscription, registration, and SMB sync.
- Wabble signup hardcodes `v24.0` in `internal/core/meta_signup/service.go`;
  other factories generally inherit the SDK's v24.0 default. Configuring v26
  in a frontend variable does not establish end-to-end backend version selection.
- Wabo preserves inbound `RawMessage` and `Referral`. Wabble's normalized webhook
  handlers need a persistence trace before we can claim attribution support.
- Wabble has a separate webhook signature-verification layer. Assess that layer
  rather than assuming SDK clients lacking AppSecret mean unauthenticated ingress.
- Wabo event queues are memory-backed, capacity 100 per event type; webhook
  dispatch ignores publication failures. This needs a durable application boundary.
- Wabble's local runtime documentation says the database contains copied production
  data and disables signup and workers in its current organization-scoped runtime.
  Do not enable those features simply to conduct an availability check.

## Ownership and interface contracts

| Responsibility | Wabo SDK | Wabble application |
| --- | --- | --- |
| Graph transport | Contexts, typed errors, version selection, HTTP injection | Per-tenant credentials, configured factories, diagnostics |
| WhatsApp domain | Message/template/media/phone/Flow APIs and wire models | Inbox, workflow, authorization, billing, customer experience |
| Onboarding | OAuth and Graph primitives | Signup frontend, state validation, asset binding, persistence, recovery |
| Attribution | Typed referral model plus raw retention | Persist source and click IDs against leads/conversations |
| Conversion reporting | Narrow typed business-messaging client | Dataset selection, eligibility, event ledger, retries, consent |
| Ads management | Separate Marketing API client/package | Campaign UI, budget approval, tenant asset selection, audit trail |
| Delivery reliability | Parsing and explicit error contracts | Durable ingress/outbox, deduplication, replay and monitoring |

## Phase 0: immediate account and capability inventory

1. Identify current Wabble checkout, runtime config and database source without
   printing secrets. Read live schema metadata before relying on stale HCL.
2. Locate exact/case-normalized matching user email; inspect only its organization
   memberships, linked WABAs, phone metadata, and signup credential metadata.
3. Compare runtime app ID to the token's issuing app where token introspection
   is available. Record expiry, validity, scopes, granular asset grants, and
   app mismatch. Do not assume the configured app is the provider-approved app.
4. With each relevant usable token, check WABA and phone reads on v24 and the
   proposed v26 target. Read current subscriptions; never modify them here.
5. Probe the token identity and authorized ad accounts where supported. Distinguish
   a denied endpoint, expired token, missing scope, empty authorized collection,
   unavailable asset, and regional/product ineligibility.
6. Inspect provider/app review and advanced access in the Meta dashboard if API
   results cannot establish them. Lack of a dashboard session is an explicit gap.
7. Record a capability table using statuses: confirmed read, denied, expired,
   missing credential, not implemented, and requires controlled write/pilot.

Acceptance: a sanitized inventory linked to the correct organization and app,
with timestamps, queried versions, source freshness, and specific blockers.

## Phase 1: lock Wabo–Wabble compatibility

1. Run Wabo build/tests and Wabble build/tests using the pinned published SDK.
2. Use a temporary module file outside either repo to repeat Wabble checks against
   this Wabo checkout. Do not introduce a production local replacement.
3. Trace changed webhook models through Wabble handlers: documents, contact cards,
   raw Change.Value, pricing presence, sender names, unknown events, and referrals.
4. Replay sanitized real payloads through SDK parsing and Wabble persistence with
   stubbed outbound HTTP and an isolated test database. Verify exact identifiers
   survive and tenant boundaries hold.
5. Centralize version selection in Wabble's factories and signup configuration;
   apply it to messaging, management, media, Flows, sync and analytics.
6. Keep v24 default until the advertised operations pass the target-version matrix.
   Record actual response version headers to detect server-side version upgrades.
7. Follow pagination before Wabble treats a phone-number list as authoritative.
   The SDK currently returns one page, while Wabble's sync deletes stored numbers
   absent from that returned list. A partial list must never trigger stale deletion.

Acceptance: both builds/test suites pass, no unexplained dependency/model breaks,
and each advertised operation has versioned contract evidence. Unit tests alone
do not certify live API or account eligibility.

## Phase 2: production onboarding and messaging

1. Confirm provider-approved app, business verification, app mode, required
   permissions, approved access, redirect origins, callback configuration and billing.
2. Run Embedded Signup with an authorized external pilot business and test phone.
3. Verify code exchange, asset ownership/binding, subscription, registration path,
   token validity/expiry and recovery without logging credentials.
4. Test normal Cloud onboarding and coexistence separately. Respect coexistence
   prerequisites and sync windows; historical sync is a controlled state change.
5. Verify approved templates, inbound/outbound media, replies and delivery statuses.
   Test account limits, consent/opt-out handling, applicable service windows, token
   revocation, access removal, and reconnect without cross-tenant asset reuse.
6. Store credential provenance and lifecycle metadata; remove historical codes when
   no longer needed and define credential rotation and access removal procedures.

Acceptance: an external pilot can connect, exchange messages and reconnect using
the approved app. Capture sanitized results; separately track accepted vs delivered.

## Phase 3: make paid lead ingestion reliable

1. Authenticate incoming payloads using the correct issuing app secret, including
   explicitly configured multiple apps if existing accounts require them.
2. Persist an authenticated inbound envelope durably before acknowledging success.
3. Process via workers with per-tenant isolation, bounded retries and dead-letter replay.
4. Deduplicate message/status processing using appropriate stable event keys;
   preserve legitimate status transitions and handle out-of-order delivery.
5. Propagate queue-full/publication errors; do not acknowledge lost events.
6. Verify restart, backlog, malformed event, duplicate delivery and downstream failure.
7. Trace cache expiry and handler lifecycle: cached clients create goroutines, so
   rebuilding clients must not accumulate abandoned subscriber workers.

Acceptance: injected duplicates/restarts/backpressure cause neither lost accepted
leads nor duplicate business actions; failures remain observable and replayable.

## Phase 4: attribution and business-messaging conversions

1. Add a typed optional ad referral in Wabo while preserving unknown/raw fields.
   Include source ID/type/URL, creative metadata and click ID when actually present.
2. Add Wabble attribution storage bound to organization, WABA, phone, inbound
   message, conversation/lead and observation time. Do not assume every message
   contains referral metadata or overwrite an existing source with absent data.
3. Verify dataset authorization and exact current business-messaging permissions,
   supported event names, identity fields and regional/account availability.
4. Implement a Wabo conversion client with context, typed errors and exact wire tests.
5. Add Wabble's conversion ledger/outbox, unique application event identity,
   eligibility checks, bounded retries and diagnostic status. Confirm current Meta
   deduplication semantics; do not assume website CAPI behavior applies.
6. Use an approved small-budget CTWA pilot: ad click, actual message, lead/order
   outcome, event acceptance and reporting. Reconcile stored referral with the ad.
7. Verify missing click ID, organic entry, repeat clicks, duplicate outcome,
   unsupported event, expired token and delayed reporting. Separate messaging-thread
   outcomes from website outcomes according to the verified endpoint contract.

Acceptance: every instrumented pilot lead retains its available source; conversions
are neither fabricated nor duplicated. Meta event acceptance and Ads Manager
attribution are independently evidenced; define an observation window before launch.

## Phase 5: programmatic ads

1. Verify customer-authorized ad accounts and Page/WhatsApp identity linkage.
2. Confirm current Marketing API access requirements for customer assets; obtain
   approved ads permissions and any Page/business permissions required by the flow.
3. Implement separate campaign, ad set, creative, ad and insights APIs in Wabo.
   Verify objective/destination/optimization combinations against current contracts.
4. Create paused drafts, read them back, and show final budget/account/identity in
   Wabble. Require explicit activation; no automatic spend from onboarding.
5. Handle partial creation, ambiguous timeouts, reconciliation and account limits.
   Do not blindly retry creation and produce duplicate spend-capable objects.
6. Verify scheduling, pause/resume, budget changes, reporting pagination and errors.
7. Keep dedicated Marketing Messages API and Status placement support behind
   individually verified capability flags rather than one generic ads flag.

Acceptance: a pilot can review a paused campaign, explicitly activate it within an
approved budget, observe the intended WhatsApp destination and pause it reliably.

## Phase 6: release and marketing

- Publish Wabo capability/version matrix, known exclusions and Wabble example.
- Publish Wabble onboarding and operating guides, including account prerequisites,
  billing responsibility and support path for lost/revoked asset access.
- Begin with an agreed small pilot cohort and one customer segment that already
  closes sales in WhatsApp. Cohort size and spend require business decisions.
- Demonstrate the full ad-to-chat-to-outcome flow with consented/sanitized evidence.
- Suggested initial claim: "Capture WhatsApp ad leads, manage follow-up, and track
  business outcomes." Only claim conversion reporting once that path is validated.
- Keep ad spend and Meta messaging charges distinct from Wabble subscription fees.
- Do not advertise complete v26 support, every WhatsApp ads placement, guaranteed
  delivery, improved return on spend, or partner status beyond verified evidence.

## Release gates and dependencies

| Gate | Evidence | Release decision |
| --- | --- | --- |
| G0 account access | Correct app, usable scoped token, reachable assets | Required for live pilot |
| G1 integration | Pinned SDK and local checkout builds/tests, contract fixtures | Required for SDK/app release |
| G2 onboarding | External pilot succeeds, revocation/reconnect verified | Required for self-service onboarding claim |
| G3 reliability | Durable ingestion and restart/duplicate/backpressure checks | Required before paid lead traffic |
| G4 attribution | Real CTWA referral persisted and linked to lead | Required for attribution claim |
| G5 conversions | Accepted event plus independently checked reporting | Required for conversion-reporting claim |
| G6 campaign control | Paused draft, approved activation, reliable pause | Required for in-app ads management |

Phases 0–1 can proceed immediately. App/provider review and contract research can
run alongside integration work. Paid traffic waits for G0–G3. Campaign management
does not block an Ads Manager-led pilot. No fixed calendar promise until the
credential/app inventory and Meta approval gaps are known.

## Evidence sources

- Wabble: `go.mod`, `docs/wabo-integration.md`, `dev/local/README.md`,
  `internal/core/meta_signup/{client,service,repository}.go`,
  `internal/core/utils/whatsapp_utils.go`, and `api/controllers/webhook_controller/`.
- Wabo: `internal/request_client`, `manager/{event,webhook}_manager.go`,
  `pkg/{client,oauth,messaging,business,events}` and the existing implementation report.
- [Meta Marketing API collection](https://www.postman.com/meta/facebook-marketing-api/collection/0zr4mes/facebook-marketing-api-mapi).
- [Meta business-messaging conversion sample](https://github.com/fbsamples/lead-ads-webhook-sample/blob/main/postman/FB%20Conversions%20API%20%28Part%201%20-%20online%29.postman_collection.json).
- [WhatsApp partner program](https://whatsappbusiness.com/partners/become-a-partner/).
- [Meta Status ads announcement](https://about.fb.com/news/2025/06/helping-you-find-more-channels-businesses-on-whatsapp/).
- Direct Meta developer reference fetches in the preceding research returned 429;
  confirm current endpoint, permission and eligibility contracts before implementation.

## Immediate investigation results

Executed 2026-11-01, approximately 09:04–09:07 UTC. All live requests were GETs.
All database transactions were read-only. No historical codes were exchanged,
messages sent, syncs started, subscriptions altered, or ads created/activated.

### Account resolution and source limits

- Neither `daniel@ujaotech` nor `daniel@ujaotech.com` exists as a User email in the
  inspected local snapshot. This does not contradict Tech Provider access under
  that Meta login; a Meta login and a Wabble login need not share an email.
- Nearby Daniel accounts resolve to organization memberships in Nextbird,
  Bastion, Diaspora Directory and a workspace without a linked WABA.
- Their three linked WABA credentials were checked in memory; no secrets were
  printed or written to this repository. Expiry and DisconnectedAt are unset in
  all three local WABA rows. Unset expiry is not proof of token validity.
- Snapshot capture time is not independently established. The newest User record
  timestamp observed is 2026-10-28; this is a freshness clue, not a backup timestamp.
- Runtime app credentials are loaded from `dev/local/.env.runtime`. The configured
  app ID ends in `9581`; its app-authenticated read succeeds.
- Mapping the Ujaotech Meta login to a specific approved provider app/dashboard
  remains unresolved. No authenticated Meta dashboard inspection was performed.

### Live capability matrix

Identifiers below are deliberately masked. A readable WABA does not prove that a
token can send messages, that a phone is eligible, or that App Review is complete.

| Organization | Issuing app | WABA and phone reads | Subscriptions | Advertising access |
| --- | --- | --- | --- | --- |
| Nextbird | App ending `1067`, different from runtime | v24 and v26 succeed; zero phones returned | One app, ending `1067` | `me/adaccounts` denied, HTTP 403/code 200; ads scopes absent |
| Bastion | App ending `1067`, different from runtime | v24 and v26 succeed; two phones returned | Apps ending `1067` and `8079` | `me/adaccounts` denied, HTTP 403/code 200; ads scopes absent |
| Diaspora Directory | App ending `9581`, matches runtime | v24 and v26 succeed; two phones returned | Apps ending `3001` and `9581` | Active owned ad account and campaign reads confirmed through owning business |

Observed phone fields:

- Bastion: one PENDING/NOT_APPLICABLE and one CONNECTED/CLOUD_API phone; both
  report `code_verification_status=EXPIRED`.
- Diaspora: one PENDING/NOT_APPLICABLE phone with NOT_VERIFIED, and one
  CONNECTED/CLOUD_API phone with EXPIRED. Do not equate the verification-status
  string with a proven inability to send; no send was attempted.
- Graph response headers explicitly reported the requested v24.0/v26.0 versions.
  The reads therefore were not merely silently upgraded requests to another version.

Diaspora credential and asset evidence:

- `debug_token`: `is_valid=true`, type SYSTEM_USER, issuing app matches runtime;
  expiry and data-access-expiry are both 0 in the returned metadata.
- Granted scopes include `ads_management`, `ads_read`, `business_management`,
  `whatsapp_business_management`, `whatsapp_business_messaging`,
  `whatsapp_business_manage_events`, `pages_show_list` and `manage_app_solution`.
  These grants do not independently prove Advanced Access approval for all customers.
- `GET /me/adaccounts` and `GET /me/businesses` both return empty collections.
  These endpoints alone would incorrectly suggest no usable advertising assets.
- `GET /{waba}?fields=id,owner_business_info` resolves the owning business.
- `GET /{business}?fields=id,verification_status` reports `verified`.
- `GET /{business}/owned_ad_accounts` returns two accounts: one with
  account_status=1 and disable_reason=0 (ID ending `0987`), and another with
  account_status=2 and disable_reason=3. Use the active account for a future pilot;
  do not change the disabled account as part of this investigation.
- Direct read of the active ad account succeeds. `GET /{ad-account}/campaigns`
  succeeds and returns three PAUSED campaigns in the requested sample of three.
  Paging metadata is present; this is a sample, not a complete campaign inventory.
- `GET /{business}/client_ad_accounts` returns no entries.
- Nextbird/Bastion token introspection with runtime app credentials returns
  HTTP 400/code 100. Independent `GET /app` confirms their different issuing app.
  Their successful WABA reads demonstrate they must not be labeled expired solely
  because runtime-app introspection failed.

### Compatibility validation

Wabble revision: `489a076c6d69a13c27a5ebe9f020dd2495fa4f36`.
Wabo revision: `30f5ec9da746f378643b62602c1ac241b52ca2af`, tag v0.10.0-beta.3.

| Check | Outcome |
| --- | --- |
| Wabo `go test ./...` and `go build ./...` | Passed in preceding review of the same unchanged revision |
| Wabble `GOCACHE=/tmp/wabo-review-cache go build ./...` | Passed; existing sqlite C compiler warning |
| Wabble `GOCACHE=/tmp/wabo-review-cache go test ./...` | Passed after allowing localhost test listeners; initial sandbox run failed only when httptest could not bind a port |
| Wabble focused signup/utils/webhook tests with published SDK | Passed |
| Same focused tests with temporary modfile replacing Wabo with local checkout | Passed |
| Wabble full build with temporary local-SDK modfile | Passed |
| Live local Wabo Business.FetchContext with Diaspora credential | Passed on v24 and v26 |
| Live local Wabo PhoneNumber.FetchAllContext using Wabble's exact six selected fields | Passed on v24 and v26; two phones decoded on each |

The temporary modfile was outside both repositories. Wabble source and its normal
dependency files were not changed. Opt-in external integration tests were not
enabled. Full live messaging, onboarding, campaign creation and attribution remain
unverified, despite successful compilation, tests and reads.

### Confirmed application gaps and execution order

1. **Resolve provider app/account provenance.** The strongest currently usable
   credential is Diaspora's matching runtime-app token, while two older accounts
   use a different app. Obtain dashboard approval evidence for the intended app
   and verify current production configuration before onboarding customers.
2. **Enforce authenticated durable ingress.** Wabble's checked-in base config has
   `whatsapp.enforce_webhook_signature=false`. In monitor mode invalid signatures
   are accepted; with no secrets the handler skips verification. Existing
   app-specific secret override support is useful but its effective production
   configuration is unknown. Wabo also ignores event publication failures.
3. **Persist attribution.** `handleTextMessage` stores text only;
   `handleIncomingMessage` does not pass Referral/RawMessage into storage.
   `handleAdInteractionEvent` and `handleUnknownEvent` are empty. Implement the
   common referral persistence path rather than relying on the ad-interaction hook.
4. **Complete promised inbound content.** Image/video/document/audio handlers
   currently update conversation/contact state but leave message persistence as
   TODOs; contact-card handling is empty. SDK model compatibility alone does not
   make these features complete in the Wabble inbox.
5. **Prevent partial-sync deletion.** `FetchPhoneNumbersFromGraphAPI` consumes
   one page and `SyncPhoneNumbersToDB` removes stored numbers outside that list.
   Implement complete paging and test multi-page accounts before claiming safe sync.
6. **Unify version and frontend configuration.** Signup backend pins v24; other
   factories inherit it. The current signup component reads `PUBLIC_META_*` while
   inspected cloud env inputs use legacy `NEXT_PUBLIC_META_*`; the inspected local
   runtime env supplies neither frontend prefix. Production Worker variables were
   not inspected. Confirm actual frontend build settings before declaring signup ready.
7. **Implement conversions and ad controls.** Granted event/ads permissions and
   readable campaigns are prerequisites, not implemented Wabo/Wabble features.
   Dataset access, supported events, Page/phone ad linkage, write authorization,
   delivery eligibility and Status/Marketing Messages availability need their own checks.

The initial exploratory inventory identified Diaspora's active ad account, but
the user's subsequent clarification selects Nextbird. Do not use another
organization's assets as a substitute or move accounts based on this inventory.

## Nextbird-specific follow-up

Target clarified and rechecked on 2026-11-01, approximately 09:58–10:00 UTC.
All requests remained read-only and database access used a read-only transaction.

| Item | Confirmed result |
| --- | --- |
| Wabble organization | Nextbird, linked to the Daniel Nextbird login |
| WABA | ID ending `1591`; readable on v24 and v26 |
| Owning business | ID ending `4889`; verification_status=`verified` |
| Issuing app | ID ending `1067` |
| Matching app credentials | Present in Wabble `dev/cloud/.env`; authenticated app read succeeds |
| Token introspection with matching app | Valid SYSTEM_USER token, matches cloud app, expiry/data-access-expiry both 0 |
| Granted scopes | catalog_management, business_management, whatsapp_business_management, whatsapp_business_messaging, whatsapp_business_manage_events, public_profile |
| Ads scopes | ads_read and ads_management absent from this token |
| Owned ad-account listing | HTTP 403, code 200; cannot conclude the business owns no ad accounts |
| Client ad-account listing | Empty result |
| Owned Pages | One returned Page, ID ending `1722`; full Page/phone linkage not verified |
| Graph phone listing | Empty on both v24 and v26 |
| Local stored phone | ID ending `4079`, Status=VERIFIED in database |
| Direct stored-phone read | HTTP 400, code 100/subcode 33: object absent or inaccessible to this token; distinguish in Meta dashboard |
| Nextbird embedded-signup sessions | Zero rows in the inspected local snapshot |
| Tech Provider/App Review status | Still requires dashboard evidence for app ending `1067`; verified business and scopes do not establish provider approval |

The previous token-introspection failure was caused by attempting the check with
the different local-runtime app's credentials. Matching cloud-app credentials
successfully introspect Nextbird's token. This resolves the app-binding question;
do not rotate or replace the existing token solely due to the earlier failure.

Nextbird launch sequence:

1. Confirm that app ending `1067` is the intended provider-approved app and identify
   the effective production configuration. Keep local-runtime app ending `9581`
   separate; finding credentials in a cloud env file does not establish deployment.
2. The user reports the old phone was lost and wants a new number. Add and verify
   a newly controlled number under the intended Nextbird WABA, then establish its
   phone/Page relationship. Preserve historical records for phone ending `4079`;
   recovery of that phone is not a prerequisite for a new-number pilot.
3. Obtain an appropriately authorized token with ads_read and ads_management for
   the intended app/use case and assign the intended ad account to its system user.
   Verify direct account and paused-campaign reads. No token changes were made here.
4. Verify existing/new onboarding with this exact app and assets, including current
   permission review/access levels for external customer businesses.
5. Complete durable ingress and attribution persistence, then conduct the separately
   authorized CTWA pilot and conversion-reporting checks using Nextbird's assets.

Current decision: the Nextbird business and WhatsApp management access are
available, but the inspected connection is **not yet ready for a messaging-plus-ads
pilot**: its stored sender is inaccessible and its token lacks advertising scopes.
This is an account/configuration readiness gap in addition to the application
implementation gaps above, not evidence of a Go SDK compilation incompatibility.

## Feem Digitals agency acquisition and partner scope

### Evidence reviewed

Reviewed the public homepage, contact page and packages page at
[Feem Digitals](https://feemdigitals.com/), and local source at
`/home/ambuche/repos/FEEM` on 2026-11-01.

- Feem positions itself around marketing, operations, lead generation, CRM and
  follow-up. The proposed WhatsApp acquisition service fits that existing scope.
- The live homepage routes consultation/package CTAs to `/contact`. Contact
  collects name, email and a free-text message; it also offers a direct WhatsApp link.
- Local `src/components/ui/Form.astro` POSTs to Web3Forms. No Nextbird ingestion,
  tenant binding, qualification or outcome tracking was found in that form path.
  Submission delivery was not exercised; no messages were sent to the agency.
- `src/config.yaml` has a null Google Analytics ID. This does not prove the deployed
  site has no independently injected analytics, but local configuration cannot
  establish a measured acquisition funnel.
- The homepage and packages page show different Starter prices. Align the offer,
  inclusions and exclusions before sending paid traffic; existing website prices
  do not establish the economics of this new service.
- Feem already has a WhatsApp marketing article describing opt-ins and follow-up.
  Use that content as a relevant acquisition asset rather than inventing social proof.
- FEEM has existing uncommitted edits across six website files. This investigation
  did not alter those files, publish changes, or overwrite ongoing work.

### The two acquisition funnels

**Agency funnel:** referral/content/paid ad -> Feem offer -> enquiry -> qualification
-> consultation -> agreed service -> client workspace -> authorized assets ->
campaign live -> first useful business outcome -> retained paying client.

**Client funnel:** client campaign or organic source -> enquiry in the client's
workspace -> response -> qualified prospect -> quotation/booking/order -> completed
sale -> permitted follow-up and repeat purchase.

Do not pool Feem's business prospects and clients' consumers in the Nextbird operator
organization. They have different ownership, access, consent and reporting scopes.
An ad click, a WhatsApp conversation, a qualified lead and a paying customer are
separate events. Neither conversation volume nor platform signup proves acquisition.

### Offer and responsibilities

Proposed entry offer: **"WhatsApp lead generation and follow-up, managed by Feem
Digitals and powered by Nextbird."** Scope one business segment, one offer and a
clear lead-to-outcome workflow for the first pilot. This is a proposed offer, not
an assertion of delivered customer results or a commitment to a new price.

| Party | Responsibility |
| --- | --- |
| Feem | Prospecting, offer, content/creative, qualification, consultations, campaign operation, client communication and outcome verification |
| Nextbird/Wabble | Agency/client access, guided onboarding, inbox, lead pipeline, attribution, follow-up tasks, campaign state and reporting |
| Wabo | Typed Meta APIs, request/error contracts, wire parsing, version compatibility and testable transport |
| Client business | Own assets, approve access and budget, provide product/service facts, fulfill sales and confirm outcomes |

Commercial structure to validate: onboarding/setup fee, managed-service fee and
software inclusion/subscription, with Meta ad spend and messaging charges stated
separately. Prefer client-paid Meta advertising for initial engagements. A partner
access relationship, Tech Provider approval and an advertising-credit arrangement
are distinct. Meta says only Solution Partners can extend WhatsApp credit lines;
do not sell an unverified credit/reseller entitlement as part of this offer.

### New Nextbird phone: required inputs and sequence

The user reports loss of the previous number; this is now a replacement-number
workflow, not a repair of that sender. Required inputs are a new number under
business control, ability to receive its SMS/voice verification code, authorized
access to the intended WABA, business display-name details and a securely stored
two-step PIN. The SMS/voice ownership code and the chosen two-step PIN are different.
Meta-account login recovery/2FA, if affected by loss of the old SIM, is a separate task.

1. Obtain the new number and establish whether it is unused on WhatsApp or already
   registered in an app. Prefer a fresh number for this pilot; existing app numbers
   require an explicitly selected supported migration/coexistence path.
2. In the intended Nextbird business/WABA, add the number and complete ownership
   verification. Do not create a replacement Meta app/business solely because a
   phone was lost; retain the approved app/portfolio relationship where applicable.
3. Register for Cloud API and set its two-step PIN through the supported flow.
   Wabo exposes request/verify-code primitives and RegisterContext; account writes
   need error/false-response checks, not merely a stored "VERIFIED" label.
4. Read the new phone back from Graph, confirm exact WABA/phone identity and the
   existing app subscription, and test the authenticated webhook path.
5. Add/sync the new phone in Wabble and switch the intended sender/primary
   selection only after checks. Preserve the old sender's historical messages and
   reporting; stop selecting it for new outbound actions. Do not rewrite history
   or silently turn historical campaigns into campaigns from the new phone.
6. Configure the Page's intended WhatsApp destination for ads and verify it with
   a controlled message. Adding a phone does not automatically establish ad linkage.
7. Keep Feem's existing public WhatsApp number unchanged until its owner explicitly
   selects the intended Feem lead-routing number. It is not the lost Nextbird number.

No new number was provided or purchased, no OTP requested and no registration
performed as part of this planning update.

### Ad ownership and onboarding other businesses

Recommended asset ownership: each client retains its Meta business portfolio,
ad account, Page/Instagram identity, WhatsApp number/WABA and conversion assets.
Feem/Nextbird receive task-specific delegated access. Nextbird uses its own
authorized assets only for Nextbird/Feem's agreed first-party acquisition campaign.

Three independent requirements must pass per client: the app is approved for the
operation, the token has the required scope, and the token's actor is assigned
the actual asset/task. A successful check of one does not establish the other two.

| Capability | Scope/access to verify | Other required asset evidence |
| --- | --- | --- |
| WABA/phone/template administration | whatsapp_business_management; business_management when required by business-asset operations | Client authorization to the specific WABA/business |
| WhatsApp sending | whatsapp_business_messaging | Correct registered phone and applicable account/template eligibility |
| Business-messaging conversion reporting | whatsapp_business_manage_events plus current endpoint-specific requirements | Authorized dataset and correct WABA/identity; live acceptance and attribution |
| Ad reporting | ads_read, or a currently documented equivalent allowed by endpoint | Access to the intended ad account and reporting tasks |
| Ad creation/control | ads_management | Campaign-management task on intended ad account and required Page/WhatsApp linkage |
| Page discovery/advertising identity | Endpoint-specific Page permissions, potentially pages_show_list, pages_read_engagement or pages_manage_ads | Correct delegated Page tasks; do not request every Page permission by default |

Meta's Marketing API collection explicitly requires Advanced Access for managing
other people's ad accounts using ads_read/ads_management. WhatsApp provider review
does not automatically approve those advertising permissions. Confirm both review
tracks for app ending `1067` before marketing self-service third-party ad creation.

Per-client onboarding sequence:

1. Identify owner, business goal, offer, service area, campaign budget approver,
   operating hours, lead qualification rule and outcome owner.
2. Create a client-owned Wabble organization. Proposed agency-client authorization
   must record agency, client, approved tasks, approving actor and revocation time;
   this is a required design, not a claim of implemented partner tenancy.
3. Use supported Meta partner asset sharing for agency operations and Embedded
   Signup for WhatsApp onboarding. Embedded Signup alone does not prove ad-account
   sharing. Verify the precise current dashboard flow when executing it.
4. Record business/WABA/phone/Page/ad-account/dataset IDs and credential provenance
   against that organization; validate granted scopes and per-asset access.
5. Confirm client billing and spending authorization. If a client needs a new ad
   account, establish ownership, currency, timezone and payment details before use;
   request authorized access to an existing client account rather than claiming it.
6. Show a readiness checklist with separate states for phone, messaging, ads reads,
   ads writes, conversions and approval. Return actionable reasons for unavailable steps.
7. Run the agreed message/campaign pilot; allow human campaign operation in Ads
   Manager while API automation is incomplete. Measure each manual task for product work.
8. On disconnect, remove future agency access, stop scheduled actions and apply
   agreed data retention/export. Client access and historical ownership remain clear.

For Nextbird today: add the new phone, obtain ads scopes through the matching app,
verify/assign its intended ad account, then check Page linkage. The previous 403
means access is denied; it is not proof that no ad account exists or that a new
one must be created.

### First product implementation across the three repositories

| Order | Repository | Concrete work | Acceptance evidence |
| --- | --- | --- | --- |
| 1 | Wabble + account setup | New Nextbird sender readiness, matching-app config, signature enforcement and durable ingress | Readback plus controlled inbound/reply; invalid signatures rejected, accepted events survive restart |
| 2 | Wabble | Tenant-scoped agency prospect intake, qualification, source and lifecycle tracking; agency/client grants | Feem test enquiry lands once in its own workspace; duplicate/retry and cross-tenant checks pass |
| 3 | FEEM | Dedicated WhatsApp acquisition offer and working Nextbird intake from form/CTA; source capture and confirmed receipt | Browser submission persists once and reaches an operator; submission errors do not show success |
| 4 | Wabble + Wabo | Common referral persistence, CTWA click identity and lead-to-outcome link | Actual ad-originating message retained; organic traffic remains separately attributable |
| 5 | Wabble | Guided client asset authorization/readiness and operating handoff | External pilot client can authorize and revoke with correct tenant isolation |
| 6 | Wabo + Wabble | Conversion reporting, ledger and diagnostics | No duplicate outcomes; Meta acceptance and reporting checked independently |
| 7 | Wabo + Wabble | Paused campaign creation, review, approved activation and insights | Correct client identity/budget; pause and ambiguous-timeout reconciliation verified |

Public forms must not receive Meta credentials or a privileged server token. The
intake API needs a server-side tenant/campaign binding, validation, rate limiting,
spam controls and idempotency. A caller-supplied organization ID alone is insufficient.
Keep public receipt separate from unsolicited automated outbound messaging.

### Acquisition channels to test

These are hypotheses, not measured CAC rankings. First test warm agency access
to validate delivery, then compare repeatable channels using the same offer and
qualification rule. A first-party enquiry is not consent for unrelated broadcasts.

| Channel | Test and intended value | Source evidence to retain |
| --- | --- | --- |
| Feem's warm relationships | Invite suitable existing clients to a scoped paid pilot; learn onboarding and fulfillment costs quickly | Referrer/account relationship, consultation, paid activation and staff time |
| Professional referrals | Work with complementary local agencies, designers and business-service partners | Partner code, referral commission, client activation and retention |
| Founder/agency-led outreach | Target one local segment with a practical follow-up audit and consultation | Outreach source and labor, responses, qualified consultations and paid clients |
| Educational content/local discovery | Use Feem's WhatsApp article and focused content pointing to a concrete audit/demo offer | UTM/source, enquiry receipt, consultation and paid-client linkage |
| Click-to-WhatsApp campaigns | Test Feem's acquisition offer once sender and durable attribution work | Spend, ctwa_clid when provided, conversation, qualification and paid outcome |
| Client-to-client referrals | Ask successful pilot clients for introductions after a verifiable useful outcome | Referred client, incentive, payment and ongoing retention |

Organic wa.me links and QR codes should use their own source mechanism and user
disclosure. Do not manufacture ctwa_clid values or report organic enquiries as paid
ad conversions. Preserve first touch and later touches; do not overwrite missing
referrals with an assumed campaign.

### Proposed acquisition measurement contract

No acquisition cost or sales baseline has been measured in this investigation.
The following definitions are proposals to instrument, not results or price targets.
Operational owner: Feem for sales/outcomes; Nextbird product owner for event integrity.

- **Agency CAC:** acquisition costs for an explicitly defined Feem prospect cohort
  divided by distinct first-time paying business clients from that cohort. Include
  Feem's acquisition ads, creative, sales labor and referral fees; define labor
  costing and follow-up cutoff before comparison. Report cash-only CAC separately
  if labor is not yet valued. Client campaign pass-through spend is not agency CAC.
- **Client customer CAC:** a separately scoped client's acquisition costs divided
  by distinct newly paying customers attributed under the declared model/window.
  Report media-only CPA separately from fully loaded CAC including allocated
  service/creative/software costs. Repeat orders are not new customers.
- **Activation-to-value rate:** among onboarded paying client businesses in a
  stated cohort/window, the share reaching the agreed first useful outcome, such
  as a qualified lead or completed booking. Publish the actual milestone; do not
  relabel a conversation as a sale or a connected account as value delivered.

Drivers: qualified-enquiry-to-paid-client conversion and time from signed agreement
to first useful client outcome. Guardrails: retained paying clients and contribution
after delivery/servicing costs; cheap acquisition is insufficient if clients churn
or agency fulfillment consumes the margin. A qualified enquiry must meet the
agreed need/service-area/decision-maker rule before its status changes.

Proposed events/fields: partner_id, organization_id, prospect_id, source/UTMs,
referrer_id, received_at, qualified_at/reason, consultation_at, agreement_at,
onboarding_started/completed, asset_readiness, campaign_launched_at,
first_value_at/type, first_payment_at, retained/cancelled_at, acquisition cost
category and currency. Consumer outcome records belong to each client tenant.

Review pipeline and operational failures weekly. Compare CAC only for sufficiently
mature cohorts, with the same cost definition and attribution/follow-up window.
Use paid-client counts and open pipeline beside CAC; zero new clients means CAC
is undefined, not zero. Establish a baseline before fixing channel budgets or
asserting a channel is cheaper. Margin, budget and retention inputs are still missing.

### Sources and remaining decisions

- [Feem homepage](https://feemdigitals.com/), [contact](https://feemdigitals.com/contact),
  [packages](https://feemdigitals.com/Pricing); inspected source paths above.
- [Meta registration reference](https://www.postman.com/meta/whatsapp-business-platform/folder/zuoeksl/registration):
  ownership verification, registration and two-step verification.
- [Meta Marketing API permissions](https://www.postman.com/meta/facebook-marketing-api/collection/0zr4mes/facebook-marketing-api-mapi):
  own-account versus other-business Advanced Access requirements.
- [WhatsApp partner program](https://whatsappbusiness.com/partners/become-a-partner/):
  Tech Provider/Tech Partner/Solution Partner distinctions and credit-line restriction.
- Current Meta partner-sharing help pages redirected to login/temporary-block
  screens. Exact dashboard steps were not independently verified in this update.

Remaining business decisions: the new number and its registration state, who owns
Feem's agency portfolio and approved access, the first client segment, pilot offer
and budget, how Feem/Nextbird split service/software revenue, and the exact paid
activation/outcome definition. None prevents preserving this scope or beginning
the non-live implementation work; number registration and ad activation depend
on the missing concrete inputs and authorization.
