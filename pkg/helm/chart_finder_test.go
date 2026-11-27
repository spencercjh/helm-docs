package helm

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestFindChartDirectoriesThroughSymlink(t *testing.T) {
	root := t.TempDir()
	for _, chart := range []string{"charts", "charts/nested", "charts/ignored"} {
		chartDir := filepath.Join(root, chart)
		require.NoError(t, os.MkdirAll(chartDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), nil, 0644))
	}
	require.NoError(t, os.Symlink("charts", filepath.Join(root, "alias")))
	require.NoError(t, os.Symlink(".", filepath.Join(root, "charts", "cycle")))

	t.Chdir(root)
	viper.SetDefault("ignore-file", ".helmdocsignore")
	t.Cleanup(viper.Reset)
	require.NoError(t, os.WriteFile(".helmdocsignore", []byte("/charts/ignored/\n"), 0644))

	charts, err := FindChartDirectories("alias")
	require.NoError(t, err)
	require.Equal(t, []string{".", "nested"}, charts)

	t.Chdir(filepath.Join(root, "alias"))
	require.NoError(t, os.WriteFile(".helmdocsignore", []byte("/ignored/\n"), 0644))
	workingDir, err := os.Getwd()
	require.NoError(t, err)
	charts, err = FindChartDirectories(workingDir)
	require.NoError(t, err)
	require.Equal(t, []string{".", "nested"}, charts)
}

func TestFindChartDirectoriesThroughSymlinkUsesRepositoryIgnorePaths(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git is required to test repository ignore paths")
	}
	root := t.TempDir()
	repository := filepath.Join(root, "repo")
	charts := filepath.Join(repository, "charts")
	require.NoError(t, os.MkdirAll(filepath.Join(charts, "ignored"), 0755))
	output, err := exec.Command("git", "init", "--quiet", repository).CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.NoError(t, os.WriteFile(filepath.Join(repository, ".helmdocsignore"), []byte("/charts/ignored/\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(charts, "Chart.yaml"), nil, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(charts, "ignored", "Chart.yaml"), nil, 0644))
	alias := filepath.Join(root, "alias")
	require.NoError(t, os.Symlink(charts, alias))

	t.Chdir(alias)
	viper.SetDefault("ignore-file", ".helmdocsignore")
	t.Cleanup(viper.Reset)
	found, err := FindChartDirectories(alias)
	require.NoError(t, err)
	require.Equal(t, []string{"."}, found)
}
