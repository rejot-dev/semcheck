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
			input:    "// semcheck:file(./some-file.md)\n",
			expected: []InlineReference{{Command: File, Args: []string{"./some-file.md"}, LineNumber: 1}},
		},
		{
			input:    "semcheck:rfc(123) some other text\n",
			expected: []InlineReference{{Command: RFC, Args: []string{"123"}, LineNumber: 1}},
		},
		{
			input:    "semcheck:url(https://example.com/) some other text\n",
			expected: []InlineReference{{Command: URL, Args: []string{"https://example.com/"}, LineNumber: 1}},
		},
		{
			input:    "Multi Line \n  // semcheck:url(https://example.com/) \n some other text\n",
			expected: []InlineReference{{Command: URL, Args: []string{"https://example.com/"}, LineNumber: 2}},
		},
		{
			input:    "semcheck:rfc(123, another)",
			expected: []InlineReference{{Command: RFC, Args: []string{"123", "another"}, LineNumber: 1}},
		},
	}

	for _, tc := range cases {
		refs, err := FindReferences(tc.input)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
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
		expected ParseError
	}{
		{
			input:    "// semcheck:invalidcommand(./some-file.md)\n",
			expected: ParseError{Err: ErrorInvalidCommand, LineNumber: 1},
		},
		{
			input:    "Multiple Lines \n// semcheck:invalidcommand(./some-file.md)\n",
			expected: ParseError{Err: ErrorInvalidCommand, LineNumber: 2},
		},
		{
			input:    "// semcheck:file(args broken\n",
			expected: ParseError{Err: ErrorInvalidArgsMissingClosingParantheses, LineNumber: 1},
		},
		{
			input:    "// semcheck(args broken\n",
			expected: ParseError{Err: ErrorInvalidCommand, LineNumber: 1},
		},
		{
			input:    "// semcheck:file\n",
			expected: ParseError{Err: ErrorInvalidArgsMissingOpeningParantheses, LineNumber: 1},
		},
		{
			input:    "// semcheck:rfc 123\n",
			expected: ParseError{Err: ErrorInvalidArgsMissingOpeningParantheses, LineNumber: 1},
		},
		{
			input:    "// semcheck\n",
			expected: ParseError{Err: ErrorInvalidCommand, LineNumber: 1},
		},
	}

	for _, tc := range cases {
		_, err := FindReferences(tc.input)

		if err == nil {
			t.Errorf("Expected error, got nothing")
			continue
		}

		if err.Err != tc.expected.Err {
			t.Errorf("Expected Error  %s, got %s", tc.expected.Err, err.Err)
		}
		if err.LineNumber != tc.expected.LineNumber {
			t.Errorf("Expected LineNumber %d, got %d", tc.expected.LineNumber, err.LineNumber)
		}
		// fmt.Printf("Error: %s\n", err.Format())
	}
}
