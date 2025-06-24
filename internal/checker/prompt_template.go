package checker

// Some notes on this prompt
//
// Recall in even top line models seems pretty bad. For example, the reasoning section
// will mention that an issue is "nice to have", but at the same time classifies it
// as ERROR level severity. Putting the GUIDELINES at the bottom on the prompt prevents this.

const PromptTemplate = `You are a code reviewer analyzing whether an implementation matches its specification.

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
- ERROR: Implementation fails to work as specified or violates explicit requirements that would break functionality. Only use ERROR when the implementation actually doesn't work according to the specification.
- WARNING: Missing recommended features, performance issues, or patterns that could cause problems or failures in certain scenarios
- INFO: Documentation inconsistencies, style issues, missing optional features, or clarifications needed that don't affect functionality


CONFIDENCE SCALE:
0.9-1.0  near-certain  
0.6-0.89 plausible  
0.3-0.59 tentative  
<0.3    speculative
`

type PromptData struct {
	RulePrompt  string
	SpecFile    string
	SpecContent string
	ImplFiles   []string
	ImplContent []string
}
