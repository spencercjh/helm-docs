package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionEnvFlagsAndDryRun(t *testing.T) {
	workspace := newWorkspace(t)
	chartDir := filepath.Join(workspace, "demo-chart")
	const sentinelContent = "sentinel-keep\n"
	const valuesContent = "# replicaCount -- Number of replicas\nreplicaCount: 2\n"

	writeChartFixture(t, chartDir, chartFixture{
		Name:       "demo-chart",
		Version:    "0.1.0",
		AppVersion: "1.0.0",
		ValuesYAML: valuesContent,
	})
	sentinelReadmePath := filepath.Join(chartDir, "README.md")
	writeTextFile(t, sentinelReadmePath, sentinelContent)

	sharedEnv := map[string]string{
		"HELM_DOCS_BADGE_STYLE":         "plastic",
		"HELM_DOCS_SKIP_VERSION_FOOTER": "true",
	}

	t.Run("version_flag", func(t *testing.T) {
		res := runHelmDocs(t, cliInvocation{
			Workspace: workspace,
			Dir:       chartDir,
			Args:      []string{"--version"},
		})
		requireExitZero(t, res)
		assertExactCLIVersionOutput(t, res.Stdout, injectedVersion)
		assert.Equal(t, sentinelContent, readTextFile(t, sentinelReadmePath))
	})

	t.Run("env_binding_and_dry_run_protection", func(t *testing.T) {
		res := runHelmDocs(t, cliInvocation{
			Workspace: workspace,
			Dir:       chartDir,
			Args:      []string{"-c", ".", "--dry-run"},
			Env:       sharedEnv,
		})
		requireExitZero(t, res)
		assert.Contains(t, res.Stdout, "style=plastic")
		assert.NotContains(t, res.Stdout, injectedVersion)
		assert.NotContains(t, res.Stdout, "[helm-docs v")
		assert.Contains(t, res.Stderr, "Generating README Documentation")
		assert.NotContains(t, res.Stdout, "Generating README Documentation")
		assert.Equal(t, sentinelContent, readTextFile(t, sentinelReadmePath))
	})

	t.Run("cli_flags_override_env_and_stream_separation", func(t *testing.T) {
		dryRunRes := runHelmDocs(t, cliInvocation{
			Workspace: workspace,
			Dir:       chartDir,
			Args: []string{
				"--badge-style=for-the-badge",
				"--skip-version-footer=false",
				"-c", ".",
				"--dry-run",
			},
			Env: sharedEnv,
		})
		requireExitZero(t, dryRunRes)
		assert.Contains(t, dryRunRes.Stdout, "style=for-the-badge")
		assert.NotContains(t, dryRunRes.Stdout, "style=plastic")
		assertFooterVersionEquals(t, dryRunRes.Stdout, injectedVersion)
		assert.Contains(t, dryRunRes.Stderr, "Generating README Documentation")
		assert.NotContains(t, dryRunRes.Stdout, "Generating README Documentation")
		assert.Equal(t, sentinelContent, readTextFile(t, sentinelReadmePath))

		// Verify stream separation cleanly without tying assertions to logrus formatting:
		// dry-run stdout must match the exact file content produced by an equivalent disk write.
		diskChartDir := filepath.Join(workspace, "demo-chart-disk")
		writeChartFixture(t, diskChartDir, chartFixture{
			Name:       "demo-chart",
			Version:    "0.1.0",
			AppVersion: "1.0.0",
			ValuesYAML: valuesContent,
		})
		writeRes := runHelmDocs(t, cliInvocation{
			Workspace: workspace,
			Dir:       diskChartDir,
			Args: []string{
				"--badge-style=for-the-badge",
				"--skip-version-footer=false",
				"-c", ".",
			},
			Env: sharedEnv,
		})
		requireExitZero(t, writeRes)
		assert.Empty(t, writeRes.Stdout)
		assert.Equal(t, readTextFile(t, filepath.Join(diskChartDir, "README.md")), dryRunRes.Stdout)
	})
}

func TestIgnoreRulesIsolation(t *testing.T) {
	t.Run("git_subdirectory_reads_repo_root_helmdocsignore", func(t *testing.T) {
		workspace := newWorkspace(t)
		repoDir := filepath.Join(workspace, "repo")
		initGitRepo(t, workspace, repoDir)

		ignoreRules := strings.Join([]string{
			"/" + filepath.Join("charts", "ignored-dir"),
			"/" + filepath.Join("charts", "ignored-file", "Chart.yaml"),
		}, "\n") + "\n"
		writeTextFile(t, filepath.Join(repoDir, ".helmdocsignore"), ignoreRules)

		chartsDir := filepath.Join(repoDir, "charts")
		writeChartFixture(t, filepath.Join(chartsDir, "active"), chartFixture{
			Name:       "active",
			ValuesYAML: "# activeChartKey -- Active chart value\nactiveChartKey: present\n",
		})
		writeChartFixture(t, filepath.Join(chartsDir, "ignored-dir"), chartFixture{
			Name:       "ignored-dir",
			ValuesYAML: "# ignoredDirKey -- Ignored directory chart value\nignoredDirKey: hidden\n",
		})
		writeChartFixture(t, filepath.Join(chartsDir, "ignored-file"), chartFixture{
			Name:       "ignored-file",
			ValuesYAML: "# ignoredFileKey -- Ignored Chart.yaml value\nignoredFileKey: hidden\n",
		})

		res := runHelmDocs(t, cliInvocation{
			Workspace: workspace,
			Dir:       chartsDir,
			Args:      []string{"-c", "."},
		})
		requireExitZero(t, res)

		activeReadme := readTextFile(t, filepath.Join(chartsDir, "active", "README.md"))
		assert.Contains(t, activeReadme, "activeChartKey")
		requireFileNotExists(t, filepath.Join(chartsDir, "ignored-dir", "README.md"))
		requireFileNotExists(t, filepath.Join(chartsDir, "ignored-file", "README.md"))
	})

	t.Run("non_git_directory_uses_custom_ignore_file", func(t *testing.T) {
		workspace := newWorkspace(t)
		writeTextFile(t, filepath.Join(workspace, ".myignore"), "skipped\n")

		writeChartFixture(t, filepath.Join(workspace, "kept"), chartFixture{
			Name:       "kept",
			ValuesYAML: "# keptChartKey -- Kept chart value\nkeptChartKey: visible\n",
		})
		writeChartFixture(t, filepath.Join(workspace, "skipped"), chartFixture{
			Name:       "skipped",
			ValuesYAML: "# skippedChartKey -- Skipped chart value\nskippedChartKey: hidden\n",
		})

		res := runHelmDocs(t, cliInvocation{
			Workspace: workspace,
			Dir:       workspace,
			Args:      []string{"-c", ".", "-i", ".myignore"},
		})
		requireExitZero(t, res)

		keptReadme := readTextFile(t, filepath.Join(workspace, "kept", "README.md"))
		assert.Contains(t, keptReadme, "keptChartKey")
		requireFileNotExists(t, filepath.Join(workspace, "skipped", "README.md"))
	})
}

func TestUmbrellaFileDependencyAliasAndSelectiveGeneration(t *testing.T) {
	workspace := newWorkspace(t)
	umbrellaRelPath := filepath.Join("charts", "umbrella")
	siblingRelPath := filepath.Join("charts", "sibling-sub")

	writeChartFixture(t, filepath.Join(workspace, umbrellaRelPath), chartFixture{
		Name:       "umbrella",
		ValuesYAML: "# parentOnlyKey -- Umbrella chart value\nparentOnlyKey: from-umbrella\n",
		Dependencies: []chartDependencyFixture{
			{
				Name:       "sibling-sub",
				Version:    "0.1.0",
				Repository: "file://../sibling-sub",
				Alias:      "sib",
			},
		},
	})
	writeChartFixture(t, filepath.Join(workspace, siblingRelPath), chartFixture{
		Name:       "sibling-sub",
		ValuesYAML: "# siblingOnlyKey -- Sibling subchart value\nsiblingOnlyKey: from-sibling\n",
	})

	res := runHelmDocs(t, cliInvocation{
		Workspace: workspace,
		Dir:       workspace,
		Args:      []string{"-c", "charts", "-u", "-g", umbrellaRelPath},
	})
	requireExitZero(t, res)

	umbrellaReadme := readTextFile(t, filepath.Join(workspace, umbrellaRelPath, "README.md"))
	valuesSection := extractValuesSection(t, umbrellaReadme)
	assert.Contains(t, valuesSection, "parentOnlyKey")
	assert.Contains(t, valuesSection, "sib.siblingOnlyKey")
	requireFileNotExists(t, filepath.Join(workspace, siblingRelPath, "README.md"))
}

func TestStrictValidationRespectsChartSelection(t *testing.T) {
	parent := filepath.Join("charts", "parent")
	child := filepath.Join("charts", "child")
	const sentinel = "previous README\n"

	for _, tc := range []struct {
		name     string
		flags    []string
		wantLint bool
	}{
		{name: "selected parent", flags: []string{"-g", parent}},
		{name: "selected child fails", flags: []string{"-g", parent + "," + child}, wantLint: true},
		{name: "all charts checked when no selection is given", wantLint: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspace := newWorkspace(t)
			writeChartFixture(t, filepath.Join(workspace, parent), chartFixture{
				ValuesYAML: "# -- Parent setting\nenabled: true\n",
				Dependencies: []chartDependencyFixture{{
					Name: "child", Version: "0.1.0", Repository: "file://../child",
				}},
			})
			writeChartFixture(t, filepath.Join(workspace, child), chartFixture{
				ValuesYAML: "# -- Provided by the child chart\nchildSetting: true\n\nundocumented: true\n",
			})
			parentReadme := filepath.Join(workspace, parent, "README.md")
			childReadme := filepath.Join(workspace, child, "README.md")
			writeTextFile(t, parentReadme, sentinel)
			writeTextFile(t, childReadme, sentinel)

			res := runHelmDocs(t, cliInvocation{
				Workspace: workspace,
				Dir:       workspace,
				Args:      append([]string{"-c", "charts", "-u", "-x"}, tc.flags...),
			})
			assert.Equal(t, sentinel, readTextFile(t, childReadme))
			if tc.wantLint {
				requireNaturalNonZeroExit(t, res)
				assert.Equal(t, 1, res.ExitCode)
				assert.Contains(t, res.Stderr, "chart child: values without documentation:")
				assert.Contains(t, res.Stderr, "undocumented")
				assert.Equal(t, sentinel, readTextFile(t, parentReadme))
				return
			}

			requireExitZero(t, res)
			valuesSection := extractValuesSection(t, readTextFile(t, parentReadme))
			assert.Contains(t, valuesSection, "| enabled | bool | `true` | Parent setting |")
			assert.Contains(t, valuesSection, "| child.childSetting | bool | `true` | Provided by the child chart |")
			assert.Contains(t, valuesSection, "| child.undocumented | bool | `true` |  |")
		})
	}
}

func TestCustomValuesOutputAndFilter(t *testing.T) {
	workspace := newWorkspace(t)
	chartDir := filepath.Join(workspace, "custom-io-chart")
	const sentinelReadme = "default-readme-sentinel\n"

	writeChartFixture(t, chartDir, chartFixture{
		Name:       "custom-io-chart",
		ValuesYAML: "# defaultFileOnlyKey -- Value only in default values.yaml\ndefaultFileOnlyKey: default-only\n",
	})
	writeTextFile(t, filepath.Join(chartDir, "values-prod.yaml"),
		"# prodDocumentedKey -- Documented production value\nprodDocumentedKey: prod-enabled\n\nprodUndocumentedKey: omit-me\n")
	writeTextFile(t, filepath.Join(chartDir, "README.md"), sentinelReadme)

	outputSubdir := filepath.Join(chartDir, "docs")
	require.NoError(t, os.MkdirAll(outputSubdir, 0o750), "pre-create docs output subdirectory")

	outputRelPath := filepath.Join("docs", "REFERENCE.md")
	res := runHelmDocs(t, cliInvocation{
		Workspace: workspace,
		Dir:       chartDir,
		Args: []string{
			"-c", ".",
			"-f", "values-prod.yaml",
			"-o", outputRelPath,
			"--ignore-non-descriptions",
		},
	})
	requireExitZero(t, res)

	referenceDoc := readTextFile(t, filepath.Join(chartDir, outputRelPath))
	assert.Contains(t, referenceDoc, "prodDocumentedKey")
	assert.NotContains(t, referenceDoc, "defaultFileOnlyKey")
	assert.NotContains(t, referenceDoc, "prodUndocumentedKey")
	assert.Equal(t, sentinelReadme, readTextFile(t, filepath.Join(chartDir, "README.md")))
}

func TestCLIInvalidInputAndNonZeroExit(t *testing.T) {
	t.Run("invalid_strict_ignore_regex_exits_nonzero_without_writing_readme", func(t *testing.T) {
		workspace := newWorkspace(t)
		chartDir := filepath.Join(workspace, "valid-chart")
		writeChartFixture(t, chartDir, chartFixture{
			Name:       "valid-chart",
			ValuesYAML: "# sampleKey -- Sample documented key\nsampleKey: ok\n",
		})

		res := runHelmDocs(t, cliInvocation{
			Workspace: workspace,
			Dir:       chartDir,
			Args:      []string{"-c", ".", "-z", "[unclosed"},
		})
		requireNaturalNonZeroExit(t, res)
		assert.Contains(t, res.Stderr, "[unclosed")
		requireFileNotExists(t, filepath.Join(chartDir, "README.md"))
	})

	t.Run("missing_chart_search_root_exits_nonzero_without_creating_paths", func(t *testing.T) {
		workspace := newWorkspace(t)
		missingRoot := filepath.Join(workspace, "missing-root")

		res := runHelmDocs(t, cliInvocation{
			Workspace: workspace,
			Dir:       workspace,
			Args:      []string{"-c", missingRoot},
		})
		requireNaturalNonZeroExit(t, res)
		assert.Contains(t, res.Stderr, "missing-root")
		requireFileNotExists(t, missingRoot)

		entries, err := os.ReadDir(workspace)
		require.NoError(t, err)
		assert.Empty(t, entries, "expected no files or directories created in workspace")
	})
}
