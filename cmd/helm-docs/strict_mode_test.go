package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrictModeCommand(t *testing.T) {
	if os.Getenv("HELM_DOCS_TEST_STRICT_CLI") == "1" {
		os.Args = append(os.Args[:1], os.Args[3:]...)
		main()
		return
	}

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "Chart.yaml"), []byte("apiVersion: v2\nname: strict-mode\nversion: 1.0.0\n"), 0644))
	values := "# -- Replica count\nreplicas: 1\n# image -- Container image\nimage: example\n" +
		"# -- Ports to expose\nports:\n  - name: web\n    port: 80\n" +
		"# -- Labels for the controller\nlabels:\n  app: controller\n" +
		"# @ignore\ninternal: true\n"
	for _, tc := range []struct {
		name, extraValues string
		wantError         bool
	}{
		{name: "documented"},
		{name: "missing", extraValues: "undocumented: '# -- part of the value'\n", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(filepath.Join(root, "values.yaml"), []byte(values+tc.extraValues), 0644))
			output, err := runStrictMode(root, "-d")
			if tc.wantError {
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr, "%s", output)
				require.Equal(t, 1, exitErr.ExitCode(), "%s", output)
				require.Contains(t, string(output), "values without documentation: \\nundocumented")
				return
			}
			require.NoError(t, err, "%s", output)
			require.Contains(t, string(output), "Replica count")
		})
	}

	t.Run("documented path syntax", func(t *testing.T) {
		root := t.TempDir()
		values := `ports:
  # ports[0].name -- Port name
  - name: web
service:
  annotations:
    # service.annotations."external-dns.alpha.kubernetes.io/hostname" -- DNS hostname
    external-dns.alpha.kubernetes.io/hostname: example.org
`
		writeStrictModeChart(t, root, values)
		output, err := runStrictMode(root, "-y=ports,service,service.annotations", "-z=")
		require.NoError(t, err, "%s", output)
		readme, err := os.ReadFile(filepath.Join(root, "README.md"))
		require.NoError(t, err)
		require.Contains(t, string(readme), "| ports[0].name | string | `\"web\"` | Port name |")
		require.Contains(t, string(readme), `| service.annotations."external-dns.alpha.kubernetes.io/hostname" |`)
		require.Contains(t, string(readme), " | DNS hostname |")
	})

	t.Run("ignore flags", func(t *testing.T) {
		root := t.TempDir()
		writeStrictModeChart(t, root, `image:
  repository: example
  tag: latest
service:
  type: ClusterIP
ports:
  - image:
      tag: latest
    service:
      type: ClusterIP
`)
		goodChart := filepath.Join(root, "good")
		writeStrictModeChart(t, goodChart, "# -- Documented value\ngood: true\n")
		readme := filepath.Join(root, "README.md")
		require.NoError(t, os.WriteFile(readme, []byte("previous README"), 0644))

		parents := []string{"image", "service", "ports", "ports[0].image", "ports[0].service"}
		for _, tc := range []struct {
			name  string
			flags []string
			paths []string
		}{
			{name: "default flags", paths: parents},
			{name: "default regex", flags: []string{"-y="}, paths: parents},
			{
				name:  "default exact paths",
				flags: []string{"-z="},
				paths: []string{"image", "service", "ports", "ports[0].image", "ports[0].image.tag", "ports[0].service", "ports[0].service.type"},
			},
			{
				name:  "both lists cleared",
				flags: []string{"-y=", "-z="},
				paths: []string{"image", "image.repository", "image.tag", "service", "service.type", "ports", "ports[0].image", "ports[0].image.tag", "ports[0].service", "ports[0].service.type"},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				output, err := runStrictMode(root, tc.flags...)
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr, "%s", output)
				require.Equal(t, 1, exitErr.ExitCode())
				_, encodedMessage, found := strings.Cut(strings.TrimSpace(string(output)), "msg=")
				require.True(t, found, "%s", output)
				message, err := strconv.Unquote(encodedMessage)
				require.NoError(t, err, "%s", output)
				require.Equal(t, "chart .: values without documentation: \n"+strings.Join(tc.paths, "\n"), message)
				contents, err := os.ReadFile(readme)
				require.NoError(t, err)
				require.Equal(t, "previous README", string(contents))
				require.NoFileExists(t, filepath.Join(goodChart, "README.md"))
			})
		}
	})
}

func runStrictMode(root string, flags ...string) ([]byte, error) {
	args := []string{"-test.run=^TestStrictModeCommand$", "--", "-x", "-l", "error", "-c", root}
	cmd := exec.Command(os.Args[0], append(args, flags...)...)
	cmd.Env = append(os.Environ(), "HELM_DOCS_TEST_STRICT_CLI=1")
	return cmd.CombinedOutput()
}

func writeStrictModeChart(t *testing.T, dir, values string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte("apiVersion: v2\nname: strict-mode\nversion: 1.0.0\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(values), 0644))
}
