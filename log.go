package main

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"strings"
)

func setupLogOutput() {
	// Setup Console Output
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	output.FormatLevel = formatLevel(noColor)
	output.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("| %-60s ", i)
	}
	log.Logger = log.Output(output)

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func formatLevel(noColor bool) zerolog.Formatter {
	return func(i interface{}) string {
		var l string
		if ll, ok := i.(string); ok {
			switch ll {
			case "debug":
				l = colorize("Debug", colorYellow, noColor)
			case "info":
				l = colorize("Info", colorGreen, noColor)
			case "warn":
				l = colorize("Warn", colorRed, noColor)
			case "error":
				l = colorize(colorize("Error", colorRed, noColor), colorBold, noColor)
			case "fatal":
				l = colorize(colorize("Fatal", colorRed, noColor), colorBold, noColor)
			case "panic":
				l = colorize(colorize("Panic", colorRed, noColor), colorBold, noColor)
			default:
				l = colorize("???", colorBold, noColor)
			}
		} else {
			l = strings.ToUpper(fmt.Sprintf("%s", i))[0:3]
		}
		return l
	}
}

// colorize returns the string s wrapped in ANSI code c, unless disabled is true.
func colorize(s interface{}, c int, disabled bool) string {
	if disabled {
		return fmt.Sprintf("%s", s)
	}
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
)
