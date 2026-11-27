package document

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/norwoodj/helm-docs/pkg/util"

	log "github.com/sirupsen/logrus"

	"github.com/norwoodj/helm-docs/pkg/helm"
)

const defaultDocumentationTemplate = `{{ template "chart.header" . }}
{{ template "chart.deprecationWarning" . }}

{{ template "chart.badgesSection" . }}

{{ template "chart.description" . }}

{{ template "chart.homepageLine" . }}

{{ template "chart.maintainersSection" . }}

{{ template "chart.sourcesSection" . }}

{{ template "chart.requirementsSection" . }}

{{ template "chart.valuesSectionMd" . }}

{{- if not .SkipVersionFooter }}
{{ template "helm-docs.versionFooter" . }}
{{- end }}
`

func getNameTemplate() string {
	nameBuilder := strings.Builder{}
	nameBuilder.WriteString(`{{ define "chart.name" }}`)
	nameBuilder.WriteString("{{ .Name }}")
	nameBuilder.WriteString("{{ end }}")

	return nameBuilder.String()
}

func getHeaderTemplate() string {
	headerTemplateBuilder := strings.Builder{}
	headerTemplateBuilder.WriteString(`{{ define "chart.header" }}`)
	headerTemplateBuilder.WriteString("# {{ .Name }}\n")
	headerTemplateBuilder.WriteString("{{ end }}")

	return headerTemplateBuilder.String()
}

func getDeprecatedTemplate() string {
	deprecatedTemplateBuilder := strings.Builder{}
	deprecatedTemplateBuilder.WriteString(`{{ define "chart.deprecationWarning" }}`)
	deprecatedTemplateBuilder.WriteString("{{ if .Deprecated }}> **:exclamation: This Helm Chart is deprecated!**{{ end }}")
	deprecatedTemplateBuilder.WriteString("{{ end }}")

	return deprecatedTemplateBuilder.String()
}

func getVersionTemplates(badgeStyle string) string {
	versionBuilder := strings.Builder{}
	versionBuilder.WriteString(`{{ define "chart.version" }}{{ .Version }}{{ end }}\n`)
	versionBuilder.WriteString(`{{ define "chart.versionBadge" }}`)
	versionBuilder.WriteString(fmt.Sprintf(`![Version: {{ .Version }}](https://img.shields.io/badge/Version-{{ .Version | replace "-" "--" }}-informational?style=%s) `, badgeStyle))
	versionBuilder.WriteString("{{ end }}")

	return versionBuilder.String()
}

func getTypeTemplate(badgeStyle string) string {
	typeBuilder := strings.Builder{}
	typeBuilder.WriteString(`{{ define "chart.type" }}{{ .Type }}{{ end }}\n`)
	typeBuilder.WriteString(`{{ define "chart.typeBadge" }}`)
	typeBuilder.WriteString(fmt.Sprintf("{{ if .Type }}![Type: {{ .Type }}](https://img.shields.io/badge/Type-{{ .Type }}-informational?style=%s) {{ end }}", badgeStyle))
	typeBuilder.WriteString("{{ end }}")

	return typeBuilder.String()
}

func getAppVersionTemplate(badgeStyle string) string {
	appVersionBuilder := strings.Builder{}
	appVersionBuilder.WriteString(`{{ define "chart.appVersion" }}{{ .AppVersion }}{{ end }}\n`)
	appVersionBuilder.WriteString(`{{ define "chart.appVersionBadge" }}`)
	appVersionBuilder.WriteString(fmt.Sprintf(`{{ if .AppVersion }}![AppVersion: {{ .AppVersion }}](https://img.shields.io/badge/AppVersion-{{ .AppVersion | replace "-" "--" }}-informational?style=%s) {{ end }}`, badgeStyle))
	appVersionBuilder.WriteString("{{ end }}")

	return appVersionBuilder.String()
}

func getBadgesTemplates() string {
	badgeBuilder := strings.Builder{}
	badgeBuilder.WriteString(`{{ define "chart.badgesSection" }}`)
	badgeBuilder.WriteString(`{{ template "chart.versionBadge" . }}{{ template "chart.typeBadge" . }}{{ template "chart.appVersionBadge" . }}`)
	badgeBuilder.WriteString("{{ end }}")

	return badgeBuilder.String()
}

func getDescriptionTemplate() string {
	descriptionBuilder := strings.Builder{}
	descriptionBuilder.WriteString(`{{ define "chart.description" }}`)
	descriptionBuilder.WriteString("{{ if .Description }}{{ .Description }}{{ end }}")
	descriptionBuilder.WriteString("{{ end }}")

	return descriptionBuilder.String()
}

func getHomepageTemplate() string {
	homepageBuilder := strings.Builder{}
	homepageBuilder.WriteString(`{{ define "chart.homepage" }}{{ .Home }}{{ end }}\n`)
	homepageBuilder.WriteString(`{{ define "chart.homepageLine" }}`)
	homepageBuilder.WriteString("{{ if .Home }}**Homepage:** <{{ .Home }}>{{ end }}")
	homepageBuilder.WriteString("{{ end }}")

	return homepageBuilder.String()
}

func getMaintainersTemplate() string {
	maintainerBuilder := strings.Builder{}
	maintainerBuilder.WriteString(`{{ define "chart.maintainersHeader" }}## Maintainers{{ end }}`)

	maintainerBuilder.WriteString(`{{ define "chart.maintainersTable" }}`)
	maintainerBuilder.WriteString("| Name | Email | URL |\n")
	maintainerBuilder.WriteString("| ---- | ------ | --- |\n")
	maintainerBuilder.WriteString("  {{- range .Maintainers }}")
	maintainerBuilder.WriteString("\n| {{ .Name }} | {{ if .Email }}<{{ .Email }}>{{ end }} | {{ if .Url }}<{{ .Url }}>{{ end }} |")
	maintainerBuilder.WriteString("  {{- end }}")
	maintainerBuilder.WriteString("{{ end }}")

	maintainerBuilder.WriteString(`{{ define "chart.maintainersSection" }}`)
	maintainerBuilder.WriteString("{{ if .Maintainers }}")
	maintainerBuilder.WriteString(`{{ template "chart.maintainersHeader" . }}`)
	maintainerBuilder.WriteString("\n\n")
	maintainerBuilder.WriteString(`{{ template "chart.maintainersTable" . }}`)
	maintainerBuilder.WriteString("{{ end }}")
	maintainerBuilder.WriteString("{{ end }}")

	return maintainerBuilder.String()
}

func getSourceLinkTemplates() string {
	sourceLinkBuilder := strings.Builder{}
	sourceLinkBuilder.WriteString(`{{ define "chart.sourcesHeader" }}## Source Code{{ end}}`)

	sourceLinkBuilder.WriteString(`{{ define "chart.sourcesList" }}`)
	sourceLinkBuilder.WriteString("{{- range .Sources }}")
	sourceLinkBuilder.WriteString("\n* <{{ . }}>")
	sourceLinkBuilder.WriteString("{{- end }}")
	sourceLinkBuilder.WriteString("{{ end }}")

	sourceLinkBuilder.WriteString(`{{ define "chart.sourcesSection" }}`)
	sourceLinkBuilder.WriteString("{{ if .Sources }}")
	sourceLinkBuilder.WriteString(`{{ template "chart.sourcesHeader" . }}`)
	sourceLinkBuilder.WriteString("\n")
	sourceLinkBuilder.WriteString(`{{ template "chart.sourcesList" . }}`)
	sourceLinkBuilder.WriteString("{{ end }}")
	sourceLinkBuilder.WriteString("{{ end }}")

	return sourceLinkBuilder.String()
}

func getRequirementsTableTemplates() string {
	requirementsSectionBuilder := strings.Builder{}
	requirementsSectionBuilder.WriteString(`{{ define "chart.requirementsHeader" }}## Requirements{{ end }}`)

	requirementsSectionBuilder.WriteString(`{{ define "chart.kubeVersion" }}{{ .KubeVersion }}{{ end }}\n`)
	requirementsSectionBuilder.WriteString(`{{ define "chart.kubeVersionLine" }}`)
	requirementsSectionBuilder.WriteString("{{ if .KubeVersion }}Kubernetes: `{{ .KubeVersion }}`{{ end }}")
	requirementsSectionBuilder.WriteString("{{ end }}")

	requirementsSectionBuilder.WriteString(`{{ define "chart.requirementsTable" }}`)
	requirementsSectionBuilder.WriteString("| Repository | Name | Version |\n")
	requirementsSectionBuilder.WriteString("|------------|------|---------|")
	requirementsSectionBuilder.WriteString("  {{- range .Dependencies }}")
	requirementsSectionBuilder.WriteString("    {{- if .Alias }}")
	requirementsSectionBuilder.WriteString("\n| {{ .Repository }} | {{ .Alias }}({{ .Name }}) | {{ .Version }} |")
	requirementsSectionBuilder.WriteString("    {{- else }}")
	requirementsSectionBuilder.WriteString("\n| {{ .Repository }} | {{ .Name }} | {{ .Version }} |")
	requirementsSectionBuilder.WriteString("    {{- end }}")
	requirementsSectionBuilder.WriteString("  {{- end }}")
	requirementsSectionBuilder.WriteString("{{ end }}")

	requirementsSectionBuilder.WriteString(`{{ define "chart.requirementsSection" }}`)
	requirementsSectionBuilder.WriteString("{{ if or .Dependencies .KubeVersion }}")
	requirementsSectionBuilder.WriteString(`{{ template "chart.requirementsHeader" . }}`)
	requirementsSectionBuilder.WriteString("\n\n")
	requirementsSectionBuilder.WriteString("{{ if .KubeVersion }}")
	requirementsSectionBuilder.WriteString(`{{ template "chart.kubeVersionLine" . }}`)
	requirementsSectionBuilder.WriteString("\n\n")
	requirementsSectionBuilder.WriteString("{{ end }}")
	requirementsSectionBuilder.WriteString("{{ if .Dependencies }}")
	requirementsSectionBuilder.WriteString(`{{ template "chart.requirementsTable" . }}`)
	requirementsSectionBuilder.WriteString("{{ end }}")
	requirementsSectionBuilder.WriteString("{{ end }}")
	requirementsSectionBuilder.WriteString("{{ end }}")

	return requirementsSectionBuilder.String()
}

func getValuesTableTemplates() string {
	valuesSectionBuilder := strings.Builder{}
	valuesSectionBuilder.WriteString(`{{ define "chart.valuesHeader" }}## Values{{ end }}`)

	valuesSectionBuilder.WriteString(`{{ define "chart.valueKeyColumnRenderMd" }}`)
	valuesSectionBuilder.WriteString("{{ .Key }}")
	valuesSectionBuilder.WriteString("{{ end }}")

	valuesSectionBuilder.WriteString(`{{ define "chart.valueTypeColumnRenderMd" }}`)
	valuesSectionBuilder.WriteString("{{ .Type }}")
	valuesSectionBuilder.WriteString("{{ end }}")

	valuesSectionBuilder.WriteString(`{{ define "chart.valueDefaultColumnRenderMd" }}`)
	valuesSectionBuilder.WriteString("{{ if .Default }}{{ .Default }}{{ else }}{{ .AutoDefault }}{{ end }}")
	valuesSectionBuilder.WriteString("{{ end }}")

	valuesSectionBuilder.WriteString(`{{ define "chart.valueDescriptionColumnRenderMd" }}`)
	valuesSectionBuilder.WriteString("{{ if .Description }}{{ .Description }}{{ else }}{{ .AutoDescription }}{{ end }}")
	valuesSectionBuilder.WriteString("{{ end }}")

	valuesSectionBuilder.WriteString(`{{ define "chart.valuesTable" }}`)
	valuesSectionBuilder.WriteString("{{ if .Sections.Sections }}")
	valuesSectionBuilder.WriteString("{{ range .Sections.Sections }}")
	valuesSectionBuilder.WriteString("\n")
	valuesSectionBuilder.WriteString("\n### {{ .SectionName }}\n")
	valuesSectionBuilder.WriteString("\n")
	valuesSectionBuilder.WriteString("| Key | Type | Default | Description |\n")
	valuesSectionBuilder.WriteString("|-----|------|---------|-------------|\n")
	valuesSectionBuilder.WriteString("  {{- range .SectionItems }}")
	valuesSectionBuilder.WriteString("\n" + `| {{ template "chart.valueKeyColumnRenderMd" . }} | {{ template "chart.valueTypeColumnRenderMd" . }} | {{ template "chart.valueDefaultColumnRenderMd" . }} | {{ template "chart.valueDescriptionColumnRenderMd" . }} |`)
	valuesSectionBuilder.WriteString("  {{- end }}")
	valuesSectionBuilder.WriteString("{{- end }}")
	valuesSectionBuilder.WriteString("{{ if .Sections.DefaultSection.SectionItems}}")
	valuesSectionBuilder.WriteString("\n")
	valuesSectionBuilder.WriteString("\n### {{ .Sections.DefaultSection.SectionName }}\n")
	valuesSectionBuilder.WriteString("\n")
	valuesSectionBuilder.WriteString("| Key | Type | Default | Description |\n")
	valuesSectionBuilder.WriteString("|-----|------|---------|-------------|\n")
	valuesSectionBuilder.WriteString("  {{- range .Sections.DefaultSection.SectionItems }}")
	valuesSectionBuilder.WriteString("\n" + `| {{ template "chart.valueKeyColumnRenderMd" . }} | {{ template "chart.valueTypeColumnRenderMd" . }} | {{ template "chart.valueDefaultColumnRenderMd" . }} | {{ template "chart.valueDescriptionColumnRenderMd" . }} |`)
	valuesSectionBuilder.WriteString("  {{- end }}")
	valuesSectionBuilder.WriteString("{{ end }}")
	valuesSectionBuilder.WriteString("{{ else }}")
	valuesSectionBuilder.WriteString("| Key | Type | Default | Description |\n")
	valuesSectionBuilder.WriteString("|-----|------|---------|-------------|\n")
	valuesSectionBuilder.WriteString("  {{- range .Values }}")
	valuesSectionBuilder.WriteString("\n" + `| {{ template "chart.valueKeyColumnRenderMd" . }} | {{ template "chart.valueTypeColumnRenderMd" . }} | {{ template "chart.valueDefaultColumnRenderMd" . }} | {{ template "chart.valueDescriptionColumnRenderMd" . }} |`)
	valuesSectionBuilder.WriteString("  {{- end }}")
	valuesSectionBuilder.WriteString("{{ end }}")
	valuesSectionBuilder.WriteString("{{ end }}")

	// TODO: Remove this block and rename `chart.valuesTable` to `chart.valuesTableMd` in the future
	valuesSectionBuilder.WriteString(`{{ define "chart.valuesTableMd" }}`)
	valuesSectionBuilder.WriteString(`{{ template "chart.valuesTable" . }}`)
	valuesSectionBuilder.WriteString("{{ end }}")

	valuesSectionBuilder.WriteString(`{{ define "chart.valuesSection" }}`)
	valuesSectionBuilder.WriteString("{{ if .Values }}")
	valuesSectionBuilder.WriteString(`{{ template "chart.valuesHeader" . }}`)
	valuesSectionBuilder.WriteString("\n\n")
	valuesSectionBuilder.WriteString(`{{ template "chart.valuesTableMd" . }}`)
	valuesSectionBuilder.WriteString("{{ end }}")
	valuesSectionBuilder.WriteString("{{ end }}")

	// TODO: Remove this block and rename `chart.valuesTable` to `chart.valuesTableMd` in the future
	valuesSectionBuilder.WriteString(`{{ define "chart.valuesSectionMd" }}`)
	valuesSectionBuilder.WriteString(`{{ template "chart.valuesSection" . }}`)
	valuesSectionBuilder.WriteString("{{ end }}")

	// For HTML tables
	valuesSectionBuilder.WriteString(`{{ define "chart.valueKeyColumnRenderHtml" }}`)
	valuesSectionBuilder.WriteString(`{{ template "chart.valueKeyColumnRenderMd" . }}`)
	valuesSectionBuilder.WriteString("{{ end }}")

	valuesSectionBuilder.WriteString(`{{ define "chart.valueTypeColumnRenderHtml" }}`)
	valuesSectionBuilder.WriteString(`{{ template "chart.valueTypeColumnRenderMd" . }}`)
	valuesSectionBuilder.WriteString("{{ end }}")

	valuesSectionBuilder.WriteString(`{{ define "chart.valueDescriptionColumnRenderHtml" }}`)
	valuesSectionBuilder.WriteString(`{{ template "chart.valueDescriptionColumnRenderMd" . }}`)
	valuesSectionBuilder.WriteString("{{ end }}")

	valuesSectionBuilder.WriteString(`
{{ define "chart.valueDefaultColumnRender" }}
{{- $defaultValue := (default .Default .AutoDefault)  -}}
{{- $notationType := .NotationType }}
{{- if (and (hasPrefix "` + "`" + `" $defaultValue) (hasSuffix "` + "`" + `" $defaultValue) ) -}}
{{- $defaultValue = (toPrettyJson (fromJson (trimAll "` + "`" + `" (default .Default .AutoDefault) ) ) ) -}}
{{- $notationType = "json" }}
{{- end -}}
<pre lang="{{ $notationType }}">
{{- if (eq $notationType "tpl" ) }}
{{ .Key }}: |
{{- $defaultValue | nindent 2 }}
{{- else }}
{{ $defaultValue }}
{{- end }}
</pre>
{{ end }}

{{ define "chart.valuesTableHtml" }}
{{ if .Sections.Sections }}
{{- range .Sections.Sections }}
<h3>{{- .SectionName }}</h3>
<table>
	<thead>
		<th>Key</th>
		<th>Type</th>
		<th>Default</th>
		<th>Description</th>
	</thead>
	<tbody>
	{{- range .SectionItems }}
		<tr>
			<td>{{ template "chart.valueKeyColumnRenderHtml" . }}</td>
			<td>{{ template "chart.valueTypeColumnRenderHtml" . }}</td>
			<td>{{ template "chart.valueDefaultColumnRenderHtml" . }}</td>
			<td>{{ template "chart.valueDescriptionColumnRenderHtml" . }}</td>
		</tr>
	{{- end }}
	</tbody>
</table>
{{- end }}
{{ if .Sections.DefaultSection.SectionItems }}
<h3>{{- .Sections.DefaultSection.SectionName }}</h3>
<table>
	<thead>
		<th>Key</th>
		<th>Type</th>
		<th>Default</th>
		<th>Description</th>
	</thead>
	<tbody>
	{{- range .Sections.DefaultSection.SectionItems }}
	<tr>
		<td>{{ template "chart.valueKeyColumnRenderHtml" . }}</td>
		<td>{{ template "chart.valueTypeColumnRenderHtml" . }}</td>
		<td>{{ template "chart.valueDefaultColumnRenderHtml" . }}</td>
		<td>{{ template "chart.valueDescriptionColumnRenderHtml" . }}</td>
	</tr>
	{{- end }}
	</tbody>
</table>
{{ end }}
{{ else }}
<table>
	<thead>
		<th>Key</th>
		<th>Type</th>
		<th>Default</th>
		<th>Description</th>
	</thead>
	<tbody>
	{{- range .Values }}
		<tr>
			<td>{{ template "chart.valueKeyColumnRenderHtml" . }}</td>
			<td>{{ template "chart.valueTypeColumnRenderHtml" . }}</td>
			<td>{{ template "chart.valueDefaultColumnRenderHtml" . }}</td>
			<td>{{ template "chart.valueDescriptionColumnRenderHtml" . }}</td>
		</tr>
	{{- end }}
	</tbody>
</table>
{{ end }}
{{ end }}

{{ define "chart.valuesSectionHtml" }}
{{ if .Sections }}
{{ template "chart.valuesHeader" . }}
{{ template "chart.valuesTableHtml" . }}
{{ end }}
{{ end }}
		`)

	// TODO: Remove this block and rename `chart.valueDefaultColumnRender` to `chart.valueDefaultColumnRenderHtml` in the future
	valuesSectionBuilder.WriteString(`{{ define "chart.valueDefaultColumnRenderHtml" }}`)
	valuesSectionBuilder.WriteString(`{{ template "chart.valueDefaultColumnRender" . }}`)
	valuesSectionBuilder.WriteString("{{ end }}")

	return valuesSectionBuilder.String()
}

func getHelmDocsVersionTemplates() string {
	versionSectionBuilder := strings.Builder{}
	versionSectionBuilder.WriteString(`{{ define "helm-docs.version" }}{{ if .HelmDocsVersion }}{{ .HelmDocsVersion }}{{ end }}{{ end }}`)
	versionSectionBuilder.WriteString(`{{ define "helm-docs.versionFooter" }}`)
	versionSectionBuilder.WriteString("{{ if .HelmDocsVersion }}\n")
	versionSectionBuilder.WriteString("----------------------------------------------\n")
	versionSectionBuilder.WriteString("Autogenerated from chart metadata using [helm-docs v{{ .HelmDocsVersion }}](https://github.com/norwoodj/helm-docs/releases/v{{ .HelmDocsVersion }})")
	versionSectionBuilder.WriteString("{{ end }}")
	versionSectionBuilder.WriteString("{{ end }}")

	return versionSectionBuilder.String()
}

type documentationTemplateSource struct {
	name string
	text string
}

const defaultDocumentationTemplateName = "built-in README"

func readDocumentationTemplates(chartDirectory string, chartSearchRoot string, templateFiles []string) ([]documentationTemplateSource, bool, error) {
	sources := make([]documentationTemplateSource, 0, len(templateFiles))
	var missingLocalTemplate, missingSharedTemplate bool

	for _, templateFile := range templateFiles {
		var fullTemplatePath string

		if util.IsRelativePath(templateFile) {
			fullTemplatePath = filepath.Join(chartSearchRoot, templateFile)
		} else if util.IsBaseFilename(templateFile) {
			fullTemplatePath = filepath.Join(chartDirectory, templateFile)
		} else {
			fullTemplatePath = templateFile
		}

		contents, err := os.ReadFile(fullTemplatePath)
		if os.IsNotExist(err) {
			log.Debugf("Did not find template file %s for chart %s", templateFile, chartDirectory)
			if util.IsBaseFilename(templateFile) {
				missingLocalTemplate = true
			} else {
				missingSharedTemplate = true
			}
			continue
		}
		if err != nil {
			return nil, false, err
		}
		sources = append(sources, documentationTemplateSource{name: fullTemplatePath, text: string(contents)})
	}

	if missingSharedTemplate {
		// Explicit shared or absolute paths retain the legacy appended fallback.
		sources = append(sources, documentationTemplateSource{name: defaultDocumentationTemplateName, text: defaultDocumentationTemplate})
		return sources, false, nil
	}

	return sources, missingLocalTemplate, nil
}

func builtInDocumentationTemplates(badgeStyle string) []string {
	return []string{
		getNameTemplate(),
		getHeaderTemplate(),
		getDeprecatedTemplate(),
		getAppVersionTemplate(badgeStyle),
		getBadgesTemplates(),
		getDescriptionTemplate(),
		getVersionTemplates(badgeStyle),
		getTypeTemplate(badgeStyle),
		getSourceLinkTemplates(),
		getRequirementsTableTemplates(),
		getValuesTableTemplates(),
		getHomepageTemplate(),
		getMaintainersTemplate(),
		getHelmDocsVersionTemplates(),
	}
}

func newChartDocumentationTemplate(chartDocumentationInfo helm.ChartDocumentationInfo, chartSearchRoot string, templateFiles []string, badgeStyle string) (*template.Template, error) {
	documentationTemplate := template.New(chartDocumentationInfo.ChartDirectory)
	funcs := util.FuncMap()
	documentationTemplate.Funcs(funcs)
	userTemplates, missingLocalTemplate, err := readDocumentationTemplates(chartDocumentationInfo.ChartDirectory, chartSearchRoot, templateFiles)
	if err != nil {
		return nil, err
	}

	for _, t := range builtInDocumentationTemplates(badgeStyle) {
		if _, err := documentationTemplate.Parse(t); err != nil {
			return nil, err
		}
	}

	var combinedParsed bool
	if len(userTemplates) > 1 {
		var combined strings.Builder
		for _, source := range userTemplates {
			combined.WriteString(source.text)
		}
		_, err := documentationTemplate.Parse(combined.String())
		combinedParsed = err == nil
	}

	return parseDocumentationTemplateFiles(documentationTemplate, funcs, userTemplates, missingLocalTemplate, combinedParsed)
}

func parseDocumentationTemplateFiles(documentationTemplate *template.Template, funcs template.FuncMap, sources []documentationTemplateSource, missingLocalTemplate, combinedParsed bool) (*template.Template, error) {
	parts := make([]*template.Template, 0, len(sources))
	var separateBody strings.Builder
	for _, source := range sources {
		part, err := template.New(source.name).Funcs(funcs).Parse(source.text)
		if err != nil {
			if combinedParsed && templateBoundariesShareBodyScope(sources, funcs) {
				return finishCombinedDocumentationTemplate(documentationTemplate, missingLocalTemplate)
			}
			return nil, err
		}
		parts = append(parts, part)
		if combinedParsed {
			separateBody.WriteString(part.Tree.Root.String())
		}
	}

	// Preserve the combined parse when variables, actions, or whitespace
	// trimming cross file boundaries.
	if combinedParsed && separateBody.String() != documentationTemplate.Tree.Root.String() {
		return finishCombinedDocumentationTemplate(documentationTemplate, missingLocalTemplate)
	}

	for _, part := range parts {
		for _, definition := range part.Templates() {
			if definition.Name() != part.Name() {
				if _, err := documentationTemplate.AddParseTree(definition.Name(), definition.Tree); err != nil {
					return nil, err
				}
			}
		}
	}

	// Keep each file's parse tree intact so execution errors retain its location.
	var body strings.Builder
	var hasBody bool
	for _, part := range parts {
		name := part.Name()
		for documentationTemplate.Lookup(name) != nil {
			name += " (body)"
		}
		if _, err := documentationTemplate.AddParseTree(name, part.Tree); err != nil {
			return nil, err
		}
		fmt.Fprintf(&body, `{{ template %q . }}`, name)
		hasBody = hasBody || !parse.IsEmptyTree(part.Tree.Root)
	}

	// A chart-local template is optional when a loaded template provides the body.
	if missingLocalTemplate && !hasBody {
		body.WriteString(defaultDocumentationTemplate)
	}
	return documentationTemplate.Parse(body.String())
}

func templateBoundariesShareBodyScope(sources []documentationTemplateSource, funcs template.FuncMap) bool {
	// A document-body variable stays in scope inside if, range, and with, but
	// named templates have their own scope. References at file boundaries
	// distinguish shared document bodies from unterminated named templates.
	variable := "$helmDocsFileBoundary"
	for _, source := range sources {
		for strings.Contains(source.text, variable) {
			variable += "_"
		}
	}

	var probe strings.Builder
	fmt.Fprintf(&probe, "{{ %s := . }}", variable)
	for i, source := range sources {
		if i > 0 {
			fmt.Fprintf(&probe, "{{ %s }}", variable)
		}
		probe.WriteString(source.text)
	}
	_, err := template.New("template file boundaries").Funcs(funcs).Parse(probe.String())
	return err == nil
}

func finishCombinedDocumentationTemplate(documentationTemplate *template.Template, missingLocalTemplate bool) (*template.Template, error) {
	if missingLocalTemplate && parse.IsEmptyTree(documentationTemplate.Tree.Root) {
		return documentationTemplate.Parse(defaultDocumentationTemplate)
	}
	return documentationTemplate, nil
}
