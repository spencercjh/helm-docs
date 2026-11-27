package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestWriteDocumentationReportsFailedChartsAndPreservesReadmes(t *testing.T) {
	root := t.TempDir()
	templates := map[string]string{
		"bad-syntax": "{{ if }}",
		"bad-field":  "partial output {{ .MissingField }}",
		"good":       "updated\n",
	}
	for name, template := range templates {
		dir := filepath.Join(root, name)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		for path, content := range map[string]string{
			"Chart.yaml":       "apiVersion: v2\nname: " + name + "\nversion: 1.0.0\n",
			"values.yaml":      "{}\n",
			"README.md.gotmpl": template,
			"README.md":        "previous",
		} {
			if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := viper.BindFlagValues(testFlagSet{
		"chart-search-root":          root,
		"template-files":             "README.md.gotmpl",
		"values-file":                "values.yaml",
		"output-file":                "README.md",
		"ignore-file":                ".helmdocsignore",
		"document-dependency-values": false,
		"log-level":                  "warn",
	}); err != nil {
		t.Fatal(err)
	}
	charts, err := readDocumentationInfoByChartPath(root, 2)
	if err != nil {
		t.Fatal(err)
	}
	err = writeDocumentation(root, charts, false, 2)
	if err == nil {
		t.Fatal("expected chart generation errors")
	}
	for _, name := range []string{"bad-syntax", "bad-field"} {
		if !strings.Contains(err.Error(), filepath.Join(root, name)) {
			t.Errorf("error should identify chart %s: %v", name, err)
		}
	}
	for name, want := range map[string]string{"bad-syntax": "previous", "bad-field": "previous", "good": "updated\n"} {
		got, err := os.ReadFile(filepath.Join(root, name, "README.md"))
		if err != nil || string(got) != want {
			t.Errorf("%s README = %q, %v; want %q", name, got, err, want)
		}
	}
}
