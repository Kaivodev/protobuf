# Kaivo Protobuf 契约

本仓库集中维护 Kaivo 服务间的 `.proto`，与用户侧 HTTP/EPay 的 Swagger 文档分开管理。各服务引用生成 SDK，不维护协议副本。

```text
proto/aipay/
  authz/v1/policy.proto            方法授权选项 (aipay.authz.v1.policy)
  common/v1/context.proto          跟踪信息、平台/组织/业务主体作用域
  common/v1/money.proto            Money、汇率；标识、时间与金额约定
  common/v1/asset.proto            币种表、链与资产登记（TRON/BSC USDT、TRX、BNB）
  common/v1/trade.proto            C2C 完整报价与不可变订单快照
  events/v1/event_stream.proto     签名事件信封与 EventStreamService 拉取接口
  approval/v1/approval.proto       审批请求与 WebAuthn 断言
  iam/v1/identity.proto            认证输入、主体、短期认证证明
  iam/v1/authorization.proto       角色、权限、绑定、授权检查与数据范围约束
  collection/v1/collection_service.proto  CollectionService：受限归集创建、查询、取消
  upstream/v1/upstream_service.proto  上游网关业务与管理方法
  payment/v1/payment.proto         支付订单契约占位
  ledger/v1/ledger.proto           分录交易、分录行与 CheckpointSigned 检查点
  ledger/v1/journal_service.proto  JournalService：分录与检查点拉取
  ledger/v1/adjustment_service.proto  AdjustmentService：人工调账申请与审批
  ledger/v1/trade.proto            担保状态、退款分支与预留凭证
  ledger/v1/reservation_service.proto  ReservationService：担保冻结、屏障与退款预留
  ledger/v1/trade_events.proto     ledger.trades 的四类账本事实
  ledger/v1/control_service.proto  LedgerControlService：只读退出与结转说明审批
  settlement/v1/settlement.proto   结算契约占位
  reconciliation/v1/reconciliation.proto  对账契约占位
  reconciliation/v1/trade_events.proto  预存核实、释放/取消/退款授权
gen/go/                           Go / gRPC 生成代码
testvectors/                      规范编码与签名测试向量（只含数据）
internal/policycheck/             make check 的方法策略检查
```

币种与链上资产以枚举值选项登记（`currency_info`、`chain_info`、`asset_info`），各服务从生成 SDK 读取同一张表；新增币种、链或合约需同时修改本仓库与各服务测试。

IAM 契约只有一个认证授权入口 `AuthorizationService.Authorize`：网关每个请求只调用一次，同时完成认证与授权；只需确认身份的接口使用表示“本人”的权限，同样走 `Authorize`。下游服务自行核验认证证明，不再回调 IAM（决策 A78）。这仅是接口定义，不表示服务已运行。

授权作用域按账户模型（决策 A83、A84）分为 `platform`、`account`、`business`、`application`（`common/v1/context.proto`）。资金只归属 `account_id`（个人或组织账户）；业务、应用作用域依次带上 `business_id`、`application_id` 及其所属账户，但不隐含账户级资金权限。组织账户可附 `org_id`，个人账户不填。旧的组织、业务主体作用域使用过的枚举值与字段号已保留，不再复用，旧版本发出的消息会校验失败。身份凭据、组织归属、角色策略在 IAM 仓库管理。ledger 契约包括分录拉取、检查点、人工调账、C2C 担保与运行控制；reconciliation 定义预存核实和担保授权事件。payment、settlement 的业务 RPC 仍为占位。字段按 docs 仓库的[跨服务契约设计](https://github.com/Kaivodev/docs/blob/main/design/contracts.md)维护，设计之外不新增。

C2C 命令仅 payment 服务可调用，冻结查询另允许 reconciliation。放币、取消与退款终态只由 reconciliation 签名授权驱动，payment 无单方放币 RPC；每次变化绑定原快照和控制版本，授权 ID 单次消费。退款暂只支持全额，部分退款待 D18。`ledger.trades` 事实与分录同事务发布，流名/类型及 schema_version 在载荷注释中登记。运行控制只经 backend 管理监听，ledger 还须检查平台作用域、请求幂等与独立 WebAuthn 审批；契约与生成 SDK 不代表这些运行时校验已实现。

## 方法授权策略

每个 RPC 用 `(aipay.authz.v1.policy)` 声明一组调用方规则：允许的 mTLS 服务名、认证证明要求（不需要、必需、必需且 MFA 与近期认证、单次使用，后者依次包含前者）、可接受的权限与近期认证时限。同一方法可以按调用方给出不同要求，例如 backend 附带证明、settlement 以服务身份调用。`make check` 用 `buf build` 的镜像检查每个方法都有合法策略，缺失即失败；servicekit 的服务端拦截器在运行时执行同一策略。`EventStreamService` 的方法策略列出全部可能的接收方，流级授权由各产生方按流清单配置。

## 签名事件、审批与测试向量

签名事件的签名输入、摘要与签名格式写在 `events/v1/event_stream.proto` 的注释中：域标签 `AIPAY-EVENT-V1` 不带长度前缀，字符串与字节串带 4 字节大端长度，整数为 8 字节大端，签名为 ASN.1 DER 编码的 ECDSA P-256。审批摘要为 `SHA-256("AIPAY-APPROVAL-V1" || 规范请求)`。

`testvectors` 包以 JSON 发布分录规范编码 V1，余额摘要、科目余额表摘要、RFC 6962 Merkle 根与检查点签名，签名事件，以及 WebAuthn 审批断言的测试向量（输入、规范字节、期望哈希、公钥与签名，以及必须被拒绝的篡改样例）。包内只有数据，没有编码器或校验器：settlement 与 reconciliation 按规范自行实现分录哈希校验器（决策 A16），再用这些向量测试。向量中的密钥只为生成向量临时创建，私钥已丢弃。

IAM 已有 Telegram/EPay 等签发实现；具体登录来源、密钥托管与撤销能力以其实际交付为准，不因协议存在推断平台管理路径已接通。EPay 证明必须绑定本次请求和权限操作，不能退化为可复用的通用 bearer 凭据。授权的 `allowed` 必须与返回的数据范围约束共同使用，资源服务还需核对真实对象归属及业务状态。

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

## v0.18.0 属主接入上游网关（已发布）

网关模式下，backend 通过 payment 的 `StoreChannelCredentials` 登记交易凭据，取得不可变引用与内容摘要。reconciliation 的 `StoreVerificationCredentials` 在两种模式下都返回内容摘要。交易通道与核验账户在网关模式下提交配置时须绑定已登记的摘要，嵌入模式拒绝该字段。省略字段时可按原字节复验历史审批材料。网关模式只接收 ABI v3 组件包，由本域网关通过 `ReplayPackage` 复验。嵌入模式沿用 ABI v2。见[属主接入上游网关契约](https://github.com/Kaivodev/docs/blob/main/design/contracts.md#v0180-属主接入上游网关)。

下单契约 `CreateOrderRequest` 和 `CreatePaymentInput` 新增可选字段 `client_ip`（买家 IP）和 `device`（设备类型），待 UG-6、UG-7 实现后按上游要求透传。

## v0.17.0 上游网关契约（已发布，提交 `8442a37`）

交易实例只接受 payment，可调用七个业务方法和五个管理方法。核验实例只接受 reconciliation，可调用 `QueryPayment`、`QueryRefund` 和五个管理方法。方法 policy 声明两种实例的调用方并集，网关按启动角色和经 mTLS 验证的对端逐方法收窄权限。幂等、派发结果、证据与管理方法的详细语义见[上游网关契约](https://github.com/Kaivodev/docs/blob/main/design/upstream-gateway-contract.md)。

## v0.15.1 两个写操作要求近期 MFA

`SetSellerChannelOnline` 与 `RequeueArchivedWork` 的方法策略由 `ASSERTION_REQUIREMENT_REQUIRED` 收紧为 `ASSERTION_REQUIREMENT_MFA_RECENT`，`recent_auth_max_age_seconds` 为 600，与 `DisablePlugin` 一致。IAM 对 `payment.channel.online`、`payment.work.requeue` 签发证明前要求近期 MFA，payment 收到后再按证明的认证时间核对。消息与字段不变。

## v0.15.0 卖方上线、归档任务恢复、账户通知与商户对账文件

- payment 的 `SetSellerChannelOnline` 按卖方自己的版本切换接单意愿。卖方视图沿用 `online`，平台视图用 `seller_online`。两个视图都返回 `seller_online_version`，从未切换时省略 `seller_online_changed_at`。新幂等键设置为当前值时仍核对版本，通过后记录未变更审计，返回当前视图，状态和版本保持原值。新单路由同时核对平台启用和卖方上线，在途订单继续处理。
- 平台运维与管理员可用 `ListArchivedWork` 查询失败归档，列表按 `archived_at` 从新到旧、再按 `kind` 和 `object_id` 排序。`RequeueArchivedWork` 以 `(kind, object_id)` 定位任务，带原因按版本恢复。`last_error_code` 使用 payment 登记的固定代码，无法归类时取 `internal`，恢复后保留。处理器重新读取业务状态，可能已派发的操作通过查单收口。卖方切换与归档恢复都校验证明中的 `command_sha256`，规范字节与摘要见 `testvectors.PaymentWebCommandV1()`。
- `payment.account_notices` 由 payment 产生，仅 backend 拉取，用于 bot 推送收款、资金可用、收款方式审核结果三类提示。收款时间取自 `OrderPaid.paid_verified_at`，资金可用时间取自 `TradeReleased` 信封的 `occurred_at`，审核结果以 `payment_access_id` 和 `revision_version` 标识审核对象。资金服务不得使用通知作为资金证据。backend 在投递时调用 IAM 的 `ListAccountNotificationRecipients`，取得账户所有者当前有效的私聊和已确认群聊。
- ledger 的 `ReportService` 列出账户级文件已生成的业务日。业务和应用范围列出相同日期，对应文件在该范围首次 Get 时生成。Get 返回 ZIP、签名清单的 SHA-256 和日结检查点 ID。backend 的账户取自已验证的证明，Get 请求携带 `account_id` 即返回 `INVALID_ARGUMENT`。IAM 核验有效邮件订阅后必须传入 `account_id` 且仅可请求 ACCOUNT 范围，ledger 逐次审计账户和业务日。

方法权限、签名事件与文件范围见[跨服务契约 v0.15.0](https://github.com/Kaivodev/docs/blob/main/design/contracts.md#v0150-卖方上线归档任务恢复账户通知与商户对账文件)。

## v0.12.0 取消与退款核验触发（待发布）

- payment 受理关单并确认上游请求已发出后，在 `payment.orders` 发布 `OrderCancellationVerificationRequested` v1。载荷只含订单 ID、冻结 ID 与控制版本。reconciliation 按订单和版本启动首次独立查单，沿用 v0.11.0 的迟到付款窗口和第二次查单规则。
- 放币前后全额退款均复用 `RefundSucceeded` / `RefundFailed` v2。reconciliation 按退款 ID 和预留控制版本核验，按原预留分支区分资金来源。ledger 识别取消触发以保持流连续，取消或退款入账仍需 reconciliation 的单次签名授权。

产生时刻、幂等键及服务接入顺序见[跨服务契约 v0.12.0](https://github.com/Kaivodev/docs/blob/main/design/contracts.md#v0120-取消与退款的首次核验)。

## v0.11.0 契约增量

- ledger 的 `GetTradeFreezeDetails` 仅供 reconciliation 读取同一账本快照中的交易冻结、全部退款与争议占用，以及原卖方和收款账户的风控冻结。
- reconciliation 放币时使用不早于 payment 核实时间的独立新鲜查单结果。未付款取消需要两次独立核实关单、未收款，并把两次结果绑定到既有签名授权摘要。
- backend 传入可信来源 `client_ip`，IAM 对 EPay 三个凭据分支执行应用白名单，网页会话可记录地址。
- payment 的 `GetCallbackPolicy` 仅供 backend 按通道取得入口限制。回调来源 CIDR 纳入既有通道配置双人审批。
- IAM 增加只读 `GetChallenge`、卖方门户恢复码登录和两步 TOTP 更换。完整方法策略与服务分工见 docs 仓库 `design/contracts.md` 的 v0.11.0 节。

## Collection 最小接入契约（待发布）

`aipay.collection.v1.DepositAddressService` 定义两个账户地址方法：GetOrCreateDepositAddress(request_id, network_id, asset_id) 和 GetAddressAllocation(oneof allocation_id/request_id)。仅 backend、USER/account scope，权限分别为 `collection.address.ensure/read`，证明 REQUIRED，audience=collection；不要求归集管理角色或资金审批。账户来自已验证证明，不接受调用方指定目标、部署字节或签名参数。

地址申请按调用服务＋规范 UUIDv7 请求 ID 幂等；绑定、原请求、订阅意图和审计同事务。查询始终重验范围。尚未完成批准部署、路线及订阅覆盖核验时返回 ALLOCATING、无地址、deposit_enabled=false；已有历史地址的生命周期和当前可用性分开。不把协议已生成当成真实订阅、backend 入口或生产部署已完成。当前实施开放范围仅为这两个地址方法，以下归集三方法仍保留契约、不注册运行入口。

`aipay.collection.v1.CollectionService` 只定义 CreateSweepJob/GetSweepJob/CancelSweepJob；仅 backend，平台 USER/WebAuthn 与单个目标账户，audience=collection。权限为 collection.sweep.create/read/cancel，写 MFA_RECENT 600 秒、读 REQUIRED，不采用 SINGLE_USE。三个请求都绑定完整方法名和规范命令摘要，幂等键与追踪 ID 分离；创建只表示受理，取消只作用于排队任务。方法选项不自动实现入口、归属、摘要或事务校验。

`PlatformRole` 追加 `PLATFORM_ROLE_COLLECTION_OPERATOR=9`，对应 IAM 的受保护角色 `collection-operator`，保留既有 0–8 值。邀请须经第二名平台管理员批准，不能用普通运营或资金审批人角色替代；枚举不授予任何人员权限，不代表邀请/撤销执行层已经实现。旧消费者须升级 SDK 后才支持该角色，未知值不能降级处理。

保留 UUPS/Safe/自有 handleOps，固定批准路线的归集不逐笔双人资金审批，治理仍由 Safe 批准。没有地址监听、到账/入账、List/Preview/Retry/Tick、暂停/停签或 settlement 下发接口；没有通用 Any/map/Struct 载荷或裸签名能力。

`testvectors.CollectionCommandV1()` 发布四组规范字节/摘要，涵盖 UTF-8 字节长度、uint256 最大金额及相同值下不同 oneof 查询分支；消费方独立编码，不用 protobuf wire/JSON 替代。完整字段、权限目录与四条管理 HTTP 目标路由见 docs 的 [Collection 契约](https://github.com/Kaivodev/docs/blob/main/design/contracts.md#collection-最小归集接口)。本轮只交付本地协议/SDK，未发布、未运行服务；公共验签字段、真实平台 IAM、collection/backend 接线须另行验收。

## v0.9.0 门户契约增量

- `payment/v1/payment_admin_service.proto`：平台查询、待审批列表/详情及插件紧急停用。
- `payment/v1/seller_portal_service.proto`、`portal_views.proto`：本人卖方查询投影与不可变追加举证；不提供退款处理或通道上下线命令。
- `ledger/v1/query_service.proto`：payment 批量卖方库存参考、backend 平台/本人余额、已入账预存与担保冻结查询；不开放分录。
- `iam/v1/notification_signing_service.proto`：仅 payment 可调用的历史订单 V1 成功通知签名。
- `payment/v1/plugin_control.proto`、`reconciliation/v1/plugin_control.proto`：插件级 CAS/WebAuthn 紧急停用；ReconciliationAdminService 另新增待审批列表。恢复仍走新的双人启用审批。

新列表使用不透明游标、每页1–100条及一致读时点；`common/v1/query.proto` 的上限选项只是契约声明，运行时仍由属主校验。卖方身份仅来自 IAM 证明绑定，请求不接受卖方账户自报。权限、范围、签名规范、门户逐项映射和后续版本见 docs 仓库 `design/contracts.md` 的 v0.9.0 节。契约/SDK 不代表服务或 HTTP 已接通。

`make breaking` 默认与本地 `v0.17.0` 标签比较。

## 生成与检查

需要 Go 1.27.x 和 Buf CLI；当前校验基线为 Buf 1.57.2。`make tools` 将固定版本的 Go 插件安装到本仓库 `.bin/`，不修改系统工具。

```sh
go install github.com/bufbuild/buf/cmd/buf@v1.57.2
make tools
make fmt
make generate
make check
```

生成文件纳入版本控制，不手工编辑。CI 执行格式、lint、方法策略、编译、测试向量与生成一致性检查，PR 还会与基线提交比较不兼容变更。

字段号一经发布不得复用；删除字段时保留编号和名称，新增枚举的零值必须为 UNSPECIFIED。改变含义或破坏兼容性时使用新版本 package，不能悄悄重定义 v1。参考 [Protobuf 语言规范](https://protobuf.dev/programming-guides/proto3/) 与 [Buf 生成配置](https://buf.build/docs/configuration/v2/buf-gen-yaml/)。

## 发布与本地联调

Go module 为 `github.com/Kaivodev/protobuf`。v0.15.0 以 v0.14.0 为兼容基线。各服务在新版发布后固定依赖版本。工作区 `go.work` 可用于联合开发，但不能替代单仓库 `GOWORK=off` 检查，不在 `go.mod` 写绝对路径 `replace`。

## 名称与兼容性

项目品牌统一为 **Kaivo**，GitHub 组织为 **Kaivodev**。Go module 已迁到 `github.com/Kaivodev/protobuf`，`GOPRIVATE` 使用同样大小写的 `github.com/Kaivodev/*`；协议/签名标识及现有配置键暂时保留原值，属于兼容接口；改名不修改历史账本或协议语义。范围与迁移规则见[命名约定](https://github.com/Kaivodev/docs/blob/main/design/naming.md)。
