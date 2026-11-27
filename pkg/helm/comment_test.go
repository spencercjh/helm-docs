package helm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCommentSkipsDoubleHashLines(t *testing.T) {
	for _, tc := range []struct {
		name, comments, key, description string
	}{
		{"anonymous", "# -- Context\n## internal\n# for notifications", "", "Context for notifications"},
		{"named", "  ## other -- internal\n  # context -- Context\n  ## internal\n  ##\n  # for notifications", "context", "Context for notifications"},
		{"raw", "  # context -- Context\n  # @raw\n  ## internal\n  #   indented example", "context", "Context\n  indented example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key, doc := ParseComment(strings.Split(tc.comments, "\n"))
			assert.Equal(t, tc.key, key)
			assert.Equal(t, tc.description, doc.Description)
		})
	}
}
