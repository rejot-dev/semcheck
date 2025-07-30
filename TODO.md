# TODOs

Issues
- `semcheck spec/inline.md` currently doesn't check for inline spec references that mention this file, only rules.

Experiments
- Consider the developer workflow, requirements -> design -> implementation plan -> execute -> verify
  - implementation plan and execution are left to the dev/coding agent
  - semcheck's domain is therefore only the verify step, making use of the requirements and design created by developers or AI coding agents
  - semcheck must be additive to existing AI IDE's, makes no sense to compete with IDE's workflows. MCP integration can be important step there
  - open format for tying spec-to-impl will allow for general adoption
- Is there a better interface than the current config yaml file?
  - Inline spec checking using comments in code
  - md frontmatter
- IDE integration?
  - Problem panel show last issues
- hooks
  - on file save, run semcheck

Maturity
- Tying issues to exact files and possibly lines
- clean up config file structure around 'files' and 'specs' (drop 'specs.path' key, rename 'files' to 'impls'?)
- Option to check a single rule
- Fail fast option
- Add rule names to issues
- Better error handling
- json schema for config file

Cool
- Auto derive rules from existing specification files and implementation
- You can provide semantic descriptions of subsections for specs: e.g. semcheck:file(spec.md, Section 14.3)
- Implementation or specification 'quotes' to be displayed along issues
  - or more generally, improve UX when trying to backtrack an issue to it's origin
- Have a "thinking" preview in the terminal saying things like "considering function arguments in config validation..."
- Parallel Execution
