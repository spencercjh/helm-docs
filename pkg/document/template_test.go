package document

import (
	"strings"
	"testing"

	"github.com/norwoodj/helm-docs/pkg/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderDefaultWithBackticks(t *testing.T) {
	values := parseYamlValues("functionTemplate: \"my_func(`value`)\"\nlongerRun: \"`value``\"\nplain: hello")
	rows, err := getSortedValuesTableRows(values, nil)
	require.NoError(t, err)
	tpl, err := newChartDocumentationTemplate(helm.ChartDocumentationInfo{}, ".", nil, "")
	require.NoError(t, err)
	data := chartTemplateData{Values: rows}

	var markdown strings.Builder
	require.NoError(t, tpl.ExecuteTemplate(&markdown, "chart.valuesTableMd", data))
	assert.Contains(t, markdown.String(), "| functionTemplate | string | ``\"my_func(`value`)\"`` |")
	assert.Contains(t, markdown.String(), "| longerRun | string | ```\"`value``\"``` |")
	assert.Contains(t, markdown.String(), "| plain | string | `\"hello\"` |")

	var html strings.Builder
	require.NoError(t, tpl.ExecuteTemplate(&html, "chart.valuesTableHtml", data))
	assert.Contains(t, html.String(), "<pre lang=\"json\">\n\"my_func(`value`)\"\n</pre>")
	assert.Contains(t, html.String(), "<pre lang=\"json\">\n\"`value``\"\n</pre>")
}

func TestGetDocumentationTemplate(t *testing.T) {
	tpl, err := getDocumentationTemplate(".", ".", []string{"testdata/nonexistent.md.gotmpl"})

	require.NoError(t, err)
	assert.Equal(t, defaultDocumentationTemplate, tpl)
}

func TestGetDocumentationTemplate_LoadDefaultOnNotFound(t *testing.T) {
	tpl, err := getDocumentationTemplate(".", ".", []string{
		"testdata/README.md.gotmpl",
		"testdata/nonexistent.md.gotmpl",
		"testdata/README2.md.gotmpl",
	})

	const expected = "hello\nhello again\n" + defaultDocumentationTemplate

	require.NoError(t, err)
	assert.Equal(t, expected, tpl)
}
