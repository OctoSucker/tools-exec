# tools-exec

在 OctoSucker Agent 的工作区目录内执行本地命令，带超时、安全策略与可选沙箱（进程级资源限制或 Docker 容器）。

## 工具

| 工具 | 说明 |
|------|------|
| `run_command` | 在指定工作目录下执行一条命令。参数：`command`（必填，按空格拆分为可执行文件与参数，**不经过 shell**，管道与分号无效）、`work_dir`（可选，默认第一个 workspace_dir）、`timeout_sec`（可选）、`env`（可选，key-value 环境变量）。返回 stdout、stderr、exit_code。工作目录必须在 `workspace_dirs` 白名单内；超时到点会终止进程。 |

未配置 `workspace_dirs` 或列表为空时，工具返回错误，不执行任何命令。命中 `command_blacklist` 的命令会被拒绝。

**安全行为**：`rm` 类命令不会真正删除文件，而是将目标移动到工作目录下的 `.trash` 目录。

## 配置

在 Agent 配置的 `tool_providers["github.com/OctoSucker/tools-exec"]` 下：

| 键 | 说明 |
|------|------|
| `workspace_dirs` | 工作目录白名单（必填）。仅允许在此列表内执行命令。 |
| `command_timeout_sec` | 单次命令超时秒数，默认 30。 |
| `command_blacklist` | 命令黑名单（子串匹配），字符串数组。命中则拒绝执行。 |
| `sandbox_enabled` | 是否启用沙箱，可选。 |
| `sandbox_type` | `local`（setrlimit，进程级资源限制）或 `docker`（容器内执行），可选。 |
| `sandbox_cpu_sec` | 沙箱 CPU 时间限制（秒），可选。 |
| `sandbox_memory_mb` | 沙箱内存限制（MB），可选。 |
| `sandbox_max_procs` | 沙箱最大进程数，可选。 |
| `sandbox_max_open_files` | 沙箱最大打开文件数，可选。 |
| `sandbox_network` | 沙箱网络：`none` 或 `full`（仅 docker 时生效），可选。 |
| `docker_image` | Docker 沙箱使用的镜像，如 `alpine:latest`。 |
| `docker_binary` | Docker 可执行路径，默认 `docker`。 |

示例（`config/agent_config.json`）：

```json
"github.com/OctoSucker/tools-exec": {
  "workspace_dirs": ["."],
  "command_timeout_sec": 30,
  "command_blacklist": ["rm -rf /", "curl | bash", "mkfs."],
  "sandbox_enabled": true,
  "sandbox_type": "docker",
  "sandbox_network": "none",
  "docker_image": "alpine:latest",
  "docker_binary": "docker"
}
```

沙箱与安全策略详见主项目 `docs/SKILL_EXEC.md`（若存在）。

## 安装

主项目中：

```bash
go get github.com/OctoSucker/tools-exec@latest
```

并保留空白导入：`_ "github.com/OctoSucker/tools-exec"`。在 `tool_providers` 下增加 `github.com/OctoSucker/tools-exec` 配置段（见上文）。
