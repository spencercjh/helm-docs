package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiFileTemplateCommand(t *testing.T) {
	if os.Getenv("HELM_DOCS_TEST_TEMPLATE_CLI") == "1" {
		os.Args = append(os.Args[:1], os.Args[3:]...)
		main()
		return
	}

	for _, tc := range []struct {
		name, first, second, want string
		wantError                 bool
	}{
		{
			name:   "shared body scope",
			first:  `{{ $title := "First" }}{{ if $title }}  `,
			second: "{{- $title }} from both{{ end }}\n",
			want:   "First from both\n",
		},
		{
			name:   "named override",
			first:  "{{ block \"title\" . }}Fallback{{ end }}\n",
			second: `{{ define "title" }}Override{{ end }}`,
			want:   "Override\n",
		},
		{
			name:      "unterminated named template",
			first:     `{{ define "broken" }}`,
			second:    `Document body{{ end }}`,
			want:      "previous README\n",
			wantError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for name, content := range map[string]string{
				"Chart.yaml":    "apiVersion: v2\nname: templates\nversion: 1.0.0\n",
				"values.yaml":   "{}\n",
				"first.gotmpl":  tc.first,
				"second.gotmpl": tc.second,
				"README.md":     "previous README\n",
			} {
				require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0644))
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestMultiFileTemplateCommand$", "--", "-c", root, "-t", "first.gotmpl", "-t", "second.gotmpl")
			cmd.Env = append(os.Environ(), "HELM_DOCS_TEST_TEMPLATE_CLI=1")
			output, err := cmd.CombinedOutput()
			if tc.wantError {
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr, "%s", output)
				require.Equal(t, 1, exitErr.ExitCode(), "%s", output)
				require.Contains(t, string(output), filepath.Join(root, "first.gotmpl")+":1: unexpected EOF")
			} else {
				require.NoError(t, err, "%s", output)
			}
			readme, err := os.ReadFile(filepath.Join(root, "README.md"))
			require.NoError(t, err)
			require.Equal(t, tc.want, string(readme))
		})
	}
}
