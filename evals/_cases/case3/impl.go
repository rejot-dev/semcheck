package stringutil

import "fmt"

func ProcessString(input string) string {
	return input + "_processed"
}

func FormatMessage(template string, arg string) string {
	return fmt.Sprintf(template, arg)
}
