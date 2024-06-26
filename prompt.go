package promptui

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/ergochat/readline"
)

// Prompt represents a single line text field input with options for validation and input masks.
type Prompt struct {
	// Label is the value displayed on the command line prompt.
	//
	// The value for Label can be a simple string or a struct that will need to be accessed by dot notation
	// inside the templates. For example, `{{ .Name }}` will display the name property of a struct.
	Label interface{}

	// Default is the initial value for the prompt. This value will be displayed next to the prompt's label
	// and the user will be able to view or change it depending on the options.
	Default string

	// AllowEdit lets the user edit the default value. If false, any key press
	// other than <Enter> automatically clears the default value.
	AllowEdit bool

	// Validate is an optional function that fill be used against the entered value in the prompt to validate it.
	Validate ValidateFunc

	// Mask is an optional rune that sets which character to display instead of the entered characters. This
	// allows hiding private information like passwords.
	Mask rune

	// LazyValidation sets whether to validate the input only after the user has pressed enter. If false, the
	// validation will be done once the user presses enter.
	LazyValidation bool

	// Templates can be used to customize the prompt output. If nil is passed, the
	// default templates are used. See the PromptTemplates docs for more info.
	Templates *PromptTemplates

	// IsConfirm makes the prompt ask for a yes or no ([Y/N]) question rather than request an input. When set,
	// most properties related to input will be ignored.
	IsConfirm bool

	// IsVimMode enables vi-like movements (hjkl) and editing.
	IsVimMode bool

	// the Pointer defines how to render the cursor.
	Pointer Pointer

	Stdin  io.Reader
	Stdout io.Writer
}

// PromptTemplates allow a prompt to be customized following stdlib
// text/template syntax. Custom state, colors and background color are available for use inside
// the templates and are documented inside the Variable section of the docs.
//
// # Examples
//
// text/templates use a special notation to display programmable content. Using the double bracket notation,
// the value can be printed with specific helper functions. For example
//
// This displays the value given to the template as pure, unstylized text.
//
//	'{{ . }}'
//
// This displays the value colored in cyan
//
//	'{{ . | cyan }}'
//
// This displays the value colored in red with a cyan background-color
//
//	'{{ . | red | cyan }}'
//
// See the doc of text/template for more info: https://golang.org/pkg/text/template/
type PromptTemplates struct {
	// Prompt is a text/template for the prompt label displayed on the left side of the prompt.
	Prompt string

	// Prompt is a text/template for the prompt label when IsConfirm is set as true.
	Confirm string

	// Valid is a text/template for the prompt label when the value entered is valid.
	Valid string

	// Invalid is a text/template for the prompt label when the value entered is invalid.
	Invalid string

	// Success is a text/template for the prompt label when the user has pressed entered and the value has been
	// deemed valid by the validation function. The label will keep using this template even when the prompt ends
	// inside the console.
	Success string

	// Unvalidated is a text/template for the prompt label when the value entered is unvalidated.
	// this is the state used when the LazyValidation option is set to true.
	Unvalidated string

	// Prompt is a text/template for the prompt label when the value is invalid due to an error triggered by
	// the prompt's validation function.
	ValidationError string

	// FuncMap is a map of helper functions that can be used inside of templates according to the text/template
	// documentation.
	//
	// By default, FuncMap contains the color functions used to color the text in templates. If FuncMap
	// is overridden, the colors functions must be added in the override from promptui.FuncMap to work.
	FuncMap template.FuncMap

	prompt      *template.Template
	valid       *template.Template
	invalid     *template.Template
	validation  *template.Template
	success     *template.Template
	unvalidated *template.Template
}

// Run executes the prompt. Its displays the label and default value if any, asking the user to enter a value.
// Run will keep the prompt alive until it has been canceled from the command prompt or it has received a valid
// value. It will return the value and an error if any occurred during the prompt's execution.
func (p *Prompt) Run() (string, error) {
	var err error

	err = p.prepareTemplates()
	if err != nil {
		return "", err
	}

	c := &readline.Config{
		Stdin:        p.Stdin,
		Stdout:       p.Stdout,
		EnableMask:   p.Mask != 0,
		MaskRune:     p.Mask,
		HistoryLimit: -1,
		VimMode:      p.IsVimMode,
	}

	rl, err := readline.NewFromConfig(c)
	if err != nil {
		return "", err
	}
	// we're taking over the cursor, so stop showing it.
	rl.Write([]byte(hideCursor))

	validFn := func(x string) error {
		return nil
	}
	if p.Validate != nil {
		validFn = p.Validate
	}

	input := p.Default
	if p.IsConfirm {
		input = ""
	}
	eraseDefault := input != "" && !p.AllowEdit
	cur := NewCursor(input, p.Pointer, eraseDefault)

	listen := func(input []rune, pos int, key rune) ([]rune, int, bool) {
		_, _, keepOn := cur.Listen(input, pos, key)
		var prompt []byte

		if !p.LazyValidation {
			err := validFn(cur.Get())

			if err != nil {
				prompt = render(p.Templates.invalid, p.Label)
			} else {
				prompt = render(p.Templates.valid, p.Label)
				if p.IsConfirm {
					prompt = render(p.Templates.prompt, p.Label)
				}
			}
		} else {
			prompt = render(p.Templates.unvalidated, p.Label)
		}

		echo := cur.Format()
		if p.Mask != 0 {
			echo = cur.FormatMask(p.Mask)
		}

		prompt = append(prompt, []byte(echo)...)
		rl.SetPrompt(string(prompt))
		rl.Refresh()
		return nil, 0, keepOn
	}

	c.Listener = listen
	c.FuncFilterInputRune = func(r rune) (rune, bool) {
		switch r {
		case readline.CharEnter, readline.CharCtrlJ:
			err = validFn(cur.Get())
			if err != nil {
				validation := render(p.Templates.validation, err)
				rl.SetPrompt(string(validation))
				return r, false
			}
			return r, true
		}
		return r, true
	}

	_, err = rl.ReadLine()

	if err != nil {
		switch err {
		case readline.ErrInterrupt:
			err = ErrInterrupt
		case io.EOF:
			err = ErrEOF
		}
		if err.Error() == "Interrupt" {
			err = ErrInterrupt
		}
		return "", err
	}

	echo := cur.Get()
	if p.Mask != 0 {
		echo = cur.GetMask(p.Mask)
	}

	prompt := render(p.Templates.success, p.Label)
	prompt = append(prompt, []byte(echo)...)

	if p.IsConfirm {
		lowerDefault := strings.ToLower(p.Default)
		inputLower := strings.ToLower(cur.Get())
		if (lowerDefault == "y" && inputLower == "n") || (lowerDefault != "y" && inputLower != "y") {
			prompt = render(p.Templates.invalid, p.Label)
			err = ErrAbort
		}

	}

	rl.Write(prompt)
	rl.Write([]byte("\n"))
	rl.Write([]byte(showCursor))
	rl.Close()

	return cur.Get(), err
}

func (p *Prompt) prepareTemplates() error {
	tpls := p.Templates
	if tpls == nil {
		tpls = &PromptTemplates{}
	}

	if tpls.FuncMap == nil {
		tpls.FuncMap = FuncMap
	}

	bold := Styler(FGBold)

	if p.IsConfirm {
		if tpls.Confirm == "" {
			confirm := "y/N"
			if strings.ToLower(p.Default) == "y" {
				confirm = "Y/n"
			}
			tpls.Confirm = fmt.Sprintf(`{{ "%s" | bold }} {{ . | bold }}? {{ "[%s]" | faint }} `, IconInitial, confirm)
		}

		tpl, err := template.New("").Funcs(tpls.FuncMap).Parse(tpls.Confirm)
		if err != nil {
			return err
		}

		tpls.prompt = tpl
	} else {
		if tpls.Prompt == "" {
			tpls.Prompt = fmt.Sprintf("%s {{ . | bold }}%s ", bold(IconInitial), bold(":"))
		}

		tpl, err := template.New("").Funcs(tpls.FuncMap).Parse(tpls.Prompt)
		if err != nil {
			return err
		}

		tpls.prompt = tpl
	}

	if tpls.Valid == "" {
		tpls.Valid = fmt.Sprintf("%s {{ . | bold }}%s ", bold(IconGood), bold(":"))
	}

	tpl, err := template.New("").Funcs(tpls.FuncMap).Parse(tpls.Valid)
	if err != nil {
		return err
	}

	tpls.valid = tpl

	if tpls.Unvalidated == "" {
		tpls.Unvalidated = fmt.Sprintf("%s {{ . | bold }}%s ", bold(IconInitial), bold(":"))
	}

	tpl, err = template.New("").Funcs(tpls.FuncMap).Parse(tpls.Unvalidated)
	if err != nil {
		return err
	}

	tpls.unvalidated = tpl

	if tpls.Invalid == "" {
		tpls.Invalid = fmt.Sprintf("%s {{ . | bold }}%s ", bold(IconBad), bold(":"))
	}

	tpl, err = template.New("").Funcs(tpls.FuncMap).Parse(tpls.Invalid)
	if err != nil {
		return err
	}

	tpls.invalid = tpl

	if tpls.ValidationError == "" {
		tpls.ValidationError = `{{ ">>" | red }} {{ . | red }} {{ "Press any key to get back to the prompt" | faint }}`
	}

	tpl, err = template.New("").Funcs(tpls.FuncMap).Parse(tpls.ValidationError)
	if err != nil {
		return err
	}

	tpls.validation = tpl

	if tpls.Success == "" {
		tpls.Success = fmt.Sprintf("{{ . | faint }}%s ", Styler(FGFaint)(":"))
	}

	tpl, err = template.New("").Funcs(tpls.FuncMap).Parse(tpls.Success)
	if err != nil {
		return err
	}

	tpls.success = tpl

	p.Templates = tpls

	return nil
}
