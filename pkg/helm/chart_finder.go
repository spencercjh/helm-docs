package helm

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/norwoodj/helm-docs/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func FindChartDirectories(chartSearchRoot string) ([]string, error) {
	chartDirs := make([]string, 0)

	if chartSearchRoot == "" {
		return chartDirs, errors.New("chart search root is empty")
	}
	absoluteRoot, err := filepath.Abs(chartSearchRoot)
	if err != nil {
		return chartDirs, err
	}
	// Resolve the root so directory discovery and ignore rules see the same paths
	// as they would when the chart is reached without a symlink.
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return chartDirs, err
	}

	ignoreContext := util.NewIgnoreContext(viper.GetString("ignore-file"))
	err = filepath.Walk(resolvedRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && ignoreContext.ShouldIgnore(path, info) {
			log.Debugf("Ignoring directory %s", path)
			return filepath.SkipDir
		}

		if filepath.Base(path) == "Chart.yaml" {
			if ignoreContext.ShouldIgnore(path, info) {
				log.Debugf("Ignoring chart file %s", path)
				return nil
			}
			relativeChartDir, err := filepath.Rel(resolvedRoot, filepath.Dir(path))

			if err != nil {
				return err
			}

			chartDirs = append(chartDirs, relativeChartDir)
		}

		return nil
	})

	return chartDirs, err
}
