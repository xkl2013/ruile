package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecuteSkillScriptInputUnmarshalAcceptsArrayArgs(t *testing.T) {
	var input ExecuteSkillScriptInput
	err := json.Unmarshal([]byte(`{
		"skill_name": "pdf-processing",
		"script_path": "scripts/analyze_form.py",
		"args": ["/tmp/demo.pdf"]
	}`), &input)

	require.NoError(t, err)
	require.Equal(t, []string{"/tmp/demo.pdf"}, input.Args)
}

func TestExecuteSkillScriptInputUnmarshalAcceptsEncodedArrayArgs(t *testing.T) {
	var input ExecuteSkillScriptInput
	err := json.Unmarshal([]byte(`{
		"skill_name": "pdf-processing",
		"script_path": "scripts/analyze_form.py",
		"args": "[\"/tmp/demo.pdf\"]"
	}`), &input)

	require.NoError(t, err)
	require.Equal(t, []string{"/tmp/demo.pdf"}, input.Args)
}

func TestExecuteSkillScriptInputUnmarshalRejectsInvalidArgs(t *testing.T) {
	var input ExecuteSkillScriptInput
	err := json.Unmarshal([]byte(`{
		"skill_name": "pdf-processing",
		"script_path": "scripts/analyze_form.py",
		"args": "not-an-array"
	}`), &input)

	require.Error(t, err)
}
