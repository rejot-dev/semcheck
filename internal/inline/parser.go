package inline

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type InlineCommand string

const (
	File    InlineCommand = "file"
	RFC     InlineCommand = "rfc"
	URL     InlineCommand = "url"
	Invalid InlineCommand = "invalid"
)

const semcheckPrefix = "semcheck:"

var (
	ErrorInvalidCommand                       = errors.New("invalid command")
	ErrorInvalidArgs                          = errors.New("invalid arguments for command")
	ErrorInvalidArgsNumber                    = errors.New("invalid number of arguments for command")
	ErrorInvalidArgsMissingOpeningParantheses = errors.New("missing opening paranthesis for argument list")
	ErrorInvalidArgsMissingClosingParantheses = errors.New("missing closing paranthesis for argument list")
	ErrorInvalidArgsMissingArguments          = errors.New("must provide at least one argument")
	ErrorInvalidArgsMissingSpecFile           = errors.New("file path not found")
	ErrorInvalidArgsRFCNumber                 = errors.New("RFC number must be a positive integer")
	ErrorInvalidArgsURL                       = errors.New("invalid URL format")
)

type InlineError struct {
	Err          error
	LineNumber   int
	ColumnNumber int
	Line         string
}

func (e InlineError) Error() string {
	return e.Err.Error()
}

func (e InlineError) Format() string {
	// Create the caret pointer line
	caret := strings.Repeat(" ", e.ColumnNumber-1) + "^"

	return fmt.Sprintf("%s\n> %2d | %s\n       %s", e.Err.Error(), e.LineNumber, e.Line, caret)
}

type InlineReference struct {
	// The command used
	Command InlineCommand
	// Arguments passed to the command
	Args []string
	// Line number where the reference was found
	LineNumber int
}

// argString is a parenthesized enclosed string containing a number of arguments, potentially with additional
// text after the closing parenthesis. Function arguments cannot contain the closing parenthesis character.
// Examples:
// (hello)
// (./some/file.txt) some other text
// (123, hello)
func consumeCommandArgs(argString string) ([]string, error) {
	if len(argString) == 0 {
		return nil, ErrorInvalidArgsMissingOpeningParantheses
	}
	if argString[0] != '(' {
		return nil, ErrorInvalidArgsMissingOpeningParantheses
	}
	endIndex := strings.Index(argString, ")")
	if endIndex == -1 {
		return nil, ErrorInvalidArgsMissingClosingParantheses
	}
	argString = argString[1:endIndex]

	if argString == "" {
		return nil, ErrorInvalidArgsMissingArguments
	}

	args := strings.Split(argString, ",")

	for i, arg := range args {
		args[i] = strings.TrimSpace(arg)
	}
	return args, nil
}

func consumeCommandName(commandArgs string) (InlineCommand, string) {
	if len(commandArgs) == 0 {
		return Invalid, ""
	}

	if strings.HasPrefix(commandArgs, "file") {
		return File, commandArgs[4:]
	} else if strings.HasPrefix(commandArgs, "rfc") {
		return RFC, commandArgs[3:]
	} else if strings.HasPrefix(commandArgs, "url") {
		return URL, commandArgs[3:]
	}
	return Invalid, ""
}

func FindReferences(document string) ([]InlineReference, []InlineError) {
	lines := strings.Split(document, "\n")
	refs := make([]InlineReference, 0)
	var inlineErrors []InlineError

	for lineNumber, line := range lines {
		index := strings.Index(line, semcheckPrefix)
		if index == -1 {
			continue
		}

		command, argString := consumeCommandName(line[index+len(semcheckPrefix):])
		if command == Invalid {
			inlineErrors = append(inlineErrors, InlineError{
				Err:          ErrorInvalidCommand,
				LineNumber:   lineNumber + 1,
				ColumnNumber: index + len(semcheckPrefix) + 1,
				Line:         line,
			})
			continue
		}

		args, err := consumeCommandArgs(argString)

		if err == nil {
			args, err = validateAndTransformArgs(command, args)
		}

		if err != nil {
			inlineErrors = append(inlineErrors, InlineError{
				Err:          err,
				LineNumber:   lineNumber + 1,
				ColumnNumber: index + len(semcheckPrefix) + len(command) + 1,
				Line:         line,
			})
			continue
		}

		refs = append(refs, InlineReference{
			Command:    command,
			Args:       args,
			LineNumber: lineNumber + 1,
		})
	}
	return refs, inlineErrors
}

func validateAndTransformArgs(command InlineCommand, args []string) ([]string, error) {
	switch command {
	case File:
		if len(args) != 1 {
			return args, ErrorInvalidArgsNumber
		}

		// TODO: Consider moving this check outside of the parser
		if _, err := os.Stat(args[0]); os.IsNotExist(err) {
			return args, ErrorInvalidArgsMissingSpecFile
		}

		return args, nil
	case RFC:
		if len(args) != 1 {
			return args, ErrorInvalidArgsNumber
		}

		// Validate that the argument is a positive integer
		rfcNumber, err := strconv.Atoi(args[0])
		if err != nil || rfcNumber <= 0 {
			return args, ErrorInvalidArgsRFCNumber
		}

		args = []string{fmt.Sprintf("https://www.rfc-editor.org/rfc/rfc%d.txt", rfcNumber)}

		return args, nil
	case URL:
		if len(args) != 1 {
			return args, ErrorInvalidArgsNumber
		}

		// Validate that the argument is a valid URL
		parsedURL, err := url.Parse(args[0])
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return args, ErrorInvalidArgsURL
		}

		// Only allow HTTP/HTTPS URLs
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return args, ErrorInvalidArgsURL
		}

		return args, nil
	default:
		return args, nil
	}
}
