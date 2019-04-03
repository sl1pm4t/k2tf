package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/hashicorp/hcl2/hclwrite"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
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

	log.Debug().Int("count", len(objs)).Msg("read objects from input")

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

func readInput() []runtime.Object {
	if input == "-" || input == "" {
		return readStdinInput()
	}
	return readFilesInput()
}

func readStdinInput() []runtime.Object {
	var objs []runtime.Object

	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if info.Mode()&os.ModeCharDevice != 0 {
		log.Fatal().Msg("No data read from stdin")
	}

	reader := bufio.NewReader(os.Stdin)
	parsed, err := parseK8SYAML(reader)
	if err != nil {
		log.Fatal().Err(err)
	}

	for _, obj := range parsed {
		if obj.GetObjectKind().GroupVersionKind().Kind == "List" {
			list := obj.(*corev1.List)
			for _, item := range list.Items {
				itemObj, err := parseK8SJSON(item.Raw)
				if err != nil {
					log.Error().Err(err)
					continue
				}
				objs = append(objs, itemObj)

			}

		} else {
			objs = append(objs, obj)

		}
	}

	return objs
}

func readFilesInput() []runtime.Object {
	var objs []runtime.Object

	if _, err := os.Stat(input); os.IsNotExist(err) {
		log.Fatal().Str("file", input).Msg("input filepath does not exist")
	}

	file, err := os.Open(input)
	if err != nil {
		log.Fatal().Err(err)
	}

	fs, err := file.Stat()
	if err != nil {
		log.Fatal().Err(err)
	}

	readFile := func(fileName string) {
		content, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatal().Err(err)
		}

		r := bytes.NewReader(content)
		obj, err := parseK8SYAML(r)
		if err != nil {
			log.Fatal().Err(err)
		}
		objs = append(objs, obj...)
	}

	if fs.Mode().IsDir() {
		// read directory
		dirContents, err := file.Readdirnames(0)
		if err != nil {
			log.Fatal().Err(err)
		}

		for _, f := range dirContents {
			if strings.HasSuffix(f, ".yml") || strings.HasSuffix(f, ".yaml") {
				readFile(filepath.Join(fs.Name(), f))
			}
		}

	} else {
		// read single file
		readFile(input)

	}

	return objs
}

var noOpCloser = func() {}

func setupOutput() (io.Writer, func()) {
	if output != "" && output != "-" {
		// writing to a file

		if _, err := os.Stat(output); err == nil && !overwriteExisting {
			// don't clobber
			log.Fatal().Str("file", output).Msg("output file already exists")
		}

		f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Fatal().Err(err)
		}
		log.Debug().Str("file", output).Msg("opened file")

		closeFn := func() {
			if err := f.Close(); err != nil {
				log.Fatal().Err(err)
			}
			log.Debug().Str("file", output).Msg("closed output file")
		}

		return f, closeFn
	}

	log.Debug().Msg("outputting HCL to Stdout")
	return os.Stdout, noOpCloser
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
