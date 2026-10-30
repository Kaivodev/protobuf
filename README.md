# AI Pay Protobuf 契约

本仓库集中维护 AI Pay 服务间的 `.proto`，与用户侧 HTTP/EPay 的 Swagger 文档分开管理。各服务引用生成 SDK，不维护协议副本。

```text
proto/aipay/
  authz/v1/policy.proto            方法授权选项 (aipay.authz.v1.policy)
  common/v1/context.proto          跟踪信息、平台/组织/业务主体作用域
  common/v1/money.proto            Money、汇率；标识、时间与金额约定
  common/v1/asset.proto            币种表、链与资产登记（TRON/BSC USDT、TRX、BNB）
  events/v1/event_stream.proto     签名事件信封与 EventStreamService 拉取接口
  approval/v1/approval.proto       审批请求与 WebAuthn 断言
  iam/v1/identity.proto            认证输入、主体、短期认证证明
  iam/v1/authorization.proto       角色、权限、绑定、授权检查与数据范围约束
  payment/v1/payment.proto         支付订单契约占位
  ledger/v1/ledger.proto           账务契约占位
  settlement/v1/settlement.proto   结算契约占位
  reconciliation/v1/reconciliation.proto  对账契约占位
gen/go/                           Go / gRPC 生成代码
testvectors/                      规范编码与签名测试向量（只含数据）
internal/policycheck/             make check 的方法策略检查
```

币种与链上资产以枚举值选项登记（`currency_info`、`chain_info`、`asset_info`），各服务从生成 SDK 读取同一张表；新增币种、链或合约需同时修改本仓库与各服务测试。

IAM 契约只有一个认证授权入口 `AuthorizationService.Authorize`：网关每个请求只调用一次，同时完成认证与授权；只需确认身份的接口使用表示“本人”的权限，同样走 `Authorize`。下游服务自行核验认证证明，不再回调 IAM（决策 A78）。这仅是接口定义，不表示服务已运行。

授权作用域按账户模型（决策 A83、A84）分为 `platform`、`account`、`business`、`application`（`common/v1/context.proto`）。资金只归属 `account_id`（个人或组织账户）；业务、应用作用域依次带上 `business_id`、`application_id` 及其所属账户，但不隐含账户级资金权限。组织账户可附 `org_id`，个人账户不填。旧的组织、业务主体作用域使用过的枚举值与字段号已保留，不再复用，旧版本发出的消息会校验失败。身份凭据、组织归属、角色策略在 IAM 仓库管理。金融服务目前只有 package 占位；业务 RPC、事件载荷与资金字段按 docs 仓库的[跨服务契约设计](https://github.com/aipay-dev/docs/blob/main/design/contracts.md)在 M0 落地，设计之外不新增。

## 方法授权策略

每个 RPC 用 `(aipay.authz.v1.policy)` 声明一组调用方规则：允许的 mTLS 服务名、认证证明要求（不需要、必需、必需且 MFA 与近期认证、单次使用，后者依次包含前者）、可接受的权限与近期认证时限。同一方法可以按调用方给出不同要求，例如 backend 附带证明、settlement 以服务身份调用。`make check` 用 `buf build` 的镜像检查每个方法都有合法策略，缺失即失败；servicekit 的服务端拦截器在运行时执行同一策略。`EventStreamService` 的方法策略列出全部可能的接收方，流级授权由各产生方按流清单配置。

## 签名事件、审批与测试向量

签名事件的签名输入、摘要与签名格式写在 `events/v1/event_stream.proto` 的注释中：域标签 `AIPAY-EVENT-V1` 不带长度前缀，字符串与字节串带 4 字节大端长度，整数为 8 字节大端，签名为 ASN.1 DER 编码的 ECDSA P-256。审批摘要为 `SHA-256("AIPAY-APPROVAL-V1" || 规范请求)`。

`testvectors` 包以 JSON 发布分录规范编码 V1、签名事件与 WebAuthn 审批断言的测试向量（输入、规范字节、期望哈希、公钥与签名，以及必须被拒绝的篡改样例）。包内只有数据，没有编码器或校验器：settlement 与 reconciliation 按规范自行实现分录哈希校验器（决策 A16），再用这些向量测试。向量中的密钥只为生成向量临时创建，私钥已丢弃。

认证证明的签发格式、密钥托管、调用方绑定与撤销机制尚待 IAM 实现；EPay 证明必须绑定本次请求和权限操作，不能退化为可复用的通用 bearer 凭据。授权的 `allowed` 必须与返回的数据范围约束共同使用，资源服务还需核对真实对象归属及业务状态。

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

Go module 为 `github.com/aipay-dev/protobuf`。当前是尚未发布的本地草案；各服务正式导入前先提交、推送并发布可解析的版本，再固定该版本依赖。工作区 `go.work` 可用于联合开发，但不能替代单仓库 `GOWORK=off` 检查，不在 `go.mod` 写绝对路径 `replace`。
