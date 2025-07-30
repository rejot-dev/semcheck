package inline

import (
	"testing"
)

func TestFindReferences(t *testing.T) {

	cases := []struct {
		input    string
		expected []InlineReference
	}{
		{
			input:    "// semcheck:file(/tmp/test_spec.md)\n",
			expected: []InlineReference{{Command: File, Args: []string{"/tmp/test_spec.md"}, LineNumber: 1}},
		},
		{
			input:    "semcheck:rfc(123) some other text\n",
			expected: []InlineReference{{Command: RFC, Args: []string{"https://www.rfc-editor.org/rfc/rfc123.txt"}, LineNumber: 1}},
		},
		{
			input:    "semcheck:url(https://example.com/) some other text\n",
			expected: []InlineReference{{Command: URL, Args: []string{"https://example.com/"}, LineNumber: 1}},
		},
		{
			input:    "Multi Line \n  // semcheck:url(https://example.com/) \n some other text\n",
			expected: []InlineReference{{Command: URL, Args: []string{"https://example.com/"}, LineNumber: 2}},
		},
	}

	for _, tc := range cases {
		refs, parseErrors := FindReferences(tc.input)

		if len(parseErrors) > 0 {
			t.Errorf("Unexpected errors: %v", parseErrors)
		}

		if len(refs) != len(tc.expected) {
			t.Errorf("Expected %d references, got %d", len(tc.expected), len(refs))
		}

		for i, ref := range refs {
			if ref.Command != tc.expected[i].Command {
				t.Errorf("Expected command %s, got %s", tc.expected[i].Command, ref.Command)
			}
			if len(ref.Args) != len(tc.expected[i].Args) {
				t.Errorf("Expected %d arguments, got %d", len(tc.expected[i].Args), len(ref.Args))
			}
			for j, arg := range ref.Args {
				if arg != tc.expected[i].Args[j] {
					t.Errorf("Expected argument %s, got %s", tc.expected[i].Args[j], arg)
				}
			}
			if ref.LineNumber != tc.expected[i].LineNumber {
				t.Errorf("Expected line number %d, got %d", tc.expected[i].LineNumber, ref.LineNumber)
			}
		}
	}

}

func TestErrorsFindReferences(t *testing.T) {

	cases := []struct {
		input    string
		expected InlineError
	}{
		{
			input:    "// semcheck:invalidcommand(./some-file.md)\n",
			expected: InlineError{Err: ErrorInvalidCommand, LineNumber: 1},
		},
		{
			input:    "Multiple Lines \n// semcheck:invalidcommand(./some-file.md)\n",
			expected: InlineError{Err: ErrorInvalidCommand, LineNumber: 2},
		},
		{
			input:    "// semcheck:file(args broken\n",
			expected: InlineError{Err: ErrorInvalidArgsMissingClosingParantheses, LineNumber: 1},
		},
		{
			input:    "// semcheck:file\n",
			expected: InlineError{Err: ErrorInvalidArgsMissingOpeningParantheses, LineNumber: 1},
		},
		{
			input:    "// semcheck:rfc 123\n",
			expected: InlineError{Err: ErrorInvalidArgsMissingOpeningParantheses, LineNumber: 1},
		},
		{
			input:    "// semcheck:file()",
			expected: InlineError{Err: ErrorInvalidArgsMissingArguments, LineNumber: 1},
		},
		{
			input:    "// semcheck:rfc()",
			expected: InlineError{Err: ErrorInvalidArgsMissingArguments, LineNumber: 1},
		},
		{
			input:    "// semcheck:rfc(not_a_number)",
			expected: InlineError{Err: ErrorInvalidArgsRFCNumber, LineNumber: 1},
		},
		{
			input:    "// semcheck:rfc(-1)",
			expected: InlineError{Err: ErrorInvalidArgsRFCNumber, LineNumber: 1},
		},
		{
			input:    "// semcheck:rfc(0)",
			expected: InlineError{Err: ErrorInvalidArgsRFCNumber, LineNumber: 1},
		},
		{
			input:    "// semcheck:url()",
			expected: InlineError{Err: ErrorInvalidArgsMissingArguments, LineNumber: 1},
		},
		{
			input:    "// semcheck:url(not_a_url)",
			expected: InlineError{Err: ErrorInvalidArgsURL, LineNumber: 1},
		},
		{
			input:    "semcheck:url(ftp://example.com)",
			expected: InlineError{Err: ErrorInvalidArgsURL, LineNumber: 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			_, parseErrors := FindReferences(tc.input)

			if len(parseErrors) == 0 {
				t.Errorf("Expected error, got nothing")
				return
			}

			// Check first error matches expected
			err := parseErrors[0]
			if err.Err != tc.expected.Err {
				t.Errorf("Expected Error  %s, got %s", tc.expected.Err, err.Err)
			}
			if err.LineNumber != tc.expected.LineNumber {
				t.Errorf("Expected LineNumber %d, got %d", tc.expected.LineNumber, err.LineNumber)
			}
			// fmt.Printf("Error: %s\n", err.Format())
		})
	}
}
