package e2e_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	injectedVersion     = "1.2.3"
	cliExecutionTimeout = 20 * time.Second
)

var (
	helmDocsBinPath    string
	emptyGitConfigPath string
)

func TestMain(m *testing.M) {
	os.Exit(runTestMain(m))
}

func runTestMain(m *testing.M) int {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to locate module root: %v\n", err)
		return 1
	}

	buildDir, err := os.MkdirTemp("", "helm-docs-e2e-build-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temporary build directory: %v\n", err)
		return 1
	}
	defer func() {
		_ = os.RemoveAll(buildDir)
	}()

	emptyGitConfigPath = filepath.Join(buildDir, "empty.gitconfig")
	if err := os.WriteFile(emptyGitConfigPath, nil, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create isolated gitconfig: %v\n", err)
		return 1
	}

	binName := "helm-docs"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	helmDocsBinPath = filepath.Join(buildDir, binName)

	buildCmd := exec.Command(
		"go",
		"build",
		"-ldflags",
		"-X main.version="+injectedVersion,
		"-o",
		helmDocsBinPath,
		"./cmd/helm-docs",
	)
	buildCmd.Dir = moduleRoot

	var buildOutput bytes.Buffer
	buildCmd.Stdout = &buildOutput
	buildCmd.Stderr = &buildOutput
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build helm-docs binary for E2E tests: %v\n%s\n", err, buildOutput.String())
		return 1
	}

	return m.Run()
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return filepath.EvalSymlinks(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above current working directory")
		}
		dir = parent
	}
}

// newWorkspace creates a temporary root directory with symlinks resolved so paths match
// canonical outputs such as `git rev-parse --show-toplevel` on macOS (/private/var/...).
func newWorkspace(t *testing.T) string {
	t.Helper()

	resolved, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err, "resolve symlinks for temporary workspace")
	return resolved
}

type cliInvocation struct {
	Workspace string
	Dir       string
	Args      []string
	Env       map[string]string
}

type cliResult struct {
	Command   []string
	Dir       string
	Workspace string
	ExtraEnv  map[string]string
	Ceiling   string
	ExitCode  int
	Exited    bool
	TimedOut  bool
	StartErr  error
	RunErr    error
	Stdout    string
	Stderr    string
}

func buildIsolatedEnv(workspace string, extraEnv map[string]string) ([]string, string) {
	ceiling := filepath.Dir(workspace)
	filtered := make([]string, 0, len(os.Environ())+len(extraEnv)+3)
	overrides := make(map[string]string, len(extraEnv)+3)
	overrides["GIT_CONFIG_NOSYSTEM"] = "1"
	overrides["GIT_CONFIG_GLOBAL"] = emptyGitConfigPath
	overrides["GIT_CEILING_DIRECTORIES"] = ceiling
	for k, v := range extraEnv {
		overrides[k] = v
	}

	overrideUpper := make(map[string]bool, len(overrides))
	for k := range overrides {
		overrideUpper[strings.ToUpper(k)] = true
	}

	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(key)
		if strings.HasPrefix(upper, "HELM_DOCS_") || strings.HasPrefix(upper, "GIT_") || overrideUpper[upper] {
			continue
		}
		filtered = append(filtered, entry)
	}

	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		filtered = append(filtered, k+"="+overrides[k])
	}

	return filtered, ceiling
}

func runHelmDocs(t *testing.T, inv cliInvocation) cliResult {
	t.Helper()

	require.NotEmpty(t, inv.Workspace, "cliInvocation.Workspace must be set")
	workDir := inv.Dir
	if workDir == "" {
		workDir = inv.Workspace
	}

	relCheck, relErr := filepath.Rel(inv.Workspace, workDir)
	require.NoError(t, relErr, "compute workDir relative to workspace")
	require.False(t, relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(filepath.Separator)),
		"workDir %q must stay inside temporary workspace %q", workDir, inv.Workspace)

	return execSubprocess(t, inv.Workspace, workDir, helmDocsBinPath, inv.Args, inv.Env)
}

func initGitRepo(t *testing.T, workspace, repoDir string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(repoDir, 0o750), "create git repository directory")
	res := execSubprocess(t, workspace, repoDir, "git", []string{"init", "--template="}, nil)
	requireExitZero(t, res)
}

func execSubprocess(t *testing.T, workspace, dir, binary string, args []string, extraEnv map[string]string) cliResult {
	t.Helper()

	env, ceiling := buildIsolatedEnv(workspace, extraEnv)
	ctx, cancel := context.WithTimeout(context.Background(), cliExecutionTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = env

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()
	res := cliResult{
		Command:   append([]string{binary}, args...),
		Dir:       dir,
		Workspace: workspace,
		ExtraEnv:  cloneMap(extraEnv),
		Ceiling:   ceiling,
		RunErr:    runErr,
		TimedOut:  ctx.Err() != nil,
		Stdout:    stdoutBuf.String(),
		Stderr:    stderrBuf.String(),
	}

	if runErr == nil {
		res.ExitCode = 0
		res.Exited = true
	} else {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			res.Exited = exitErr.ProcessState != nil && exitErr.Exited()
		} else {
			res.ExitCode = -1
			res.StartErr = runErr
		}
	}

	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("\n%s", formatDiagnostics(res))
		}
	})

	return res
}

func requireExitZero(t *testing.T, res cliResult) {
	t.Helper()

	require.False(t, res.TimedOut, "subprocess timed out after %s", cliExecutionTimeout)
	require.NoError(t, res.StartErr, "subprocess failed to start")
	require.True(t, res.Exited, "subprocess did not exit cleanly: %v", res.RunErr)
	require.Equal(t, 0, res.ExitCode, "expected exit code 0, got %d", res.ExitCode)
}

func requireNaturalNonZeroExit(t *testing.T, res cliResult) {
	t.Helper()

	require.False(t, res.TimedOut, "expected natural CLI error exit, but subprocess timed out after %s", cliExecutionTimeout)
	require.NoError(t, res.StartErr, "expected subprocess to start cleanly before failing")
	require.Error(t, res.RunErr, "expected non-zero exit error from subprocess")
	require.True(t, res.Exited, "expected subprocess to terminate via normal exit rather than signal: %v", res.RunErr)
	require.Greater(t, res.ExitCode, 0, "expected positive non-zero exit code, got %d", res.ExitCode)
}

func formatDiagnostics(res cliResult) string {
	var b strings.Builder
	b.WriteString("=== E2E Subprocess Diagnostics ===\n")
	fmt.Fprintf(&b, "Command: %q\n", res.Command)
	fmt.Fprintf(&b, "WorkDir: %s\n", res.Dir)
	fmt.Fprintf(&b, "Workspace: %s (GIT_CEILING_DIRECTORIES=%s)\n", res.Workspace, res.Ceiling)
	fmt.Fprintf(&b, "ExplicitEnv: %v\n", formatEnvMap(res.ExtraEnv))
	fmt.Fprintf(&b, "Status: exitCode=%d exited=%t timedOut=%t startErr=%v runErr=%v\n",
		res.ExitCode, res.Exited, res.TimedOut, res.StartErr, res.RunErr)
	b.WriteString("--- STDOUT ---\n")
	b.WriteString(res.Stdout)
	if !strings.HasSuffix(res.Stdout, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("--- STDERR ---\n")
	b.WriteString(res.Stderr)
	if !strings.HasSuffix(res.Stderr, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("--- WORKSPACE FILES ---\n")
	b.WriteString(snapshotWorkspaceTree(res.Workspace))
	b.WriteString("==================================")
	return b.String()
}

func formatEnvMap(env map[string]string) string {
	if len(env) == 0 {
		return "(none)"
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%q", k, env[k]))
	}
	return strings.Join(pairs, ", ")
}

func snapshotWorkspaceTree(root string) string {
	if _, err := os.Stat(root); err != nil {
		return fmt.Sprintf("(workspace %s unavailable: %v)\n", root, err)
	}

	var lines []string
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			lines = append(lines, fmt.Sprintf("  [err] %s: %v", path, err))
			return nil
		}
		if path == root {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				lines = append(lines, fmt.Sprintf("  [dir]  %s/ (git metadata)", filepath.ToSlash(rel)))
				return filepath.SkipDir
			}
			lines = append(lines, fmt.Sprintf("  [dir]  %s/", filepath.ToSlash(rel)))
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			lines = append(lines, fmt.Sprintf("  [file] %s (stat err: %v)", filepath.ToSlash(rel), infoErr))
			return nil
		}
		lines = append(lines, fmt.Sprintf("  [file] %s (%d bytes)", filepath.ToSlash(rel), info.Size()))
		return nil
	})
	if walkErr != nil {
		lines = append(lines, fmt.Sprintf("  [walk-err] %v", walkErr))
	}
	if len(lines) == 0 {
		return "  (empty)\n"
	}
	return strings.Join(lines, "\n") + "\n"
}

func cloneMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

type chartDependencyFixture struct {
	Name       string
	Version    string
	Repository string
	Alias      string
}

type chartFixture struct {
	Name         string
	Description  string
	Version      string
	AppVersion   string
	ValuesYAML   string
	Dependencies []chartDependencyFixture
}

func writeChartFixture(t *testing.T, chartDir string, spec chartFixture) {
	t.Helper()

	name := spec.Name
	if name == "" {
		name = filepath.Base(chartDir)
	}
	version := spec.Version
	if version == "" {
		version = "0.1.0"
	}
	description := spec.Description
	if description == "" {
		description = "Generated E2E test chart " + name
	}

	var chartYAML strings.Builder
	chartYAML.WriteString("apiVersion: v2\n")
	fmt.Fprintf(&chartYAML, "name: %s\n", name)
	fmt.Fprintf(&chartYAML, "description: %s\n", description)
	chartYAML.WriteString("type: application\n")
	fmt.Fprintf(&chartYAML, "version: %s\n", version)
	if spec.AppVersion != "" {
		fmt.Fprintf(&chartYAML, "appVersion: %q\n", spec.AppVersion)
	}
	if len(spec.Dependencies) > 0 {
		chartYAML.WriteString("dependencies:\n")
		for _, dep := range spec.Dependencies {
			depVersion := dep.Version
			if depVersion == "" {
				depVersion = "0.1.0"
			}
			fmt.Fprintf(&chartYAML, "  - name: %s\n", dep.Name)
			fmt.Fprintf(&chartYAML, "    version: %s\n", depVersion)
			if dep.Repository != "" {
				fmt.Fprintf(&chartYAML, "    repository: %s\n", dep.Repository)
			}
			if dep.Alias != "" {
				fmt.Fprintf(&chartYAML, "    alias: %s\n", dep.Alias)
			}
		}
	}

	writeTextFile(t, filepath.Join(chartDir, "Chart.yaml"), chartYAML.String())
	if spec.ValuesYAML != "" {
		writeTextFile(t, filepath.Join(chartDir, "values.yaml"), spec.ValuesYAML)
	}
}

func writeTextFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750), "create parent directory for %s", path)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600), "write file %s", path)
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err, "expected file %s to exist and be readable", path)
	return string(data)
}

func requireFileNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err), "expected path %s not to exist, got stat err=%v", path, err)
}

// extractValuesSection returns the Markdown body under `## Values` up to the next H2 heading
// or version footer separator so assertions target rendered value rows rather than metadata tables.
func extractValuesSection(t *testing.T, markdown string) string {
	t.Helper()

	const marker = "## Values"
	idx := strings.Index(markdown, marker)
	require.NotEqual(t, -1, idx, "expected %q section in rendered markdown:\n%s", marker, markdown)

	rest := markdown[idx+len(marker):]
	for _, stop := range []string{"\n## ", "\n----------------------------------------------"} {
		if end := strings.Index(rest, stop); end != -1 {
			rest = rest[:end]
		}
	}
	return rest
}
