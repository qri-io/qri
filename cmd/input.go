package cmd

import (
	"fmt"
	"strings"
)

func prompt(msg string) string {
	var input string
	printPrompt(msg)
	fmt.Scanln(&input)
	return strings.TrimSpace(input)
}

func InputText(message, defaultText string) string {
	if message == "" {
		message = "enter text:"
	}
	input := prompt(fmt.Sprintf("%s [%s]: ", message, defaultText))

	return input
}
