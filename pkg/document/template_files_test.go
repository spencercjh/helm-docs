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

func TestDocumentationTemplateOverridesBlockAcrossFiles(t *testing.T) {
	root := t.TempDir()
	chartA := filepath.Join(root, "chart-a")
	chartB := filepath.Join(root, "chart-b")
	require.NoError(t, os.Mkdir(chartA, 0755))
	require.NoError(t, os.Mkdir(chartB, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "default.gotmpl"),
		[]byte("Changelog\n{{ block \"changelog\" . }}No breaking changes{{ end }}\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(chartA, "on_chart.gotmpl"),
		[]byte("{{ define \"changelog\" }}Breaking changes{{ end }}Footer\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(chartA, "last.gotmpl"),
		[]byte(`{{ define "changelog" }}Latest changes{{ end }}`), 0644))

	for _, tt := range []struct {
		chart string
		files []string
		want  string
	}{
		{chartA, []string{"./default.gotmpl", "on_chart.gotmpl"}, "Changelog\nBreaking changes\nFooter\n"},
		{chartA, []string{"./default.gotmpl", "on_chart.gotmpl", "last.gotmpl"}, "Changelog\nLatest changes\nFooter\n"},
		{chartA, []string{"./default.gotmpl", "on_chart.gotmpl", "missing.gotmpl"}, "Changelog\nBreaking changes\nFooter\n"},
		{chartB, []string{"./default.gotmpl", "on_chart.gotmpl"}, "Changelog\nNo breaking changes\n"},
	} {
		tpl, err := newChartDocumentationTemplate(helm.ChartDocumentationInfo{ChartDirectory: tt.chart}, root, tt.files, "")
		require.NoError(t, err)
		var output strings.Builder
		require.NoError(t, tpl.Execute(&output, chartTemplateData{}))
		assert.Equal(t, tt.want, output.String())
	}
}

func TestDocumentationTemplateKeepsSharedVariablesAndTrimming(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct{ name, first, second string }{
		{"shared variable", "{{ $suffix := \" across files\" }}Body  \n", "{{- $suffix }}\n"},
		{"trim across the boundary", "Body  \n", "{{- \" across files\" }}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(filepath.Join(dir, "first.gotmpl"), []byte(tc.first), 0644))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "second.gotmpl"), []byte(tc.second), 0644))
			tpl, err := newChartDocumentationTemplate(helm.ChartDocumentationInfo{ChartDirectory: dir}, dir, []string{"first.gotmpl", "second.gotmpl"}, "")
			require.NoError(t, err)
			var output strings.Builder
			require.NoError(t, tpl.Execute(&output, chartTemplateData{}))
			assert.Equal(t, "Body across files\n", output.String())
		})
	}
}

func TestDocumentationTemplateFallsBackAfterDefinitionFiles(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"first.gotmpl":  `{{ define "chart.header" }}First header{{ end }}`,
		"second.gotmpl": `{{ define "chart.header" }}Second header{{ end }}`,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0644))
	}
	info := helm.ChartDocumentationInfo{ChartDirectory: dir}
	for _, files := range [][]string{
		{"second.gotmpl", "README.md.gotmpl"},
		{"first.gotmpl", "second.gotmpl", "README.md.gotmpl"},
	} {
		tpl, err := newChartDocumentationTemplate(info, dir, files, "flat-square")
		require.NoError(t, err)
		var output strings.Builder
		data := chartTemplateData{ChartDocumentationInfo: helm.ChartDocumentationInfo{ChartMeta: helm.ChartMeta{Version: "1.0.0"}}}
		require.NoError(t, tpl.Execute(&output, data))
		assert.Equal(t, "Second header\n\n\n![Version: 1.0.0](https://img.shields.io/badge/Version-1.0.0-informational?style=flat-square)", strings.TrimSpace(output.String()))
	}
}

func TestDocumentationTemplateReportsSourceFile(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.gotmpl")
	second := filepath.Join(dir, "second.gotmpl")
	require.NoError(t, os.WriteFile(first, []byte(`{{ block "title" . }}Heading{{ end }}`), 0644))
	info := helm.ChartDocumentationInfo{ChartDirectory: dir}
	files := []string{"first.gotmpl", "second.gotmpl"}

	for _, tc := range []struct{ name, body string }{
		{"execution after an override", "{{ define \"title\" }}New heading{{ end }}\n{{ .MissingField }}"},
		{"execution without an override", "More content\n{{ .MissingField }}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(second, []byte(tc.body), 0644))
			tpl, err := newChartDocumentationTemplate(info, dir, files, "")
			require.NoError(t, err)
			var output strings.Builder
			err = tpl.Execute(&output, chartTemplateData{})
			require.ErrorContains(t, err, second+":2:")
		})
	}

	t.Run("duplicate in one file", func(t *testing.T) {
		body := "{{ define \"title\" }}New heading{{ end }}\n{{ define \"title\" }}Duplicate{{ end }}"
		require.NoError(t, os.WriteFile(second, []byte(body), 0644))
		_, err := newChartDocumentationTemplate(info, dir, files, "")
		require.ErrorContains(t, err, second+":2:")
		require.ErrorContains(t, err, "multiple definition of template")
	})
}
