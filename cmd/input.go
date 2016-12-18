package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/qri-io/dataset"
)

var validAddress = regexp.MustCompile(`^[a-z0-9\.]1,250}$`)

func prompt(msg string) string {
	var input string
	printPrompt(msg)
	fmt.Scanln(&input)
	return strings.TrimSpace(input)
}

func InputAddress(message string, defaultAdr dataset.Address) (dataset.Address, error) {
	if message == "" {
		message = "please enter an address"
	}
	input := strings.ToLower(prompt(fmt.Sprintf("%s [%s]: ", message, defaultAdr)))

	if input == "" && defaultAdr.String() != "" {
		return defaultAdr, nil
	}

	if !dataset.ValidAddressString(input) {
		PrintRed("invalid address: '%s'. Addresses are lower_case.alpha_numeric.separated.by.dots", input)
		return InputAddress(message, defaultAdr)
	}

	return dataset.NewAddress(input), nil
}
