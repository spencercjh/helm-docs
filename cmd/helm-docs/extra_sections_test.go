package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestDefaultReadmePreservesValueCommentsAfterExtraSections(t *testing.T) {
	chart := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(chart, "Chart.yaml"), []byte("apiVersion: v2\nname: service\nversion: 1.0.0\n"), 0644))
	const values = `replicas: 1

# @extra -- Upgrading
# Read the notes before changing the image.

# -- Image to deploy
# @default -- nginx:1.25
# @extra -- Example
# Example values file:
#
# -- Select a compatible image.
# @default -- example only
image: nginx

# @extra -- Ports
# Check the service port before deployment.

# port -- Port to expose
# @default -- assigned by the platform
port: 8080
`
	require.NoError(t, os.WriteFile(filepath.Join(chart, "values.yaml"), []byte(values), 0644))

	t.Cleanup(viper.Reset)
	cmd, err := newHelmDocsCommand(helmDocs)
	require.NoError(t, err)
	cmd.SetArgs([]string{"--chart-search-root", chart, "--skip-version-footer"})
	require.NoError(t, cmd.Execute())
	readme, err := os.ReadFile(filepath.Join(chart, "README.md"))
	require.NoError(t, err)

	table, sections, found := strings.Cut(string(readme), "### Example\n")
	require.True(t, found, "missing Example section in README")
	require.Contains(t, table, "| image | string | nginx:1.25 | Image to deploy |")
	require.Contains(t, table, "| port | int | assigned by the platform | Port to expose |")
	require.Equal(t, `Example values file:

-- Select a compatible image.
@default -- example only

### Ports
Check the service port before deployment.

### Upgrading
Read the notes before changing the image.`, strings.TrimSpace(sections))
}
