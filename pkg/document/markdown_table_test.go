package document

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/norwoodj/helm-docs/pkg/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValueTablePipeEscaping(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md.gotmpl"),
		[]byte(`{{ define "chart.valueKeyColumnRenderMd" }}key|{{ .Key }}{{ end }}`), 0644))
	tpl, err := newChartDocumentationTemplate(helm.ChartDocumentationInfo{ChartDirectory: dir}, dir, []string{"README.md.gotmpl"}, "")
	require.NoError(t, err)
	values := parseYamlValues("value: 'a\\|b'")
	rows, err := getSortedValuesTableRows(values, nil)
	require.NoError(t, err)
	data := chartTemplateData{Values: append(rows, valueRow{
		Key:         "feature",
		Type:        "a|b",
		AutoDefault: "`\"foo|bar\"`",
		Description: "text | `escaped\\|pipe` `escape-and-backslash\\\\|pipe`",
		Section:     "Feature",
	})}

	var markdown strings.Builder
	require.NoError(t, tpl.ExecuteTemplate(&markdown, "chart.valuesTableMd", data))
	const tableHeader = "| Key | Type | Default | Description |\n|-----|------|---------|-------------|"
	const valueLine = "\n| key\\|value | string | `\"a\\\\\\|b\"` |  |"
	const featureLine = "\n| key\\|feature | a\\|b | `\"foo\\|bar\"` | text \\| `escaped\\|pipe` `escape-and-backslash\\\\\\|pipe` |"
	assert.Equal(t, tableHeader+valueLine+featureLine, markdown.String())

	markdown.Reset()
	data.Sections = getSectionedValueRows(data.Values)
	require.NoError(t, tpl.ExecuteTemplate(&markdown, "chart.valuesTableMd", data))
	assert.Equal(t, "\n\n### Feature\n\n"+tableHeader+featureLine+"\n\n### Other Values\n\n"+tableHeader+valueLine, markdown.String())

	var html strings.Builder
	require.NoError(t, tpl.ExecuteTemplate(&html, "chart.valuesTableHtml", data))
	assert.Contains(t, html.String(), "<td>key|feature</td>")
	assert.Contains(t, html.String(), "<td>a|b</td>")
	assert.Contains(t, html.String(), "<pre lang=\"json\">\n\"foo|bar\"\n</pre>")
	assert.Contains(t, html.String(), "<td>text | `escaped\\|pipe` `escape-and-backslash\\\\|pipe`</td>")
}

func TestValueTableEscapesRawAndCustomDefaults(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "custom.gotmpl"), []byte(`{{ define "chart.valueDefaultColumnRenderMd" }} {{ .Default }} {{ end }}`), 0644))
	info := helm.ChartDocumentationInfo{ChartDirectory: dir}
	for _, tc := range []struct {
		name, values, want, wantHTML string
		files                        []string
	}{
		{
			name:     "raw notation",
			values:   "# -- Raw default\n# @notationType -- tpl\nvalue: 'a\\|b'",
			want:     "\n| value | tpl | a\\\\\\|b | Raw default |",
			wantHTML: "value: |\n  a\\|b\n</pre>",
		},
		{
			name:     "custom markdown default",
			values:   "value: 'call(`a\\|b`)'",
			want:     "\n| value | string |  ``\"call(`a\\\\\\|b`)\"``  |  |",
			wantHTML: "<pre lang=\"json\">\n\"call(`a\\\\|b`)\"\n</pre>",
			files:    []string{"custom.gotmpl"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := getSortedValuesTableRows(parseYamlValues(tc.values), nil)
			require.NoError(t, err)
			tpl, err := newChartDocumentationTemplate(info, dir, tc.files, "")
			require.NoError(t, err)
			var markdown strings.Builder
			require.NoError(t, tpl.ExecuteTemplate(&markdown, "chart.valuesTableMd", chartTemplateData{Values: rows}))
			assert.Equal(t, "| Key | Type | Default | Description |\n|-----|------|---------|-------------|"+tc.want, markdown.String())

			var html strings.Builder
			require.NoError(t, tpl.ExecuteTemplate(&html, "chart.valuesTableHtml", chartTemplateData{Values: rows}))
			assert.Contains(t, html.String(), tc.wantHTML)
		})
	}
}
