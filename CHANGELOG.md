# Changelog
Changelog entries should be added to this file in reverse chronological order. On release change the version number in the header and create a new [Unreleased] section. Use subsections "Added", "Changed" and "Removed" to distinguish between different types of changes, omit a subsection if there are no changes.

## [Unreleased]

### Added
- **Anchor Support**: Added support for URL fragments/anchors for structured formats such as markdown and HTML. Include a subsection of your specification by using link fragment like `#section-3.1.1`. For HTML documents, this follows the same semantics as your browser targeting a section, for markdown you can use the header text as a link instead.
- **Log Level CLI Option**: Added `--log-level` command line option to control logging verbosity (info, debug, error, warning)

### Changed
- **HTML Content Filtering**: Implemented content filtering to remove unwanted HTML elements (scripts, styles, forms, etc.) and attributes using allowlist approach
- **Document Processing**: Updated implementation for downloading/reading specifications

## v1.1

### Added
- **Inline Specification References**: Link specifications directly in code comments using `semcheck:file()`, `semcheck:rfc()`, or `semcheck:url()` commands
- **New landing page**: Added a dedicated landing page hosted on [semcheck.ai](https://semcheck.ai)
- **Expanded Eval test cases**: Added additional test cases for the evaluation suite that test larger codebases and specifications

### Changed
  - **Config**: Rules are now optional, semcheck can run with only inline specification references
  - **Logging**: Replaced fmt.Println with charmbracelet/log for colorful output and migrated stdout reporter from fatih/color to lipgloss to keep things consistent

### Removed
- The `confidence_threshold` configuration option has been marked as deprecated, and will be removed in a future release. We found that LLMs almost always report high confidence levels and that this mechanism doesn't bring much value.


## v1.0

Initial release of semcheck.
