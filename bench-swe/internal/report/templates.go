package report

import "text/template"

var summaryTmpl = template.Must(template.New("summary").Parse(`# SWE-Bench Summary

Generated: {{ .Date }} | Embed: ` + "`{{ .EmbedModel }}`" + ` | Claude: ` + "`{{ .ClaudeModel }}`" + `

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

{{ .ResultsTableHeader }}
{{ .ResultsTableSep }}
{{ range .Rows }}{{ .ResultsRow }}
{{ end }}
## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
{{ range .ScenarioAggs }}| **{{ .Name }}** | {{ .Perfect }} | {{ .Good }} | {{ .Poor }} | {{ .AvgCost }} | {{ .AvgTime }} | {{ .AvgTokens }} |
{{ end }}
## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
{{ range .LangAggs }}| {{ .Language }} | {{ .BaselineWins }} | {{ .WithLumenWins }} |
{{ end }}
`))

var detailTmpl = template.Must(template.New("detail").Parse(`# SWE-Bench Detail Report

Generated: {{ .Date }}

{{ range .Tasks }}---

## {{ .ID }} [{{ .Language }} / {{ .Difficulty }}]

**Issue:** {{ .IssueTitle }}

> {{ .IssueBodyQuoted }}

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
{{ range .MetricsRows }}| **{{ .Scenario }}** | {{ .Duration }} | {{ .InputTokens }} | {{ .CacheRead }} | {{ .OutputTokens }} | {{ .Cost }} |
{{ end }}
{{ range .ScenarioDetails }}### {{ .Scenario }}

{{ if .Rating }}**Rating: {{ .Rating }}**

{{ end }}{{ if .Explanation }}{{ .Explanation }}

{{ end }}{{ if .Patch }}` + "```diff" + `
{{ .Patch }}` + "```" + `

{{ end }}{{ end }}{{ end }}
`))
