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
| M2 | TUI 外壳 | ✅ 完成 | 单测 83.5%；VM 真实 /proc 单帧渲染通过（顶栏/8列表格/状态符号/底栏；root/user 徽标；PID 解析） |
| M3 | 过滤系统 | ✅ 完成 | 单测 app 81.9% / filter 89.8%；VM 真实 /proc 过滤冒烟通过（`sshd` 文本过滤仅显示 sshd socket，过滤栏显示摘要） |
| M4 | 进程与容器 | ✅ 完成 | 单测 app 81.5%/process 85.4%/collector 82.7%；VM 端到端：host-net 容器 nginx:8099 → TUI 显示 PID+进程+容器名 `pmtoptest`（Docker API 解析），详情/信号对话框单测通过 |
| M5 | 特权与配置 | ✅ 完成 | 单测 config 82.5%/export 85.4%；VM `--help`/`version`/`--no-elevate`/flags 正常；TUI `e` 导出 JSON、`SetConfig` 接线、NO_COLOR/色盲、分层 TOML 加载 |
| M6 | CLI 模式 | ✅ 完成 | VM `list`(TSV/JSON/CSV+过滤)、`info`(text/JSON)、`kill`(SIGTERM 终止子进程 exit 143) 全部正常；buildListFilter 单测覆盖 |
| M7 | 手册与补全 | ✅ 完成 | VM `pmtop man` 生成 11 个 man 页（pmtop.8 含 NAME/SYNOPSIS/DESCRIPTION/EXAMPLES）；bash/zsh/fish 补全生成正常 |
| M8 | CI/CD 与打包 | ✅ 完成 | `make build-all` 交叉编译 amd64(11M)+arm64(9.9M) 静态 stripped ELF；goreleaser/ci/release YAML 语法校验通过；nfpm/AUR/homebrew 配置就绪 |
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

### M2 TUI 外壳
- [x] `internal/app`：Model/Update/View + DataSource 接口（解耦 /proc）
- [x] `internal/app/keymap.go`：可配置按键绑定（PRD 6.4，`k`=上移/`K`=发信号，记录偏离）
- [x] `internal/app/sort.go`：多列排序（端口/PID/进程/状态/地址/协议/容器），默认 Proto+端口
- [x] `internal/ui`：8 列表格、状态符号+着色、`NO_COLOR` 处理、顶/底栏
- [x] 导航：↑↓/jk、PgUp/PgDn、Home/End
- [x] 自动刷新（2s，可配）、暂停/恢复（Space）、手动刷新（r）、光标保持
- [x] `internal/platform`：CurrentUID（build tag 分离，跨平台编译）
- [x] `cmd/pmtop`：root 命令启动 TUI
- [x] 单测：app 83.5% / ui 89%
- [x] VM 真实 /proc 单帧渲染冒烟通过

### M3 过滤系统
- [x] `internal/filter`：Filter 结构 + Match/Apply + Summary（AND 逻辑 FR-03-07）
- [x] 端口范围解析（80, 80,443, 8080-8090, 混合，去重排序 FR-03-01）
- [x] 协议/状态多选解析（FR-03-02/03）
- [x] CIDR 解析（IPv4/IPv6，裸 IP 当 /32 或 /128，FR-03-05）
- [x] 进程/PID/用户/容器模糊（大小写不敏感 FR-03-04/06）
- [x] TUI 集成：`/` 实时搜索、`f` 过滤表单（多字段+Tab）、`Esc` 清空（FR-03-08）、过滤栏摘要
- [x] 单测：filter 89.8% / app 81.9%
- [x] VM 真实 /proc 过滤冒烟通过

### M4 进程与容器
- [x] `internal/collector/docker.go`：Docker Engine API 客户端（纯 HTTP over Unix socket，无 CGO/无 Docker SDK）+ ContainerInfo/PortMapping + ParseContainersJSON
- [x] `Collector.WithContainerResolver`：富化容器名/image/status（FR-05-02/03）
- [x] `internal/process/signal.go`：信号定义/解析/校验（FR-06-02，SIGTERM 默认）
- [x] `internal/process/signal_{linux,other}.go`：syscall.Kill 发送（build tag 分离）
- [x] `internal/process/pkg_owner.go`：dpkg -S / rpm -qf 包归属（FR-04-05）
- [x] TUI：`Enter` 进程详情侧栏（PID/PPID/cmdline/exe/cwd/start/mem/包/容器/cgroup FR-04-02）
- [x] TUI：`K` 信号选择对话框 + 确认 + 反馈（FR-06-01..04）
- [x] `ui/panel.go`：Box/Dialog 带边框面板
- [x] 单测：app 81.5% / process 85.4% / collector 82.7%
- [x] VM 端到端：host-net 容器 nginx:8099 → TUI 行显示 `nginx root pmtoptest`（Docker API 容器名解析）
- [ ] 容器详情视图（FR-05-04，P2，归入后续）

### M5 特权与配置
- [x] `internal/elevate`：UID 检测、非 root 横幅、opt-in sudo 重启（FR-07-01..04），build tag 分离
- [x] `internal/config`：分层 TOML（/etc → ~/.config → flags，FR-08-01..04）+ NO_COLOR env（FR-08-06）
- [x] `internal/export`：JSON/CSV/TSV 导出（FR-08-05/FR-03-09）
- [x] `cmd/pmtop`：全局 flags（--no-elevate/--interval/--sort/--asc/--colorblind/--no-color/--docker-socket）
- [x] TUI：`SetConfig` 接线、`e` 导出键写 JSON 文件 + 状态反馈
- [x] 单测：config 82.5% / export 85.4% / app 81.2%
- [x] VM 冒烟：`--help`/`version`/`--no-elevate`/flags 正常

### M6 CLI 模式
- [x] `pmtop list`：TSV(默认)/JSON/CSV + 过滤标志（--ports/--proto/--state/--process/--pid/--user/--container/--local-cidr/--remote-cidr/--text，FR-09-01..03）
- [x] `pmtop kill <PID>`：--signal 名称或数字，默认 SIGTERM（FR-09-04）
- [x] `pmtop info <PID>`：进程详情 text/JSON（FR-09-05）
- [x] 复用 collector/filter/process/export，零重复
- [x] buildListFilter/formatSocketRow 单测
- [x] VM 验证：list 三格式+过滤、info text/json、kill SIGTERM 终止子进程（exit 143）

### M7 手册与补全
- [x] `pmtop man --output-dir`：cobra GenManTree 生成 man 页（PRD 8.2）
- [x] 命令补 Long/Example（root/version/list/kill/info/man，PRD 8.3）
- [x] bash/zsh/fish 补全：cobra 内置 `completion` 子命令（PRD 9）
- [x] Makefile `man`/`completions` 目标接线
- [x] VM 验证：11 个 man 页生成、三 shell 补全生成

### M8 CI/CD 与打包
- [x] `make build-all`：交叉编译 amd64 + arm64，CGO 禁用，-trimpath -ldflags -s -w
- [x] VM 验证：amd64 ELF 可运行、arm64 aarch64 ELF、均静态链接 stripped
- [x] `.goreleaser.yaml`：builds/archives/nfpms(deb+rpm)/aurs/brews/release，YAML 语法校验通过
- [x] `.github/workflows/ci.yml`：lint/test(覆盖率阈值)/build-all
- [x] `.github/workflows/release.yml`：tag→goreleaser→deb/rpm 冒烟测试
- [ ] goreleaser `release --snapshot` 本地全流程（goreleaser 二进制下载超时，留待 CI）

## 验证记录

| 日期 | 内容 | 结果 |
|------|------|------|
| 2026-06-25 | 搭建 VM 验证环境（Go 1.26.4、SSH/paramiko、GOPROXY=goproxy.cn） | 通过 |
| 2026-06-25 | 脚手架：`go build`/`go vet`/`pmtop version` | 通过 |
| 2026-06-25 | M1 单测：`go test -race -cover ./internal/collector/... ./pkg/netstat/...` | 通过（91.4% / 86.3%） |
| 2026-06-25 | M1 真实 /proc：socket 计数 vs `ss`（TCP 19/UDP 14/Unix 171/LISTEN 11） | 一致 |
| 2026-06-25 | M1 真实 Docker 容器：cgroup 检测 ContainerID vs `docker ps` | 一致 |
| 2026-06-25 | M2 单测：`go test -race -cover ./internal/app/... ./internal/ui/...` | 通过（83.5% / 89%） |
| 2026-06-25 | M2 真实 /proc：TUI 单帧渲染（顶栏/表格/状态符号/底栏，root 徽标+PID） | 通过 |
| 2026-06-25 | M3 单测：`go test -race -cover ./internal/filter/... ./internal/app/...` | 通过（89.8% / 81.9%） |
| 2026-06-25 | M3 真实 /proc：文本过滤 `sshd` 仅显示 sshd socket，过滤栏摘要正确 | 通过 |
| 2026-06-25 | M4 单测：`go test -race -cover ./...`（app 81.5%/process 85.4%/collector 82.7%） | 通过 |
| 2026-06-25 | M4 端到端：host-net 容器 nginx:8099 → TUI 显示 PID+nginx+容器名 pmtoptest（Docker API） | 通过 |
| 2026-06-25 | M5 单测：config 82.5% / export 85.4% / app 81.2% | 通过 |
| 2026-06-25 | M5 VM 冒烟：`--help`/`version`/`--no-elevate`/flags、TUI `e` 导出 | 通过 |
| 2026-06-25 | M6 VM：`list`(TSV/JSON/CSV+过滤)、`info`(text/json)、`kill`(SIGTERM exit 143) | 通过 |
| 2026-06-25 | M7 VM：`pmtop man` 生成 11 个 man 页、bash/zsh/fish 补全生成 | 通过 |
| 2026-06-25 | M8 `make build-all`：amd64(11M)+arm64(9.9M) 静态 stripped ELF | 通过 |
| 2026-06-25 | M8 goreleaser/ci/release YAML 语法校验 | 通过 |
