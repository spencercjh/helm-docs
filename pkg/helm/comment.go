package helm

import (
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
	var valueKey string
	var c ChartValueDescription
	var docStartIdx int

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
		match := valuesDescriptionRegex.FindStringSubmatch(commentLines[i])
		if len(match) < 3 {
			continue
		}

		valueKey = match[1]
		c.Description = match[2]
		docStartIdx = i
		break
	}

	valueTypeMatch := valueTypeRegex.FindStringSubmatch(c.Description)
	if len(valueTypeMatch) > 0 && valueTypeMatch[1] != "" {
		c.ValueType = valueTypeMatch[1]
		c.Description = valueTypeMatch[2]
	}

	var isRaw = false

	for _, line := range commentLines[docStartIdx+1:] {
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
