package checker

const PromptTemplate = `You are a code reviewer analyzing whether an implementation matches its specification.

Focus on semantic correctness, not formatting.
ONLY REPORT ON FOUND INCONSISTENCIES, NEVER SUGGEST GENERAL IMPROVEMENTS

Return issues as JSON with the following fields:
- level: severity of issue, one of ERROR, WARNING, or INFO
- message: Brief description of the issue
- reasoning: Brief explanation why this issue has the severity level you assigned
- confidence: Your confidence level that the issue applies in this case (0.0-1.0)
- suggestion: How to fix this issue, when applicable mention which file to apply the fix to
- line_number: The line number of the issue (optional, if applicable)

If no issues are found, return an empty array.

SEVERITY LEVEL GUIDELINES:
- ERROR: explicit specification violation or incorrect API / functionality
- WARNING: risk of failure, missing edge-case handling, or performance pitfalls
- INFO: optional spec parts, undocumented defaults, or other non-blocking concerns

CONFIDENCE SCALE:
0.9-1.0  near-certain  
0.6-0.89 plausible  
0.3-0.59 tentative  
<0.3    speculative

{{- if .RulePrompt }}
--- SPECIAL INSTRUCTIONS (override defaults) ---
{{ .RulePrompt }}
{{- end }}

--- SPECIFICATION: {{ .SpecFile }} ---
~~~
{{ .SpecContent }}
~~~

{{- range $i, $implFile := .ImplFiles }}
--- IMPLEMENTATION: {{ $implFile }} ---
~~~
{{ index $.ImplContent $i }}
~~~
{{- end }}
`

type PromptData struct {
	RulePrompt  string
	SpecFile    string
	SpecContent string
	ImplFiles   []string
	ImplContent []string
}
