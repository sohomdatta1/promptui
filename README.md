# promptui

Interactive prompt for command-line applications.

This project is a fork of the original [promptui project](https://github.com/manifoldco/promptui) by the Manifold Co.

[![GitHub release](https://img.shields.io/github/tag/sohomdatta1/promptui.svg?label=latest)](https://github.com/sohomdatta1/promptui/releases)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/sohomdatta1/promptui)
[![License](https://img.shields.io/badge/license-BSD-blue.svg)](./LICENSE.md)

## Overview

![promptui](https://media.giphy.com/media/xUNda0Ngb5qsogLsBi/giphy.gif)

Promptui is a library providing a simple interface to create command-line
prompts for go. It can be easily integrated into
[spf13/cobra](https://github.com/spf13/cobra),
[urfave/cli](https://github.com/urfave/cli) or any cli go application.

Promptui has two main input modes:

- `Prompt` provides a single line for user input. Prompt supports
  optional live validation, confirmation and masking the input.

- `Select` provides a list of options to choose from. Select supports
  pagination, search, detailed view and custom templates.

For a full list of options check [GoDoc](https://godoc.org/github.com/sohomdatta1/promptui).

## Basic Usage

### Prompt

```go
package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/sohomdatta1/promptui"
)

func main() {
	validate := func(input string) error {
		_, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return errors.New("Invalid number")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Number",
		Validate: validate,
	}

	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Printf("You choose %q\n", result)
}
```

### Select

```go
package main

import (
	"fmt"

	"github.com/sohomdatta1/promptui"
)

func main() {
	prompt := promptui.Select{
		Label: "Select Day",
		Items: []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday",
			"Saturday", "Sunday"},
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Printf("You choose %q\n", result)
}
```

### More Examples

See full list of [examples](https://github.com/sohomdatta1/promptui/tree/master/_examples)
