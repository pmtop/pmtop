# pmtop 开发进度

> 按 PRD v1.1 分里程碑实现。每项完成后自我验证（Windows 编译 + Ubuntu VM `go build`/`go test -race`）再继续。

- **Module**: `github.com/pmtop/pmtop`
- **Go**: 1.22+（开发/验证用 1.26.4）
- **日志**: `go.uber.org/zap`（遵循 AGENTS.md，覆盖 PRD 的 log/slog）
- **验证环境**: Ubuntu 24.04 VM（192.168.1.27），Go 1.26.4 @ `$HOME/go-sdk`，Docker + containerd

## 状态总览

| 里程碑 | 内容 | 状态 | 备注 |
|--------|------|------|------|
| 脚手架 | 项目结构、go.mod、Makefile、CI、goreleaser | 进行中 | |
| M1 | 核心采集层（procfs/inode/process/cgroup/docker） | 待开始 | |
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
- [ ] go.mod / 目录结构
- [ ] Makefile
- [ ] LICENSE / README
- [ ] .goreleaser.yaml
- [ ] .github/workflows/ci.yml、release.yml
- [ ] .gitignore

## 验证记录

| 日期 | 内容 | 结果 |
|------|------|------|
| 2026-06-25 | 搭建 VM 验证环境（Go 1.26.4、SSH/paramiko） | 通过 |
