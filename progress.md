# pmtop 开发进度

> 按 PRD v1.1 分里程碑实现。每项完成后自我验证（Windows 编译 + Ubuntu VM `go build`/`go test -race`）再继续。

- **Module**: `github.com/pmtop/pmtop`
- **Go**: 1.22+（开发/验证用 1.26.4）
- **日志**: `go.uber.org/zap`（遵循 AGENTS.md，覆盖 PRD 的 log/slog）
- **验证环境**: Ubuntu 24.04 VM（192.168.1.27），Go 1.26.4 @ `$HOME/go-sdk`，Docker + containerd

## 状态总览

| 里程碑 | 内容 | 状态 | 备注 |
|--------|------|------|------|
| 脚手架 | 项目结构、go.mod、Makefile、CI、goreleaser | ✅ 完成 | VM `go build`/`vet`/`run` 通过 |
| M1 | 核心采集层（procfs/inode/process/cgroup/docker） | ✅ 完成 | 单测覆盖 91.4%/86.3%；VM 对 `ss` 计数一致；真实 Docker 容器检测 ID 匹配 `docker ps` |
| M2 | TUI 外壳 | 待开始 | |
| M3 | 过滤系统 | 待开始 | |
| M4 | 进程与容器 | 待开始 | |
| M5 | 特权与配置 | 待开始 | |
| M6 | CLI 模式 | 待开始 | |
| M7 | 手册与补全 | 待开始 | |
| M8 | CI/CD 与打包 | 待开始 | |
| M9 | 发布准备 | 待开始 | |

## 详细日志

### 脚手架
- [x] go.mod / 目录结构
- [x] Makefile
- [x] LICENSE / README
- [x] .goreleaser.yaml
- [x] .github/workflows/ci.yml、release.yml
- [x] .gitignore / .gitattributes
- [x] VM 验证：`go build ./...`、`go vet ./...`、`go run ./cmd/pmtop version` 通过

### M1 核心采集层
- [x] `pkg/netstat`：Protocol/State/Symbol 类型与映射
- [x] `internal/collector/fs.go`：FS 接口 + osFS（跨平台可测）
- [x] `internal/collector/procfs.go`：/proc/net/{tcp,tcp6,udp,udp6,raw,raw6,unix} 解析，小端地址解码，IPv6 `::` 压缩
- [x] `internal/collector/inode_index.go`：单遍 inode→PID 索引
- [x] `internal/collector/process.go`：stat/status/cmdline/exe/cwd/comm + /etc/passwd/group 解析 + btime 启动时间
- [x] `internal/collector/cgroup.go`：cgroup v1/v2 解析 + docker/containerd/podman/crio 检测
- [x] `internal/collector/collector.go`：聚合 Collect + 富化 + 受限模式（WithRestricted）
- [x] 单元测试：collector 91.4% / netstat 86.3%（>80%）
- [x] VM 真实 /proc 集成：socket 计数与 `ss` 完全一致（TCP 19 / UDP 14 / Unix 171 / LISTEN 11）
- [x] VM 真实 Docker 容器：cgroup v2 检测 `Runtime=docker`，ContainerID 与 `docker ps` 一致
- [ ] docker Engine API 富信息客户端（FR-05-03，归入 M4）

## 验证记录

| 日期 | 内容 | 结果 |
|------|------|------|
| 2026-06-25 | 搭建 VM 验证环境（Go 1.26.4、SSH/paramiko、GOPROXY=goproxy.cn） | 通过 |
| 2026-06-25 | 脚手架：`go build`/`go vet`/`pmtop version` | 通过 |
| 2026-06-25 | M1 单测：`go test -race -cover ./internal/collector/... ./pkg/netstat/...` | 通过（91.4% / 86.3%） |
| 2026-06-25 | M1 真实 /proc：socket 计数 vs `ss`（TCP 19/UDP 14/Unix 171/LISTEN 11） | 一致 |
| 2026-06-25 | M1 真实 Docker 容器：cgroup 检测 ContainerID vs `docker ps` | 一致 |
