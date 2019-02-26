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

var (
	Version string = "0.1.0"
	Build   string = ""
)

// Command line flags
var (
	debug              bool
	input              string
	output             string
	includeUnsupported bool
)

func init() {
	// init command line flags
	flag.BoolVarP(&debug, "debug", "d", false, "enable debug output")
	flag.StringVarP(&input, "filepath", "f", "-", `file or directory that contains the YAML configuration to convert. Use "-" to read from stdin.`)
	flag.StringVarP(&output, "output", "o", "-", `file or directory where Terraform config will be written`)
	flag.BoolVarP(&includeUnsupported, "include-unsupported", "I", false, `set to true to include unsupported Attributes / Blocks in the generated TF config`)

	// Setup Logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().Str("version", Version).Msg("k2tf")
	flag.Parse()

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func main() {
	objs := readInput()

	log.Debug().Int("count", len(objs)).Msg("read objects from input")

	w, closer := setupOutput()
	defer closer()

	for _, obj := range objs {
		f := hclwrite.NewEmptyFile()
		WriteObject(obj, f.Body())
		fmt.Fprint(w, string(f.Bytes()))
		fmt.Fprintln(w)
	}

}

func readInput() []runtime.Object {
	if input == "-" || input == "" {
		return readStdinInput()
	} else {
		return readFilesInput()
	}
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
	parsed, err := ParseK8SYAML(reader)
	if err != nil {
		log.Fatal().Err(err)
	}

	for _, obj := range parsed {
		if obj.GetObjectKind().GroupVersionKind().Kind == "List" {
			list := obj.(*corev1.List)
			for _, item := range list.Items {
				itemObj, err := ParseK8SJSON(item.Raw)
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
		obj, err := ParseK8SYAML(r)
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
		if _, err := os.Stat(output); os.IsExist(err) {
			log.Fatal().Str("file", output).Msg("output file already exists")
		}

		f, err := os.OpenFile(output, os.O_RDWR|os.O_CREATE, 0755)
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
