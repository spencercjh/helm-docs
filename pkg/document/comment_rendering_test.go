package document

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/norwoodj/helm-docs/pkg/helm"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestChartDescriptionsSkipInternalComments(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte("apiVersion: v2\nname: comments\nversion: 1.0.0\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(`app:
  ## app.internal -- Example only
  # -- Anonymous prose
  # app.named -- Named documentation
  ## @default -- ignored default
  ## @section -- ignored section
  ## @raw
  ##
  # continuation
  named: true

  ## internal note mentioning # -- syntax
  # ordinary YAML comment
  undocumented: 1

  # -- Headings
  # @raw
  # ## Heading
  heading: true
`), 0644))
	viper.Set("values-file", "values.yaml")
	t.Cleanup(viper.Reset)
	info, err := helm.ParseChartInformation(dir, helm.ChartValuesDocumentationParsingConfig{})
	require.NoError(t, err)
	require.Equal(t, map[string]helm.ChartValueDescription{
		"app.named": {Description: "Named documentation continuation"},
	}, info.ChartValuesDescriptions)

	rows, err := getUnsortedValueRows(info.ChartValues, info.ChartValuesDescriptions)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, "app.named", rows[0].Key)
	require.Equal(t, "`true`", rows[0].Default)
	require.Empty(t, rows[0].Section)
	require.Equal(t, "app.undocumented", rows[1].Key)
	require.Empty(t, rows[1].AutoDescription)
	require.Equal(t, "Headings\n## Heading", rows[2].AutoDescription)
}
