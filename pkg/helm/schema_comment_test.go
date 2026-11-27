package helm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCommentSkipsSchemaAnnotations(t *testing.T) {
	for _, tc := range []struct {
		name, comments, key, description string
	}{
		{"inline", "# -- Pull policy\n# @schema enum: [Always, Never]\n# Use Never offline", "", "Pull policy Use Never offline"},
		{"block", "  # policy -- Pull policy\n  # @schema\n  # enum: [Always, Never]\n  # @schema\n  # Use Never offline", "policy", "Pull policy Use Never offline"},
		{"raw", "# -- Pull policy\n# @raw\n# @schema\n# type: string\n# @schema\n#   Example: Never", "", "Pull policy\n  Example: Never"},
		{"unpaired", "# -- Pull policy\n# @schema\n# Use Never offline", "", "Pull policy Use Never offline"},
		{"similar prefix", "# -- Pull policy\n# @schema-related information", "", "Pull policy @schema-related information"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key, doc := ParseComment(strings.Split(tc.comments, "\n"))
			assert.Equal(t, tc.key, key)
			assert.Equal(t, tc.description, doc.Description)
		})
	}
}
