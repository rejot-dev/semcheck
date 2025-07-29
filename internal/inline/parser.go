package inline

import (
	"errors"
	"fmt"
	"strings"
)

type InlineCommand string

const (
	File    InlineCommand = "file"
	RFC     InlineCommand = "rfc"
	URL     InlineCommand = "url"
	Invalid InlineCommand = "invalid"
)

const semcheckPrefix = "semcheck"

var (
	ErrorInvalidCommand                       = errors.New("invalid command")
	ErrorInvalidArgs                          = errors.New("invalid arguments for command")
	ErrorInvalidArgsMissingOpeningParantheses = errors.New("missing opening paranthesis for argument list")
	ErrorInvalidArgsMissingClosingParantheses = errors.New("missing closing paranthesis for argument list")
	ErrorInvalidArgsMissingArguments          = errors.New("must provide at least one argument")
)

type ParseError struct {
	Err          error
	LineNumber   int
	ColumnNumber int
	Line         string
}

func (e ParseError) Error() string {
	return e.Err.Error()
}

func (e ParseError) Format() string {
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

type FileCommandArgs struct {
	SpecFile string
}

type UrlCommandArgs struct {
	Url string
}

type RfcCommandArgs struct {
	Rfc int
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
	if len(commandArgs) == 0 || commandArgs[0] != ':' {
		// A colon is required to delineate the command from the prefix (i.e. semcheck:command)
		return Invalid, ""
	}

	commandArgs = commandArgs[1:]

	if strings.HasPrefix(commandArgs, "file") {
		return File, commandArgs[4:]
	} else if strings.HasPrefix(commandArgs, "rfc") {
		return RFC, commandArgs[3:]
	} else if strings.HasPrefix(commandArgs, "url") {
		return URL, commandArgs[3:]
	}
	return Invalid, ""
}

func FindReferences(document string) ([]InlineReference, []ParseError) {
	lines := strings.Split(document, "\n")
	refs := make([]InlineReference, 0)
	var parseErrors []ParseError

	for lineNumber, line := range lines {
		index := strings.Index(line, semcheckPrefix)
		if index == -1 {
			continue
		}

		command, argString := consumeCommandName(line[index+len(semcheckPrefix):])
		if command == Invalid {
			parseErrors = append(parseErrors, ParseError{
				Err:          ErrorInvalidCommand,
				LineNumber:   lineNumber + 1,
				ColumnNumber: index + len(semcheckPrefix) + 1,
				Line:         line,
			})
			continue
		}

		args, err := consumeCommandArgs(argString)

		if err != nil {
			parseErrors = append(parseErrors, ParseError{
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
	return refs, parseErrors
}
