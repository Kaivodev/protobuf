# AI Pay 契约仓库规则

- 全部服务间 `.proto` 只在本仓库维护。HTTP/EPay Swagger 由 backend 维护，不混为 gRPC 接口。
- package 使用 `aipay.<service>.v1` 等版本化命名，`go_package` 指向本仓库 `gen/go`。
- 金融服务当前只留契约占位；没有业务设计时不编造记账、结算或对账 RPC。
- IAM 的身份、组织关系和权限来源必须可验证；请求里的 ID、作用域和跟踪字段本身不是授权证据。未知枚举、缺失约束、过期证明和服务错误均须拒绝。
- 凭据、令牌、认证断言不写入日志；不在协议注释、示例和测试中放真实密钥。
- 生成工具版本由 Makefile 固定，执行 `make fmt`、`make generate`、`make check`，生成代码必须与源码一致。
- 不手改生成代码；不复用已发布字段号；删除字段需 reserved；破坏性变更使用新版本 package 并运行 Buf breaking 检查。
- 提交遵循 Conventional Commits，正文包含背景、决策、验证与后续事项，方便协作者和其他 AI Agent 接续。
- 未请求时不提交或推送；已有远程配置保持不变。CLAUDE.md 仅引用本文件。
