package processor

const SystemPrompt = `You're tasked with generating a grep query that can be used to find
specific lines in a file. The query should be expressed as a regular expression. The user will
provide some context about which sections are needed in a <specifically> section. The rule name
will be provided in <rule_name> section and the rule description in <description> section.
The header of the specification document will also be provided in <header> section.
Use this header section as a hint on the structure of this particular document,
it is simply the first N characters of the document.
You should output the query ONLY as JSON object with a single "query" key.`

const UserPromptTemplate = `
<header>{{ .Header }}</header>
<specifically>{{ .Specifically }}</specifically>
<rule_name>{{ .RuleName }}</rule_name>
<description>{{ .RuleDescription }}</description>
`

type PromptData struct {
	Header          string
	Specifically    string
	RuleName        string
	RuleDescription string
}
