# Kaivo Protobuf contracts

This repository maintains the `.proto` files shared by Kaivo services. The user-facing HTTP/EPay Swagger documentation is maintained separately. Each service uses the generated SDK instead of maintaining its own protocol copy.

```text
proto/aipay/
  authz/v1/policy.proto            Method authorization option (aipay.authz.v1.policy)
  common/v1/context.proto          Trace information and platform, organization and business entity scopes
  common/v1/money.proto            Money, exchange rates and conventions for identifiers, time and amounts
  common/v1/asset.proto            Currency table and registry of chains and assets (TRON/BSC USDT, TRX, BNB)
  common/v1/trade.proto            Full C2C quote and immutable order snapshot
  events/v1/event_stream.proto     Signed event envelope and EventStreamService pull interface
  approval/v1/approval.proto       Approval requests and WebAuthn assertions
  iam/v1/identity.proto            Authentication inputs, principals and short-lived identity assertions
  iam/v1/authorization.proto       Roles, permissions, bindings, authorization checks and data scope constraints
  collection/v1/collection_service.proto  CollectionService creates, queries and cancels restricted sweep jobs
  upstream/v1/upstream_service.proto  Upstream gateway business and management methods
  payment/v1/payment.proto         Payment order contract placeholder
  payment/v1/cashier_service.proto  Token-based EPay cashier lookup
  payment/v1/risk_extension.proto  Exceptional risk extension requests, eligibility and platform approvals
  payment/v1/amount_mismatch.proto Amount mismatch cases and phase 1 recovery approvals
  ledger/v1/ledger.proto           Journal transactions, journal lines and CheckpointSigned checkpoints
  ledger/v1/journal_service.proto  JournalService retrieves journal transactions and checkpoints
  ledger/v1/adjustment_service.proto  AdjustmentService submits evidence files and approves manual adjustments
  ledger/v1/approval_service.proto    AdjustmentQueryService reads adjustments and evidence, and LedgerApprovalService serves the approval inbox
  ledger/v1/trade.proto            Escrow state, refund branches and reservation records
  ledger/v1/reservation_service.proto  ReservationService manages escrow freezes, barriers and refund reservations
  ledger/v1/trade_events.proto     Four types of ledger facts on ledger.trades
  ledger/v1/control_service.proto  LedgerControlService approvals for read-only exit, carryover and USD rate policy
                                   GetReadOnlyStatus and GetPeriodCarryovers for exit requests and period-end suspense balances
  settlement/v1/settlement.proto   Settlement contract placeholder
  reconciliation/v1/reconciliation.proto  Reconciliation contract placeholder
  reconciliation/v1/trade_events.proto  Deposit verification and release, cancellation and refund authorizations
gen/go/                           Generated Go / gRPC code
testvectors/                      Canonical encoding and signature test vectors (data only)
internal/policycheck/             Method policy checks for make check
```

Enum value options (`currency_info`, `chain_info`, `asset_info`) register currencies and on-chain assets. All services read the same table from the generated SDK. Adding a currency, chain or contract requires changes to this repository and to each service's tests.

The IAM contract has a single entry point for authentication and authorization, `AuthorizationService.Authorize`. The gateway calls it once per request to perform both checks. Endpoints that only need to verify identity also use `Authorize`, with a permission that represents the caller. Downstream services verify the assertion themselves without calling IAM again (decision A78). This interface definition does not indicate whether the service is running.

The account model (decisions A83 and A84) defines the `platform`, `account`, `business` and `application` authorization scopes in `common/v1/context.proto`. Funds belong only to `account_id`, which identifies a personal or organization account. Business scope adds `business_id` to the owning account, and application scope also adds `application_id`. Neither grants account-wide funds permissions. Organization accounts may include `org_id`. Personal accounts must omit it. The enum values and field numbers for the old organization and business entity scopes are reserved. Messages from older versions fail validation. The IAM repository manages identity credentials, organization membership and role policies. The ledger contract covers journal retrieval, checkpoints, manual adjustments, C2C escrow and operational controls. The reconciliation contract defines deposit verification and escrow authorization events. Business RPCs for payment and settlement are still placeholders. Add only fields defined in the docs repository's [cross-service contract design](https://github.com/Kaivodev/docs/blob/main/design/contracts.md).

Only payment can issue C2C commands. Reconciliation can also query freezes. Only a signed authorization from reconciliation can finalize a release, cancellation or refund. Payment has no RPC to release funds unilaterally. Each change binds the original snapshot and control version. Each authorization ID can be consumed only once. Only full refunds are supported for now. Partial refunds are pending D18. Ledger publishes `ledger.trades` facts in the same transaction as the journal entries. The payload comments specify the stream name, event type and `schema_version`. Operational controls are accessible only through the backend management listener. Ledger must also check platform scope, request idempotency and independent WebAuthn approval. The contract and generated SDK do not show whether those runtime checks are implemented.

## Method authorization policies

Each RPC declares caller rules with `(aipay.authz.v1.policy)`. Each rule specifies the allowed mTLS service names, the identity assertion requirement, the acceptable permissions and the maximum age of recent authentication. The assertion levels are no assertion, required assertion, required assertion with MFA and recent authentication, and single use. Each level includes the checks from the preceding levels. Requirements can vary by caller on the same method. For example, backend can supply an assertion while settlement calls with its service identity. `make check` inspects the image from `buf build` and fails if any method lacks a valid policy. The servicekit server interceptor enforces the same policy at runtime. Method policies on `EventStreamService` list all possible receivers. Each producer configures stream-level authorization from its stream list.

## Signed events, approvals and test vectors

The comments in `events/v1/event_stream.proto` specify the signing input, digest and signature format for signed events. The domain tag `AIPAY-EVENT-V1` has no length prefix. Strings and byte strings have a 4-byte big-endian length prefix. Integers use 8-byte big-endian encoding. Signatures use ECDSA P-256 with ASN.1 DER encoding. The approval digest is `SHA-256("AIPAY-APPROVAL-V1" || request_bytes)`.

The `testvectors` package publishes JSON test vectors for canonical journal encoding V1, balances digests, trial balance digests, RFC 6962 Merkle roots and checkpoint signatures, signed events, and WebAuthn approval assertions. The vectors include inputs, canonical bytes, expected hashes, public keys and signatures, and tampered cases that verifiers must reject. The package contains only data and provides no encoder or verifier. The settlement and reconciliation services each implement journal hash verification from the specification (decision A16) and test it against these vectors. The keys were created solely to generate the vectors. The private keys were discarded.

IAM already implements assertion issuance for Telegram and EPay, among others. The available login sources, key custody and revocation capabilities depend on what IAM has delivered. The protocol alone does not establish that the platform management path is connected. EPay assertions must bind to the current request and the operation being authorized. They must never serve as reusable bearer credentials. Resource services must use the `allowed` decision together with the returned data scope constraints and check the object's actual ownership and business state.

## v0.29.0 EPay cashier (A153, unreleased)

- Backend sends the cashier token’s SHA-256 on `CreateOrder` for page entries and, with the JSON cashier switch enabled, for JSON entries. Payment excludes the hash from idempotency checks and registers it on the original order for an identical retry. Tokens expire after 24 hours, and registering more than 20 live tokens per order revokes the oldest.
- Only backend calls the read-only `CashierService.GetCashier`, using its service identity and the raw token. Buyers see the cashier state, applicable expiry, original CNY amount and payment method. Only `READY` includes a payment instruction.
- After verified payment, backend can append the signed merchant notification to the snapshot’s return URL as its query. Payment supplies this target only while the application’s current approved, enabled revision still has the same return origin.

## v0.28.0 exceptional risk extensions and amount mismatch recovery, phase 1 (A150, A151, unreleased)

- A platform user or the original seller can request more time on a paid, unreleased trade. Payment places a time-bounded hold on ledger before opening the approval. While the hold is recorded, release authorizations must take effect at or after `pending_until`. Two distinct platform approvers other than the initiator must sign to move the risk deadline later. A seller request also excludes all members of the seller account from approval.
- Platform users can request re-verification of an amount mismatch or a manual refund of the buyer's observed payment. Both outcomes require two independent approvals. Re-verification resumes queries under the existing evidence rules. For a refund, ledger must confirm the reservation before the seller receives the refund notice. Reconciliation independently verifies both approvals returned by `GetSellerRefundReviewForVerification` and matches the recovery's refund ID, trade, escrow and approved fiat amount to the seller review.
- Platform approval responses expose only `credential_id` in assertions. Backend routes `SellerRiskExtensionNotice` to the upstream bot and sends `MerchantRiskExtensionNotice` to the original recipient on application of the extension. Only backend consumes these notices.
- `payment_risk_extension_v1.json` and `payment_amount_mismatch_v1.json` publish canonical bytes and approval digests for both origins and both recovery outcomes. Go consumers can read them through `testvectors.PaymentRiskExtensionV1()` and `testvectors.PaymentAmountMismatchV1()`.

## v0.27.0 merchant and upstream Telegram bots and seller notices (A149, unreleased)

- Backend sets `TelegramCredential.bot` from the verified webhook route. IAM shares one Telegram-to-user link across both bots and scopes sessions, updates and group bindings to each bot. Deploy IAM before enabling the upstream webhook. During rollout, IAM treats an omitted credential or notification bot selector as `MERCHANT`.
- The merchant bot supports self-registration. Sellers bind the upstream bot with a single-use code from their upstream portal session after recent TOTP authentication. The portal can check `GetBotBinding` and unbind with the user's active TOTP. Unbinding removes the global link only when its last bot binding is gone.
- Payment publishes `SellerOrderAssigned`, `SellerInventoryLow` and `SellerDisputeNotice` on `payment.account_notices`. Backend delivers all seller notices through the upstream bot and merchant notices through the merchant bot. Funds services must not consume this stream.
- Since UG-5, payment and reconciliation use their respective gateways for ABI v3 Rust components and refuse to start against databases containing ABI v2 releases. Uploads allow up to 4 MiB per component, 3 MiB of fixtures and 16 MiB for the whole request.

## v0.26.0 dispute payments and approval assertion reads (A147, A148, unreleased)

- Reconciliation calls `DisputeService.ReportDisputeRulingReview` after it denies or quarantines a pre-release ruling to release funds or return them to the seller. Under its issuance lock, it checks that no authorization exists for the order and control version and records a fence against issuing one carrying this ruling at that version. Payment can then let the ruling lapse immediately. The report never moves funds or grants authorization.
- `PlatformDisputeView.payment_observation` tracks a buyer payment observed during the dispute. Payment writes `paid_verified_at` and emits `OrderPaid` only once a return-to-seller cancellation can no longer execute. If ledger cancels first, `LatePayment.dispute_id` links the A139 late payment to the dispute. `PlatformDisputeView.ruling_review` shows the recorded review of the displayed ruling. Sellers see only `SellerDisputeView.payment_state` for the observation.
- Owners store complete original WebAuthn assertions. Only reconciliation receives them through payment's `GetDisputeRuling` and `GetSellerRefundReviewForVerification`. In every platform-facing approval record, including `Approve*` responses, the assertion contains only `credential_id`. Approver identity, sign count, verification time and request digest stay unchanged. `ListApprovals` returns headers only.
- Deploy payment before reconciliation starts reporting reviews. Reconciliation treats `UNIMPLEMENTED` as retryable. Deploy backend after payment to consume the new view fields.

## v0.25.0 ledger adjustment reads and unified ledger approvals (unreleased)

- `SubmitAdjustment` accepts one to five PDF, PNG or JPEG files. The ledger recomputes the evidence manifest digest from each file's name, content type, size and SHA-256 in submission order. Approvers sign that digest through the existing request encoding. Each line encodes debit as ASCII `'D'` (0x44) and credit as ASCII `'C'` (0x43). `testvectors.LedgerAdjustmentV1()` provides the files, manifest digests, request bytes and approval digests.
- Submission validates each line. The final approval opens eligible platform accounts and posts the entry in one transaction, in the period and business day open at posting. A posting failure leaves the request pending and rolls back the final approval.
- With `ledger.adjustment.read`, backend can call `ListAdjustments` across statuses, inspect line accounts and journal entries with `GetAdjustment`, and download a numbered file with `GetAdjustmentEvidenceFile`. The reads work in read-only mode and honor the target funds account in the verified proof.
- With `ledger.approval.read`, backend can call `LedgerApprovalService.ListApprovals` for pending adjustment, read-only exit, carryover and USD rate policy requests. In v0.25.0, `GetApproval` returned any state with the structured material and complete recorded assertions.
- Deploy ledger before backend submits evidence files. An older ledger ignores request field 6. Callers can detect this from the missing `evidence_files` echo in the submission response.

## v0.24.0 ledger control permissions and reads (unreleased)

- Read-only exit and carryover now have separate read, submit and approve permissions under `ledger.read_only_exit.*` and `ledger.carryover.*`. These replace the shared `ledger.control.request` and `ledger.control.approve` keys. IAM never registered either key, and the four write RPCs had no callers. Existing wire formats and signed `request_bytes` stay the same.
- `GetReadOnlyStatus` returns the ledger mode and up to 20 pending, unexpired exit requests from the current read-only episode, newest first. Approvers can inspect the request bytes, digest, snapshot and recorded approvals before signing.
- `GetPeriodCarryovers` returns nonzero suspense balances and the latest 200 carryover requests across all statuses for the requested period. Balances are to date while the period is running. Each balance identifies the executed carryover that satisfies the period-close suspense check, if one exists.

## v0.23.0 payment instruction lifetime, late-payment window and USD rate policy (A130, unreleased)

- Payment channel approvals can set the buyer instruction lifetime for new dispatches. An absent value preserves earlier approval and web-command bytes and uses the deployment lifetime. The deployment ceiling also caps expiry supplied by a plugin or gateway.
- Reconciliation pins the late-payment window at the first cancellation observation and includes its version in the signed authorization. Version zero preserves the original payload bytes. Receivers that reject unknown fields must upgrade before reconciliation emits a nonzero version. Backend can submit, approve, read and list window versions.
- Ledger exposes the current USD rate deviation policy, its effective threshold and any pending request. Two distinct approvers other than the requester must approve a change. The deployment ceiling limits the effective threshold and applies before the first version is approved.

## v0.22.0 platform forced close (A145, unreleased)

- A platform operator can request a forced close for an unpaid order whose upstream has no close API. Payment records the initiating operator as `requested_by` and binds the trade, escrow reservation and control version, reason, order status, upstream order ID, `never_sent` and `eligible_at` in `OrderForcedCloseMaterial`. Two other platform users must sign the `forced_close` operation with WebAuthn. All three must be distinct natural persons. Payment rechecks the material under the order lock at each approval.
- On the second signature, payment registers the close intent and emits the signed `OrderCancellationVerificationRequested` event at schema version 1. Reconciliation still needs two independent unpaid observations separated by the late-payment window. With `OPERATOR_FORCED_CLOSE` (2), payment has no upstream order ID and never delivered an instruction. Reconciliation may count its own not-found results, as for A144. With `OPERATOR_FORCED_CLOSE_BOUND` (3), each observation must show the bound upstream order closed. A not-found result stays inconclusive.
- Only backend calls `OrderForcedCloseService`. `GetForcedClose` requires an assertion with `payment.order.force_close.read`. `SubmitForcedClose` requires `payment.order.force_close.submit` and MFA with an authentication age of at most 600 seconds. `ApproveForcedClose` requires an assertion with `payment.order.force_close.approve` and the approver's WebAuthn assertion.
- If payment later discovers a payment on an order cancelled this way, `LatePayment.forced_close_approval_request_id` identifies the approved request. `testvectors.PaymentForcedCloseV1()` publishes canonical approval bytes and digests for a never-sent unbound order and a bound order after instruction expiry.
- As for A144, deploy reconciliation before payment emits basis 2 or 3. Older reconciliation versions reject unknown basis values. [Platform forced close design](https://github.com/Kaivodev/docs/blob/main/design/forced-close.md)

## v0.20.0 customer fund coverage attestation (A141, unreleased)

- Operators submit daily TRON USDT observations for registered platform addresses. Reconciliation pages through ledger’s read-only `GetLiabilitySnapshot` at one journal sequence, verifies the balances digest, and classifies the accounts itself. Unclassified positive USDT credit balances count toward user liabilities.
- Reconciliation deducts window outflows and unrecorded inflows from observed assets, caps the result at the ledger wallet balance, then deducts unmatched receipts and user liabilities. Two distinct platform approvers sign the material with WebAuthn. The submitter may be one approver. Approval requests expire 24 hours after submission, and an attestation cannot be approved once its own expiry has passed.
- Backend can submit, approve, revoke and inspect attestations, read coverage status and list active addresses. Before issuing a release, cancellation or refund, reconciliation re-verifies the newest approved attestation. Low margin permits those issuances. Revenue transfers require the full buffer. Expiry or revocation closes both gates until a new approval.
- `testvectors.ReconciliationCoverageV1()` publishes canonical approval bytes and digests for all three results, UTF-8 evidence digests, address registry versions and a WebAuthn revocation challenge. [Customer fund coverage design](https://github.com/Kaivodev/docs/blob/main/design/coverage-attestation.md)

## v0.19.0 seller manual refund (A139, released)

- Full refunds reserve ledger funds and enter `AWAITING_SELLER`. The seller refunds the buyer in the upstream back office and submits versioned JPEG evidence for two platform WebAuthn approvers. Approval starts the objection window (24 hours by default). Rejection or a timely objection reopens submission. A separate two-person approval can declare failure.
- Payment exposes refund origin, seller and objection deadlines, and the approved upstream refund time. The seller portal accepts evidence and serves the seller's own screenshots. Platform operators can initiate refunds, inspect late payments and review submissions, duplicates, objections and failure requests. Late payments on cancelled orders close inside payment.
- Backend calls `ListSellerLatePayments` and `GetSellerLatePayment` with a required seller assertion and `payment.refund.read`. Both return `SellerLatePaymentView` scoped to the seller's own account, with current evidence when available. The list supports order, status and channel account filters and requires a page limit of 1–100.
- Reconciliation retrieves the approval that produced the terminal fact and independently verifies both assertions against its own registry, the signed material, the payment event and the ledger occupation. The A139 comments document the manual mapping in `RefundSucceeded` and `RefundFailed` v2 and the existing `TradeRefundAuthorized` and ledger receiver rules. Under A136, verification plugins must declare `query_payment` and may declare `query_refund`.
- Backend alone receives `SellerRefundNotice` and `MerchantRefundNotice` for bot delivery. The test vector package publishes canonical evidence and approval bytes in `PaymentSellerRefundV1()` and both recent-MFA command digests in `PaymentWebCommandV1()`. [Seller manual refund design](https://github.com/Kaivodev/docs/blob/main/design/manual-refund.md)

## v0.18.0 upstream gateway integration for the owning services (released)

In gateway mode, backend registers trade credentials through payment's `StoreChannelCredentials` and receives an immutable reference and a content digest. In both modes, `StoreVerificationCredentials` on reconciliation returns a content digest. In gateway mode, configuration submissions for trade channels and verification accounts must bind the registered digest. Embedded mode rejects this field. Omitting the field allows historical approval material to be verified against its original bytes. Gateway mode accepts only ABI v3 component packages. The gateway in the same trust domain verifies them again through `ReplayPackage`. Embedded mode continues to use ABI v2. See the [upstream gateway integration contract for the owning services](https://github.com/Kaivodev/docs/blob/main/design/contracts.md#v0180-属主接入上游网关).

The order creation contracts `CreateOrderRequest` and `CreatePaymentInput` add the optional fields `client_ip` (buyer IP address) and `device` (device type). Once UG-6 and UG-7 are implemented, the gateway will pass these fields through as the upstream requires.

## v0.17.0 upstream gateway contract (released, commit `8442a37`)

The trade instance accepts calls only from payment, which can call seven business methods and five management methods. The verification instance accepts calls only from reconciliation, which can call `QueryPayment`, `QueryRefund` and the five management methods. Each method policy lists the callers allowed across both instances. The gateway narrows access for each method using its startup role and the peer identity verified by mTLS. See the [upstream gateway contract](https://github.com/Kaivodev/docs/blob/main/design/upstream-gateway-contract.md) for details on idempotency, dispatch results, evidence and management methods.

## v0.15.1 two write operations require recent MFA

The method policies for `SetSellerChannelOnline` and `RequeueArchivedWork` now require `ASSERTION_REQUIREMENT_MFA_RECENT` in place of `ASSERTION_REQUIREMENT_REQUIRED`. As with `DisablePlugin`, `recent_auth_max_age_seconds` is 600. IAM requires recent MFA before issuing assertions for `payment.channel.online` and `payment.work.requeue`. Payment checks the authentication time in the assertion on receipt. The messages and fields are unchanged.

## v0.15.0 seller online status, archived task recovery, account notices and merchant statement files

- In payment, `SetSellerChannelOnline` uses the seller's own version to change whether the seller accepts orders. The seller view keeps `online`, and the platform view uses `seller_online`. Both return `seller_online_version` and omit `seller_online_changed_at` if the seller has never switched. A request with a new idempotency key still checks the version when it asks for the current value. If the version matches, payment audits the no-op and returns the current view without changing the state or version. Routing a new order requires both an enabled channel on the platform and an online seller. Orders already in progress continue.
- Platform operators and administrators can use `ListArchivedWork` to list archived failures, sorted by `archived_at` from newest to oldest, then by `kind` and `object_id`. `RequeueArchivedWork` identifies the task by `(kind, object_id)` and requeues it with a reason after checking its version. `last_error_code` uses a fixed code registered by payment, defaults to `internal` for unclassified errors and is retained after requeue. The handler rereads the business state. If an operation may already have been dispatched, the handler resolves it by querying the order. Both seller status changes and task requeues verify `command_sha256` in the assertion. See `testvectors.PaymentWebCommandV1()` for the canonical bytes and digests.
- Payment publishes `payment.account_notices`. Only backend pulls the stream to send bot notices for payments received, funds available and payment method review results. The payment time comes from `OrderPaid.paid_verified_at`. The time funds became available comes from `occurred_at` in the `TradeReleased` envelope. Review results identify the item reviewed by `payment_access_id` and `revision_version`. Funds services must never use notices as evidence of funds. At delivery, backend calls IAM's `ListAccountNotificationRecipients` to get the account owners' currently active private chats and confirmed group chats.
- The ledger `ReportService` lists business days with a generated account-level file. Business and application scopes list the same dates. The first Get for a given scope generates its file. Get returns the ZIP, the SHA-256 of the signed manifest and the daily closing checkpoint ID. For backend calls, the account comes from the verified assertion. A backend Get request that includes `account_id` returns `INVALID_ARGUMENT`. After verifying an active email subscription, IAM must pass `account_id` and may request only ACCOUNT scope. For each IAM call, ledger audits the account and business day.

See the [v0.15.0 cross-service contracts](https://github.com/Kaivodev/docs/blob/main/design/contracts.md#v0150-卖方上线归档任务恢复账户通知与商户对账文件) for method permissions, signed events and file scopes.

## v0.12.0 cancellation and refund verification triggers (unreleased)

- Once payment accepts an order close and confirms that the upstream request was sent, it publishes `OrderCancellationVerificationRequested` v1 on `payment.orders`. The payload contains only the order ID, escrow reservation ID and control version. For that order and version, reconciliation starts the first independent order query. It uses the late-payment window and second-query rules from v0.11.0.
- Full refunds before and after release both use `RefundSucceeded` / `RefundFailed` v2. Reconciliation verifies the refund by refund ID and reservation control version. The original reservation branch identifies the source of funds. To maintain stream continuity, ledger recognizes the cancellation trigger. Posting a cancellation or refund still requires a single-use signed authorization from reconciliation.

See the [v0.12.0 cross-service contracts](https://github.com/Kaivodev/docs/blob/main/design/contracts.md#v0120-取消与退款的首次核验) for when events are emitted, their idempotency keys and the order in which services integrate.

## v0.11.0 contract additions

- Only reconciliation can call ledger's `GetTradeFreezeDetails`. It reads the trade freeze, all refund and dispute holds, and the risk freezes on the original seller and recipient accounts from a single ledger snapshot.
- To release funds, reconciliation uses a fresh, independent order query result obtained no earlier than payment's verification time. Cancelling an unpaid order requires two independent checks that the order is closed and no payment was received. Reconciliation binds both results to the existing signed authorization digest.
- The backend service passes `client_ip` from a trusted source. IAM enforces the application's allowlist for all three EPay credential branches. Web sessions can record the address.
- Only backend can call payment's `GetCallbackPolicy` to retrieve per-channel restrictions on incoming callbacks. The existing two-person approval for channel configuration covers callback source CIDR ranges.
- IAM adds the read-only `GetChallenge` method, recovery code login for the seller portal, and two-step TOTP replacement. The v0.11.0 section of `design/contracts.md` in the docs repository lists the full method policies and service responsibilities.

## Minimal Collection integration contract (unreleased)

`aipay.collection.v1.DepositAddressService` defines two account address methods, GetOrCreateDepositAddress(request_id, network_id, asset_id) and GetAddressAllocation(oneof allocation_id/request_id). Only backend may call them. Both require USER/account scope, a REQUIRED assertion and audience=collection. Their respective permissions are `collection.address.ensure/read`. Neither a collection management role nor funds approval is required. The account comes from the verified assertion. Callers cannot specify a target, deployment bytes or signing parameters.

Address requests are idempotent by calling service and canonical UUIDv7 request ID. The binding, original request, subscription intent and audit record commit in one transaction. Each query rechecks the scope. Until the approved deployment, route and subscription coverage have been verified, the response returns ALLOCATING, no address and deposit_enabled=false. An existing address's lifecycle is tracked separately from its current availability. Generating the protocol alone does not establish that a subscription is live, a backend endpoint is connected or production deployment is complete. The current implementation exposes only these two address methods. The three sweep methods below have contracts but no registered runtime endpoints.

`aipay.collection.v1.CollectionService` defines only CreateSweepJob/GetSweepJob/CancelSweepJob. Only backend may call them, with platform USER/WebAuthn, a single target account and audience=collection. The permissions are collection.sweep.create/read/cancel. Writes use MFA_RECENT with a 600-second limit. Reads use REQUIRED. None uses SINGLE_USE. All three requests bind the full method name and canonical command digest. The idempotency key is separate from the trace ID. Creation acknowledges acceptance only. Cancellation applies only to queued jobs. Method options alone do not validate the entry point, ownership, digest or transaction.

`PlatformRole` adds `PLATFORM_ROLE_COLLECTION_OPERATOR=9` for IAM's protected `collection-operator` role and retains the existing values 0–8. A second platform administrator must approve each invitation. An ordinary operator or funds approver role cannot substitute for the protected role. The enum itself grants no permissions and does not establish that invitations or revocations are implemented. Older consumers must upgrade the SDK to support this role. They must reject unknown values without falling back to another role.

Collection retains UUPS/Safe and its own handleOps. Sweeps on a fixed, approved route require no per-job two-person funds approval. Governance actions still require Safe approval. The contract exposes no interfaces for address monitoring, deposit receipt or posting, List/Preview/Retry/Tick, pausing operations or signing, or dispatch from settlement. It provides no generic Any/map/Struct payloads or raw signing capability.

`testvectors.CollectionCommandV1()` publishes four sets of canonical bytes and digests. They cover UTF-8 byte lengths, the maximum uint256 amount and different oneof query branches with the same value. Consumers must implement this encoding independently and cannot substitute protobuf wire/JSON encoding. The docs repository's [collection contract](https://github.com/Kaivodev/docs/blob/main/design/contracts.md#collection-最小归集接口) lists all fields, the permission catalog and the four target HTTP management routes. This work delivers only the local protocol and SDK. There is no release or running service. The shared signature verification fields, actual platform IAM and collection/backend integration require separate acceptance checks.

## v0.9.0 portal contract additions

- `payment/v1/payment_admin_service.proto` provides platform queries, pending approval lists and details, and emergency plugin deactivation.
- `payment/v1/seller_portal_service.proto` and `portal_views.proto` provide the seller's own query views and immutable, append-only evidence submissions. They expose no refund processing or channel online/offline commands.
- `ledger/v1/query_service.proto` gives payment a batch lookup of seller inventory for reference. For backend, it provides platform and self-service queries for balances, posted deposits and escrow freezes. The interface does not expose journal entries.
- `iam/v1/notification_signing_service.proto` lets only payment request V1 success notification signatures for historical orders.
- `payment/v1/plugin_control.proto` and `reconciliation/v1/plugin_control.proto` provide plugin-level emergency deactivation using CAS/WebAuthn. ReconciliationAdminService also adds a pending approval list. Reactivation still requires a new two-person approval to enable the plugin.

The new lists use opaque cursors, 1–100 items per page and a consistent point in time for reads. The limit option in `common/v1/query.proto` declares the contract. The owning service must enforce it at runtime. The seller's identity comes only from the IAM assertion binding. Requests do not accept a seller-supplied account. The v0.9.0 section of `design/contracts.md` in the docs repository covers permissions, scopes, signature specifications, individual portal mappings and later versions. The contract and SDK alone do not establish that services or HTTP endpoints are connected.

`make breaking` compares against the local `v0.17.0` tag by default.

## Generation and checks

Go 1.27.x and the Buf CLI are required. The current verification baseline is Buf 1.57.2. `make tools` installs the pinned Go plugins in the repository's `.bin/` without modifying system tools.

```sh
go install github.com/bufbuild/buf/cmd/buf@v1.57.2
make tools
make fmt
make generate
make check
```

Generated files are under version control. Never edit them by hand. CI checks formatting, lint, method policies, compilation, test vectors and consistency between the source and generated code. For PRs, CI also checks for breaking changes against the baseline commit.

Never reuse a published field number. Reserve both the number and name when deleting a field. The zero value of each new enum must be UNSPECIFIED. Use a new versioned package for changes that alter meaning or break compatibility. Never silently redefine v1. See the [Protobuf language specification](https://protobuf.dev/programming-guides/proto3/) and [Buf generation configuration](https://buf.build/docs/configuration/v2/buf-gen-yaml/).

## Releases and local integration

The Go module is `github.com/Kaivodev/protobuf`. The compatibility baseline for v0.15.0 is v0.14.0. Each service pins its dependency version after a new release. A workspace `go.work` can support joint development. Each repository must still pass its own `GOWORK=off` checks. Never add an absolute-path `replace` to `go.mod`.

## Naming and compatibility

The project name is **Kaivo** and the GitHub organization is **Kaivodev**. The Go module has moved to `github.com/Kaivodev/protobuf`. Use the same capitalization in `GOPRIVATE` (`github.com/Kaivodev/*`). Protocol and signature identifiers and existing configuration keys retain their values for compatibility. The rename leaves the historical ledger and protocol semantics intact. See the [naming conventions](https://github.com/Kaivodev/docs/blob/main/design/naming.md) for the scope and migration rules.
