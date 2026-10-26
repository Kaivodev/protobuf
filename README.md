# AI Pay Protobuf 契约

本仓库集中维护 AI Pay 服务间的 `.proto`，与用户侧 HTTP/EPay 的 Swagger 文档分开管理。各服务引用生成 SDK，不维护协议副本。

```text
proto/aipay/
  common/v1/context.proto          跟踪信息、平台/组织/业务主体作用域
  iam/v1/identity.proto            认证输入、主体、短期认证证明
  iam/v1/authorization.proto       角色、权限、绑定、授权检查与数据范围约束
  ledger/v1/ledger.proto           账务契约占位
  settlement/v1/settlement.proto   结算契约占位
  reconciliation/v1/reconciliation.proto  对账契约占位
gen/go/                           Go / gRPC 生成代码
```

IAM 初版契约包含 `IdentityService.Authenticate` 和 `AuthorizationService.CheckPermission`，仅定义接口，不表示服务已运行。身份凭据、组织归属、角色策略在 IAM 仓库管理；金融服务只预留 package，不编造资金字段或 RPC。

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

生成文件纳入版本控制，不手工编辑。CI 执行格式、lint、编译、生成一致性检查，PR 还会与基线提交比较不兼容变更。

字段号一经发布不得复用；删除字段时保留编号和名称，新增枚举的零值必须为 UNSPECIFIED。改变含义或破坏兼容性时使用新版本 package，不能悄悄重定义 v1。参考 [Protobuf 语言规范](https://protobuf.dev/programming-guides/proto3/) 与 [Buf 生成配置](https://buf.build/docs/configuration/v2/buf-gen-yaml/)。

## 发布与本地联调

Go module 为 `github.com/aipay-dev/protobuf`。当前是尚未发布的本地草案；各服务正式导入前先提交、推送并发布可解析的版本，再固定该版本依赖。工作区 `go.work` 可用于联合开发，但不能替代单仓库 `GOWORK=off` 检查，不在 `go.mod` 写绝对路径 `replace`。
