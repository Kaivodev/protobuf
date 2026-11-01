# Kaivo 契约仓库规则

- 全部服务间 `.proto` 只在本仓库维护。HTTP/EPay Swagger 由 backend 维护，不混为 gRPC 接口。
- 异步任务使用 Redis + Asynq，但只限同一信任域内部；同域跨服务任务的版本化 Protobuf 载荷由本仓库维护，任务处理器仍归属主服务。跨信任域协作只用 mTLS gRPC 契约（发送方 outbox 中继调用，或接收方拉取签名事件，接收方按事件 ID 幂等），不定义跨信任域的 Asynq 任务载荷。信任域划分见 docs 仓库的 `architecture/security.md`；契约设计见 `design/contracts.md`。具体用例确定前不新增空协议。
- 每个 RPC 必须用方法选项 `(aipay.authz.v1.policy)` 声明允许的调用方、认证证明要求（不需要、必需、必需且 MFA 与近期认证、单次使用）与可接受的权限，取值以 docs 仓库 `design/contracts.md` 的同步 RPC 清单为准；`make check` 拒绝未声明策略的方法。
- package 使用 `aipay.<service>.v1` 等版本化命名，`go_package` 指向本仓库 `gen/go`。
- payment、ledger、settlement、reconciliation 的 RPC、签名事件与载荷以 docs 仓库的 `design/contracts.md` 及各服务详细设计为准，在 M0 起按路线图落地；设计中没有的字段、事件或 RPC 不新增，需要时先改设计。
- IAM 的身份、组织关系和权限来源必须可验证；请求里的 ID、作用域和跟踪字段本身不是授权证据。未知枚举、缺失约束、过期证明和服务错误均须拒绝。
- 凭据、令牌、认证断言不写入日志；不在协议注释、示例和测试中放真实密钥。
- 生成工具版本由 Makefile 固定，执行 `make fmt`、`make generate`、`make check`，生成代码必须与源码一致。
- 不手改生成代码；不复用已发布字段号；删除字段需 reserved；破坏性变更使用新版本 package 并运行 Buf breaking 检查。
- 提交遵循 Conventional Commits，正文包含背景、决策、验证与后续事项，方便协作者和其他 AI Agent 接续。
- 未请求时不提交或推送；已有远程配置保持不变。CLAUDE.md 仅引用本文件。
