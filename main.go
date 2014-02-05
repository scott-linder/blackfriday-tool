//
// Blackfriday Markdown Processor
// Available at http://github.com/russross/blackfriday
//
// Copyright © 2011 Russ Ross <russ@russross.com>.
// Distributed under the Simplified BSD License.
// See README.md for details.
//

//
//
// Example front-end for command-line use
//
//

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/russross/blackfriday"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime/pprof"
)

const (
	configFilename = "blackfriday.json"
)

var cfg = struct {
	Page        bool
	TOC         bool
	TOCOnly     bool
	XHTML       bool
	Latex       bool
	Smartypants bool
	LatexDashes bool
	Fractions   bool
	Footnotes   bool
	Title       string
	CSS         string
	CPUProfile  string
	Repeat      int
}{
	XHTML:       true,
	Smartypants: true,
	LatexDashes: true,
	Fractions:   true,
	Repeat:      1,
}

// Parse config file; error will never be never be NotExist
func parseConfig() error {
	paths := [...]string{
		path.Join(os.ExpandEnv("${XDG_CONFIG_HOME}"), configFilename),
		path.Join(os.ExpandEnv("${HOME}"), ".config", configFilename),
		path.Join("/etc", configFilename),
	}
	var configFile io.ReadCloser
	var err error
	for i, path := range paths {
		configFile, err = os.Open(path)
		if err != nil {
			if i == len(paths) {
				// exhausted all options
				return errors.New("config file not found")
			}
		} else {
			// opened file successfully
			break
		}
	}
	defer configFile.Close()
	return json.NewDecoder(configFile).Decode(&cfg)
}

// Parse flags
func parseFlags() {
	flag.BoolVar(&cfg.Page, "page", cfg.Page,
		"Generate a standalone HTML page (implies -latex=false)")
	flag.BoolVar(&cfg.TOC, "toc", cfg.TOC,
		"Generate a table of contents (implies -latex=false)")
	flag.BoolVar(&cfg.TOCOnly, "toconly", cfg.TOCOnly,
		"Generate a table of contents only (implies -toc)")
	flag.BoolVar(&cfg.XHTML, "xhtml", cfg.XHTML,
		"Use XHTML-style tags in HTML output")
	flag.BoolVar(&cfg.Latex, "latex", cfg.Latex,
		"Generate LaTeX output instead of HTML")
	flag.BoolVar(&cfg.Smartypants, "smartypants", cfg.Smartypants,
		"Apply smartypants-style substitutions")
	flag.BoolVar(&cfg.LatexDashes, "latexdashes", cfg.LatexDashes,
		"Use LaTeX-style dash rules for smartypants")
	flag.BoolVar(&cfg.Fractions, "fractions", cfg.Fractions,
		"Use improved fraction rules for smartypants")
	flag.BoolVar(&cfg.Footnotes, "footnotes", cfg.Footnotes,
		"Use Pandoc-style footnotes")
	flag.StringVar(&cfg.Title, "title", cfg.Title,
		"Explicit page title (implies -page)")
	flag.StringVar(&cfg.CSS, "css", cfg.CSS,
		"Link to a CSS stylesheet (implies -page)")
	flag.StringVar(&cfg.CPUProfile, "cpuprofile", cfg.CPUProfile,
		"Write cpu profile to a file")
	flag.IntVar(&cfg.Repeat, "repeat", cfg.Repeat,
		"Process the input multiple times (for benchmarking)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Blackfriday Markdown Processor v"+
			blackfriday.VERSION+
			"\nAvailable at http://github.com/russross/blackfriday\n\n"+
			"Copyright © 2011 Russ Ross <russ@russross.com>\n"+
			"Distributed under the Simplified BSD License\n"+
			"See website for details\n\n"+
			"Usage:\n"+
			"  %s [options] [inputfile [outputfile]]\n\n"+
			"Options:\n",
			os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
}

func main() {
	// parse config file and command-line options
	if err := parseConfig(); err != nil {
		log.Printf("warn: problem parsing config: %v", err)
	}
	parseFlags()

	// enforce implied options
	if cfg.CSS != "" || cfg.Title != "" {
		cfg.Page = true
	}
	if cfg.Page {
		cfg.Latex = false
	}
	if cfg.TOCOnly {
		cfg.TOC = true
	}
	if cfg.TOC {
		cfg.Latex = false
	}

	// turn on profiling?
	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// read the input
	var input []byte
	var err error
	args := flag.Args()
	switch len(args) {
	case 0:
		if input, err = ioutil.ReadAll(os.Stdin); err != nil {
			fmt.Fprintln(os.Stderr, "Error reading from Stdin:", err)
			os.Exit(-1)
		}
	case 1, 2:
		if input, err = ioutil.ReadFile(args[0]); err != nil {
			fmt.Fprintln(os.Stderr, "Error reading from", args[0], ":", err)
			os.Exit(-1)
		}
		// Use filename as title if there isn't one already
		if cfg.Title == "" {
			cfg.Title = args[0]
		}
	default:
		flag.Usage()
		os.Exit(-1)
	}

	// set up options
	extensions := 0
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS
	if cfg.Footnotes {
		extensions |= blackfriday.EXTENSION_FOOTNOTES
	}

	var renderer blackfriday.Renderer
	if cfg.Latex {
		// render the data into LaTeX
		renderer = blackfriday.LatexRenderer(0)
	} else {
		// render the data into HTML
		htmlFlags := 0
		if cfg.XHTML {
			htmlFlags |= blackfriday.HTML_USE_XHTML
		}
		if cfg.Smartypants {
			htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
		}
		if cfg.Fractions {
			htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
		}
		if cfg.LatexDashes {
			htmlFlags |= blackfriday.HTML_SMARTYPANTS_LATEX_DASHES
		}
		if cfg.Page {
			htmlFlags |= blackfriday.HTML_COMPLETE_PAGE
		}
		if cfg.TOCOnly {
			htmlFlags |= blackfriday.HTML_OMIT_CONTENTS
		}
		if cfg.TOC {
			htmlFlags |= blackfriday.HTML_TOC
		}
		renderer = blackfriday.HtmlRenderer(htmlFlags, cfg.Title, cfg.CSS)
	}

	// parse and render
	var output []byte
	for i := 0; i < cfg.Repeat; i++ {
		output = blackfriday.Markdown(input, renderer, extensions)
	}

	// output the result
	var out *os.File
	if len(args) == 2 {
		if out, err = os.Create(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v", args[1], err)
			os.Exit(-1)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	if _, err = out.Write(output); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing output:", err)
		os.Exit(-1)
	}
}
