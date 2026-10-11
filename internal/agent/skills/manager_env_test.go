package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Tencent/WeKnora/internal/sandbox"
	"github.com/stretchr/testify/require"
)

type capturingSkillSandboxManager struct {
	config *sandbox.ExecuteConfig
}

func (m *capturingSkillSandboxManager) Execute(
	_ context.Context,
	config *sandbox.ExecuteConfig,
) (*sandbox.ExecuteResult, error) {
	m.config = config
	return &sandbox.ExecuteResult{}, nil
}

func (m *capturingSkillSandboxManager) Cleanup(context.Context) error { return nil }

func (m *capturingSkillSandboxManager) GetSandbox() sandbox.Sandbox { return nil }

func (m *capturingSkillSandboxManager) GetType() sandbox.SandboxType {
	return sandbox.SandboxTypeLocal
}

func TestManagerInjectsConfiguredSkillEnvironment(t *testing.T) {
	skillRoot := t.TempDir()
	skillDir := filepath.Join(skillRoot, "env-demo")
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: env-demo\ndescription: Environment demo\n---\n\nRun the script."),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(skillDir, "scripts", "run.py"),
		[]byte("print('ok')"),
		0o644,
	))

	sandboxManager := &capturingSkillSandboxManager{}
	manager := NewManager(&ManagerConfig{
		SkillDirs: []string{skillRoot},
		SkillEnvs: map[string]map[string]string{
			"env-demo": {"API_TOKEN": "secret"},
		},
		Enabled: true,
	}, sandboxManager)

	_, err := manager.ExecuteScript(context.Background(), "env-demo", "scripts/run.py", nil, "")
	require.NoError(t, err)
	require.NotNil(t, sandboxManager.config)
	require.Equal(t, map[string]string{"API_TOKEN": "secret"}, sandboxManager.config.Env)
}
