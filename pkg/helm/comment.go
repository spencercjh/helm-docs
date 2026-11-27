package helm

import (
	"regexp"
	"slices"
	"strings"
)

const (
	PrefixComment = "# --"
)

// yaml.v3 may attach a standalone section and the next value description to
// the same HeadComment. A physical blank line ends the section; a bare "#"
// is a blank line within its Markdown body.
func withoutExtraSections(lines []string) []string {
	var valueLines []string
	start := 0
	for i := 0; i < len(lines); i++ {
		if !extraRegex.MatchString(lines[i]) {
			continue
		}
		valueLines = append(valueLines, lines[start:i]...)
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			i++
		}
		start = i
	}
	if start == 0 {
		return lines
	}
	return append(valueLines, lines[start:]...)
}

func ParseComment(commentLines []string) (string, ChartValueDescription) {
	commentLines = withoutSchemaAnnotations(commentLines)
	if len(commentLines) == 0 {
		return "", ChartValueDescription{}
	}
	var valueKey string
	var c ChartValueDescription
	docStartIdx := -1

	commentLines = withoutExtraSections(commentLines)
	if len(commentLines) == 0 {
		return valueKey, c
	}

	// Work around https://github.com/norwoodj/helm-docs/issues/96 by considering only
	// the last "group" of comment lines starting with '# --'.
	lastIndex := 0
	for i, v := range commentLines {
		if strings.HasPrefix(v, PrefixComment) {
			lastIndex = i
		}
	}
	if lastIndex > 0 {
		// If there's a non-zero last index, consider that alone.
		return ParseComment(commentLines[lastIndex:])
	}

	for i := range commentLines {
		if isInternalComment(commentLines[i]) {
			continue
		}
		match := valuesDescriptionRegex.FindStringSubmatch(commentLines[i])
		if len(match) < 3 {
			continue
		}

		valueKey = match[1]
		c.Description = match[2]
		docStartIdx = i
		break
	}
	if docStartIdx < 0 {
		return "", ChartValueDescription{}
	}

	valueTypeMatch := valueTypeRegex.FindStringSubmatch(c.Description)
	if len(valueTypeMatch) > 0 && valueTypeMatch[1] != "" {
		c.ValueType = valueTypeMatch[1]
		c.Description = valueTypeMatch[2]
	}

	var isRaw = false

	for _, line := range commentLines[docStartIdx+1:] {
		if isInternalComment(line) {
			continue
		}
		rawFlagMatch := rawDescriptionRegex.FindStringSubmatch(line)
		defaultCommentMatch := defaultValueRegex.FindStringSubmatch(line)
		notationTypeCommentMatch := valueNotationTypeRegex.FindStringSubmatch(line)
		sectionCommentMatch := sectionRegex.FindStringSubmatch(line)

		if !isRaw && len(rawFlagMatch) == 1 {
			isRaw = true
			continue
		}

		if len(defaultCommentMatch) > 1 {
			c.Default = defaultCommentMatch[1]
			continue
		}

		if len(notationTypeCommentMatch) > 1 {
			c.NotationType = notationTypeCommentMatch[1]
			continue
		}

		if len(sectionCommentMatch) > 1 {
			c.Section = sectionCommentMatch[1]
			continue
		}

		commentContinuationMatch := commentContinuationRegex.FindStringSubmatch(line)

		if isRaw {

			if len(commentContinuationMatch) > 1 {
				c.Description += "\n" + commentContinuationMatch[2]
			}
			continue
		} else {
			if len(commentContinuationMatch) > 1 {
				c.Description += " " + commentContinuationMatch[2]
			}
			continue
		}
	}
	return valueKey, c
}

var schemaAnnotationRegex = regexp.MustCompile(`^[ \t]*#[ \t]+@schema(?:[ \t]+(.*))?$`)

func withoutSchemaAnnotations(lines []string) []string {
	first := slices.IndexFunc(lines, schemaAnnotationRegex.MatchString)
	if first < 0 {
		return lines
	}

	filtered := slices.Clone(lines[:first])
	for i := first; i < len(lines); i++ {
		match := schemaAnnotationRegex.FindStringSubmatch(lines[i])
		if match == nil {
			filtered = append(filtered, lines[i])
			continue
		}
		if strings.TrimSpace(match[1]) == "" {
			// Skip a multiline block only when it has a closing marker.
			end := slices.IndexFunc(lines[i+1:], func(line string) bool {
				closing := schemaAnnotationRegex.FindStringSubmatch(line)
				return closing != nil && strings.TrimSpace(closing[1]) == ""
			})
			if end >= 0 {
				i += end + 1
			}
		}
	}
	return filtered
}
