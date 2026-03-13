# skill-exec

在 OctoSucker Agent 的工作区目录内执行本地命令，带超时、安全策略与可选沙箱（进程级资源限制或 Docker 容器）。

## 工具

- **run_command**：在指定工作目录下执行命令，支持超时与可选环境变量；可配置沙箱（local/docker）。

## 配置

需在 Agent 配置中提供 `exec.workspace_dirs`（工作目录白名单）。可选：`command_timeout_sec`、`command_blacklist`、沙箱相关（`sandbox_enabled`、`sandbox_type`、`sandbox_cpu_sec`、`sandbox_memory_mb`、`docker_image` 等）。

详见 [docs/SKILL_EXEC.md](../docs/SKILL_EXEC.md)。

## 安装

在 OctoSucker 主项目中：

```bash
go get github.com/OctoSucker/skill-exec@latest
```

并在 `main.go` 中增加空白导入：

```go
_ "github.com/OctoSucker/skill-exec"
```

配置中增加 `exec` 段（见上文文档）。
