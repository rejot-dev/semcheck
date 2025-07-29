# Inline Specification References

## Overview

Inline specification references allow developers to link specification files directly from implementation files using special comment syntax. This feature enables granular specification-to-implementation mapping at the code level, complementing the global rule-based configuration.

## Syntax

The inline specification reference uses a language-agnostic syntax and can be used within comments of your respective language.

```
// semcheck:[command]([args])
```

### Components

- **semcheck**: Required identifier (case-sensitive)
- **:[command]**: Required command specifier
- **([...args])**: Command arguments in parentheses

### Supported Commands

#### 1. file(path)
Links to a local specification file.

**Syntax**: `semcheck:file(path)`

**Arguments**:
- `path`: file relative path or repo relative path to a specification file, absolute paths starting with `/` consider the repo as the root directory. If not located inside a git repository, the CWD will be used as root.

**Examples**:
```javascript
// semcheck:file(../specs/api.md)
// semcheck:file(/specs/protocol.md)
```

#### 2. rfc(number)
Links to an IETF RFC document.

**Syntax**: `semcheck:rfc(number)`

**Arguments**:
- `number`: RFC number (e.g., 7946, 8259)

**Examples**:
```python
# semcheck:rfc(8259)  # JSON specification
# semcheck:rfc(7946)  # GeoJSON specification
```

#### 3. url(url)
Links to a document on the web.

**Syntax**: `semcheck:url(url)`

**Arguments**:
- `url`: URL of the document

**Examples**:
```python
# semcheck:url(https://docs.anthropic.com/en/api/messages)
```

## Processing Behavior

### File Discovery
1. During semcheck initialization, all files in the working directory are scanned
2. Files matching the following patterns are ignored by default:
   - `**/.git`, `**/.svn`, `**/.hg`, `**/.jj`, `**/CVS` (version control)
   - `**/.DS_Store`, `**/Thumbs.db` (system files)
   - `**/.classpath`, `**/.settings` (IDE files)
3. Files excluded by `.gitignore`, `.semignore`, or rule exclusion patterns are also skipped
4. Each remaining file is parsed line-by-line for inline specification references

#### Ignore Files
- **`.gitignore`**: Standard Git ignore patterns are respected
- **`.semignore`**: semcheck-specific ignore patterns using the same syntax as `.gitignore`

### Reference Resolution
1. **File references**: Resolved relative to the working directory of where semcheck is running
2. **RFC references**: Fetched from `https://www.rfc-editor.org/rfc/rfc{number}.txt`
3. **URL references**: Fetched from the provided URL
4. Invalid references generate warnings but don't halt processing

### Integration with Rules
- Inline references are additive to configured rules in the semcheck config
- Each pair of implementation file and specification file are considered an implicit semantic rule and will be checked independently, if multiple files reference the same specification, only one call to the LLM will be performed with all implementation files.

## Examples

```javascript
// semcheck:file(./specs/user-api.md)
class UserService {
    // ... implementation
}
```

```python
# semcheck:rfc(8259)
class MyJsonParser:

    def __init__(self):
        pass

    def parse_from_string(self, data: str):
        pass
```

## Validation Rules

### Syntax Validation
- Command is one of  'file', 'rfc', 'url'
- Arguments must be enclosed in parentheses
- Multiple arguments separated by commas, with optional whitespace

### Path Validation
- File paths are validated for existence during processing,
- urls and rfc are validated only for syntactic correctness

### Error Handling
- Malformed syntax generates warnings and are ignored
- Missing local files fail semcheck
