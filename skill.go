package exec

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	tools "github.com/OctoSucker/octosucker-tools"
)

const providerName = "github.com/OctoSucker/tools-exec"

type SkillExec struct {
	mu             sync.RWMutex
	roots          []string
	timeoutSec     int
	blacklist      []string
	sandboxEnabled bool
	sandboxType    string
	sandboxLimits  SandboxLimits
	executor       Executor
	dockerImage    string
	dockerBinary   string
}

func (s *SkillExec) Init(config map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config == nil {
		s.roots = nil
		s.timeoutSec = 0
		s.blacklist = nil
		s.executor = &LocalExecutor{}
		return nil
	}

	raw, _ := config["workspace_dirs"].([]interface{})
	roots, err := normalizeRoots(raw)
	if err != nil {
		return fmt.Errorf("skill-exec: %w", err)
	}
	s.roots = roots

	s.timeoutSec = 30
	if v, ok := config["command_timeout_sec"].(float64); ok && v > 0 {
		s.timeoutSec = int(v)
	}
	if v, ok := config["command_timeout_sec"].(int); ok && v > 0 {
		s.timeoutSec = v
	}

	s.blacklist = nil
	if list, ok := config["command_blacklist"].([]interface{}); ok {
		for _, item := range list {
			if str, ok := item.(string); ok && str != "" {
				s.blacklist = append(s.blacklist, str)
			}
		}
	}

	s.sandboxLimits, s.sandboxEnabled = ParseSandboxLimits(config)
	s.sandboxType = "local"
	if t, ok := config["sandbox_type"].(string); ok && (t == "local" || t == "docker") {
		s.sandboxType = t
	}
	s.dockerImage, _ = config["docker_image"].(string)
	s.dockerBinary, _ = config["docker_binary"].(string)
	if s.dockerBinary == "" {
		s.dockerBinary = "docker"
	}

	if s.sandboxType == "docker" {
		if s.dockerImage != "" {
			s.executor = NewDockerExecutor(s.dockerBinary, s.dockerImage, s.sandboxLimits.Network)
		}
		if s.dockerImage == "" {
			s.executor = &LocalExecutor{}
		}
	} else {
		s.executor = &LocalExecutor{}
	}

	return nil
}

func (s *SkillExec) Cleanup() error {
	s.mu.Lock()
	s.roots = nil
	s.timeoutSec = 0
	s.blacklist = nil
	s.executor = &LocalExecutor{}
	s.mu.Unlock()
	return nil
}

func (s *SkillExec) getExecutor() Executor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.executor != nil {
		return s.executor
	}
	return &LocalExecutor{}
}

func (s *SkillExec) getSandboxLimits() SandboxLimits {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sandboxLimits
}

func (s *SkillExec) getSandboxEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sandboxEnabled
}

func (s *SkillExec) getAllowedRoots() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.roots) == 0 {
		return nil
	}
	out := make([]string, len(s.roots))
	copy(out, s.roots)
	return out
}

func (s *SkillExec) getTimeoutSec() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.timeoutSec <= 0 {
		return 30
	}
	return s.timeoutSec
}

func (s *SkillExec) isBlacklisted(cmdLine string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, pattern := range s.blacklist {
		if strings.Contains(cmdLine, pattern) {
			return true
		}
	}
	return false
}

func RegisterExecSkill(registry *tools.ToolRegistry, agent interface{}) error {
	registry.Register(&tools.Tool{
		Name:        "run_command",
		Description: "在指定工作目录下执行一条命令，支持超时与可选环境变量。工作目录必须在配置的 workspace_dirs 白名单内；禁止无工作目录或根目录 /。超时到点会终止进程。返回中有 stdout、stderr、exit_code。若用户要求把命令执行结果发给他（如 Telegram），必须将返回的 stdout 用 send_telegram_message 发回，不要只输出总结而不发送。",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "要执行的命令，如 \"ls -la\" 或 \"echo hello\"。将按空格拆分为可执行文件与参数，不经过 shell，故管道与分号无效。",
				},
				"work_dir": map[string]interface{}{
					"type":        "string",
					"description": "工作目录。相对路径相对于第一个 workspace_dir；绝对路径必须在 workspace_dirs 内。不传则使用第一个 workspace_dir。",
				},
				"timeout_sec": map[string]interface{}{
					"type":        "integer",
					"description": "超时秒数（可选），不传则使用配置的默认值。",
				},
				"env": map[string]interface{}{
					"type":        "object",
					"description": "可选的环境变量，key 为变量名，value 为字符串。",
				},
			},
			"required": []string{"command"},
		},
		Handler: handleRunCommand,
	})
	return nil
}

func handleRunCommand(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	command, ok := params["command"].(string)
	if !ok || strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("command is required and must be non-empty")
	}

	roots := globalSkillExec.getAllowedRoots()
	if len(roots) == 0 {
		return nil, fmt.Errorf("skill-exec: workspace_dirs not configured. Add workspace_dirs to skill-exec config to allow run_command")
	}

	if globalSkillExec.isBlacklisted(command) {
		return nil, fmt.Errorf("skill-exec: command is forbidden by command_blacklist: %s", command)
	}

	workDir := ""
	if w, ok := params["work_dir"].(string); ok {
		workDir = strings.TrimSpace(w)
	}
	if workDir == "" {
		workDir = roots[0]
	} else {
		resolved, err := resolveWorkDir(workDir, roots)
		if err != nil {
			return nil, err
		}
		workDir = resolved
	}

	timeoutSec := globalSkillExec.getTimeoutSec()
	if v, ok := params["timeout_sec"].(float64); ok && v > 0 {
		timeoutSec = int(v)
	}
	if v, ok := params["timeout_sec"].(int); ok && v > 0 {
		timeoutSec = v
	}

	argv := splitCommand(command)
	if len(argv) == 0 {
		return nil, fmt.Errorf("command produced no executable")
	}

	if isRmCommand(argv) {
		paths := parseRmPaths(argv)
		if len(paths) == 0 {
			return nil, fmt.Errorf("rm requires at least one path argument")
		}
		var absPaths []string
		for _, p := range paths {
			abs, err := resolvePathInWorkspace(p, workDir, roots)
			if err != nil {
				return nil, err
			}
			absPaths = append(absPaths, abs)
		}
		moved, err := moveToTrash(workDir, absPaths)
		if err != nil {
			return nil, fmt.Errorf("move to trash: %w", err)
		}
		trashDir := filepath.Join(workDir, trashDirName)
		msg := fmt.Sprintf("rm is redirected to trash: %d item(s) moved to %s (not deleted)", len(moved), trashDir)
		return map[string]interface{}{
			"success":       true,
			"stdout":        msg,
			"stderr":        "",
			"exit_code":     0,
			"work_dir":      workDir,
			"trash_dir":     trashDir,
			"moved":         moved,
			"rm_redirected": true,
		}, nil
	}

	envList := make([]string, 0)
	if env, ok := params["env"].(map[string]interface{}); ok && len(env) > 0 {
		for k, v := range env {
			if vs, ok := v.(string); ok {
				envList = append(envList, k+"="+vs)
			}
		}
	}

	limits := globalSkillExec.getSandboxLimits()
	if !globalSkillExec.getSandboxEnabled() {
		limits = SandboxLimits{}
	}

	executor := globalSkillExec.getExecutor()
	result, err := executor.Run(ctx, argv, workDir, envList, timeoutSec, limits)
	if err != nil {
		return nil, fmt.Errorf("run_command: %w", err)
	}

	out := map[string]interface{}{
		"success":   result.ExitCode == 0 && !result.Timeout,
		"stdout":    string(result.Stdout),
		"stderr":    string(result.Stderr),
		"exit_code": result.ExitCode,
		"work_dir":  workDir,
	}
	if result.Timeout {
		out["timeout"] = true
		out["message"] = result.Message
	}
	if result.SandboxViolation {
		out["sandbox_violation"] = true
		out["message"] = result.Message
	}
	if result.Message != "" && out["message"] == nil {
		out["message"] = result.Message
	}
	return out, nil
}

// splitCommand 将 "ls -la" 拆分为 ["ls", "-la"]，不解析引号
func splitCommand(s string) []string {
	return strings.Fields(s)
}

var globalSkillExec *SkillExec

func init() {
	globalSkillExec = &SkillExec{}
	tools.RegisterToolProviderWithMetadata(
		providerName,
		tools.ToolProviderMetadata{
			Name:        providerName,
			Version:     "0.1.0",
			Description: "执行 - 在工作区目录内执行命令，带超时与安全策略",
			Author:      "OctoSucker",
			Tags:        []string{"exec", "command", "shell", "runtime"},
		},
		RegisterExecSkill,
		globalSkillExec,
	)
}
