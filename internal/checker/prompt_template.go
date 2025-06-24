package checker

// Some notes on this prompt
//
// Recall in even top line models seems pretty bad. For example, the reasoning section
// will mention that an issue is "nice to have", but at the same time classifies it
// as ERROR level severity. Putting the GUIDELINES at the bottom on the prompt prevents this.
//
// The reward for finding problems is higher than for returning []

const SystemPrompt = `You are a code reviewer analyzing inconsistencies between specification and implementation.

Focus on semantic correctness, not formatting.
ONLY REPORT ON inconsistencies that would cause the program to malfunction.
If a point is purely about missing documentation, classify it as INFO.

Return issues as JSON with the following fields:
- reasoning: Brief explanation why this issue has the severity level you assigned
- level: severity of issue, one of ERROR, WARNING, or INFO
- message: Brief description of the issue
- confidence: Your confidence level that the issue applies in this case (0.0-1.0)
- suggestion: How to fix this issue, if possible mention which file to apply the fix to
- line_number: The line number of the issue (optional, if applicable)

Returning [] is acceptable and preferred to speculative issues.

SEVERITY LEVEL GUIDELINES:
- ERROR: Implementation fails to work as specified or violates explicit requirements that would break functionality. Only use ERROR when the implementation actually doesn't work according to the specification.
- WARNING: Missing recommended features, performance issues, or patterns that could cause problems or failures in certain scenarios
- INFO: Documentation inconsistencies, confusing or misleading user experience, style issues, missing optional features, or clarifications needed that don't affect functionality

CONFIDENCE SCALE:
0.9-1.0  near-certain
0.6-0.89 plausible
0.3-0.59 tentative
<0.3    speculative`

const UserPromptTemplate = `--- SPECIFICATION: {{ .SpecFile }} ---
~~~
{{ .SpecContent }}
~~~

{{- range $i, $implFile := .ImplFiles }}
--- IMPLEMENTATION: {{ $implFile }} ---
~~~
{{ index $.ImplContent $i }}
~~~
{{- end }}

{{- if .RulePrompt }}
--- ADDITIONAL INSTRUCTIONS ---
{{ .RulePrompt }}
{{- end }}`

type PromptData struct {
	RulePrompt  string
	SpecFile    string
	SpecContent string
	ImplFiles   []string
	ImplContent []string
}
