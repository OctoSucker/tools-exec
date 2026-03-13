package exec

type SandboxLimits struct {
	CPUsec      int
	MemoryMB    int
	MaxProcs    int
	MaxOpenFiles int
	Network     string
}

func ParseSandboxLimits(config map[string]interface{}) (limits SandboxLimits, enabled bool) {
	if config == nil {
		return SandboxLimits{}, false
	}
	enabled, _ = config["sandbox_enabled"].(bool)
	if !enabled {
		return SandboxLimits{}, false
	}
	if v, ok := config["sandbox_cpu_sec"].(float64); ok && v > 0 {
		limits.CPUsec = int(v)
	}
	if v, ok := config["sandbox_cpu_sec"].(int); ok && v > 0 {
		limits.CPUsec = v
	}
	if v, ok := config["sandbox_memory_mb"].(float64); ok && v > 0 {
		limits.MemoryMB = int(v)
	}
	if v, ok := config["sandbox_memory_mb"].(int); ok && v > 0 {
		limits.MemoryMB = v
	}
	if v, ok := config["sandbox_max_procs"].(float64); ok && v > 0 {
		limits.MaxProcs = int(v)
	}
	if v, ok := config["sandbox_max_procs"].(int); ok && v > 0 {
		limits.MaxProcs = v
	}
	if v, ok := config["sandbox_max_open_files"].(float64); ok && v > 0 {
		limits.MaxOpenFiles = int(v)
	}
	if v, ok := config["sandbox_max_open_files"].(int); ok && v > 0 {
		limits.MaxOpenFiles = v
	}
	if s, ok := config["sandbox_network"].(string); ok && (s == "none" || s == "full") {
		limits.Network = s
	}
	return limits, enabled
}
