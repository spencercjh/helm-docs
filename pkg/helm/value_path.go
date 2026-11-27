package helm

import (
	"fmt"
	"strings"
)

// AppendValueKey adds a mapping key using the path syntax of the values table.
func AppendValueKey(prefix, key string) string {
	if strings.ContainsAny(key, ". ") {
		key = `"` + key + `"`
	}
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// AppendValueIndex adds a list index using the path syntax of the values table.
func AppendValueIndex(prefix string, index int) string {
	return fmt.Sprintf("%s[%d]", prefix, index)
}
