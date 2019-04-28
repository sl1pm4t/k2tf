package main

import (
	"fmt"
	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/hashicorp/hcl2/hclwrite"
	flag "github.com/spf13/pflag"

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
	tf12format bool
)

func init() {
	// init command line flags
	flag.BoolVarP(&overwriteExisting, "overwrite-existing", "x", false, "allow overwriting existing output file(s)")
	flag.BoolVarP(&debug, "debug", "d", false, "enable debug output")
	flag.StringVarP(&input, "filepath", "f", "-", `file or directory that contains the YAML configuration to convert. Use "-" to read from stdin`)
	flag.StringVarP(&output, "output", "o", "-", `file or directory where Terraform config will be written`)
	flag.BoolVarP(&includeUnsupported, "include-unsupported", "I", false, `set to true to include unsupported Attributes / Blocks in the generated TF config`)
	flag.BoolVarP(&tf12format, "tf12format", "F", false, `Use Terraform 0.12 formatter`)

	flag.Parse()

	setupLogOutput()
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

		formatted := formatObject(f.Bytes())

		fmt.Fprint(w, string(formatted))
		fmt.Fprintln(w)
	}
}

func formatObject(in []byte) ([]byte) {
	var result []byte
	var err error

	if tf12format {
		result = hclwrite.Format(in)
	} else {
		result, err = printer.Format(in)
		if err != nil {
			log.Error().Err(err).Msg("could not format object")
			return in
		}
	}

	return result
}