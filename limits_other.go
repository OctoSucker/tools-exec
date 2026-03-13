//go:build !linux && !darwin

package exec

func applySandboxLimits(limits SandboxLimits) (restore func(), err error) {
	return func() {}, nil
}
