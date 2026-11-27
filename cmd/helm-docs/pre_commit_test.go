package main

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPreCommitHooksMatchValuesFiles(t *testing.T) {
	manifest, err := os.ReadFile("../../.pre-commit-hooks.yaml")
	require.NoError(t, err)
	var hooks []struct {
		ID    string
		Files string
	}
	require.NoError(t, yaml.Unmarshal(manifest, &hooks))
	require.NotEmpty(t, hooks)
	for _, hook := range hooks {
		t.Run(hook.ID, func(t *testing.T) {
			files, err := regexp.Compile(hook.Files)
			require.NoError(t, err)
			for _, name := range []string{"README.md.gotmpl", "charts/demo/Chart.yaml", "requirements.yaml", "values.yaml", "charts/demo/values-prod.yaml", "charts/demo/values.prod.yml"} {
				require.True(t, files.MatchString(name), "should match %s", name)
			}
			for _, name := range []string{"deploy.yaml", "values/document.yaml", "values-prod.yaml.bak", "unrelated-values.yaml"} {
				require.False(t, files.MatchString(name), "should skip %s", name)
			}
		})
	}
}
