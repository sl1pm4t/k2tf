package main

import (
	"fmt"
	"os"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/hashicorp/hcl2/hclwrite"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Build time variables
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Command line flags
var (
	debug              bool
	input              string
	output             string
	includeUnsupported bool
	noColor            bool
	overwriteExisting  bool
)

func init() {
	// init command line flags
	flag.BoolVarP(&overwriteExisting, "overwrite-existing", "x", false, "allow overwriting existing output file(s)")
	flag.BoolVarP(&debug, "debug", "d", false, "enable debug output")
	flag.StringVarP(&input, "filepath", "f", "-", `file or directory that contains the YAML configuration to convert. Use "-" to read from stdin.`)
	flag.StringVarP(&output, "output", "o", "-", `file or directory where Terraform config will be written`)
	flag.BoolVarP(&includeUnsupported, "include-unsupported", "I", false, `set to true to include unsupported Attributes / Blocks in the generated TF config`)

	flag.Parse()

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

func main() {
	log.Debug().
		Str("version", version).
		Str("commit", commit).
		Str("builddate", date).
		Msg("starting k2tf")

	objs := readInput()

	log.Debug().Msgf("read %d objects from input", len(objs))

	w, closer := setupOutput()
	defer closer()

	for i, obj := range objs {
		f := hclwrite.NewEmptyFile()
		err := WriteObject(obj, f.Body())
		if err != nil {
			log.Error().Int("obj#", i).Err(err).Msg("error writing object")
		}

		fmt.Fprint(w, string(f.Bytes()))
		fmt.Fprintln(w)
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
