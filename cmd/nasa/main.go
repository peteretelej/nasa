package main

import (
	"fmt"
	"io"
	"os"
)

var version = "dev"

func main() {
	// Handle global flags before subcommand dispatch.
	// --version: print version and exit 0
	// --help or -h: print usage and exit 0
	// No args: print usage and exit 1

	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--version", "-version":
		fmt.Println(version)
		return
	case "--help", "-help", "-h":
		printUsage(os.Stdout)
		return
	case "apod":
		apodCmd(os.Args[2:])
	case "neo":
		neoCmd(os.Args[2:])
	case "donki":
		donkiCmd(os.Args[2:])
	case "images":
		imagesCmd(os.Args[2:])
	case "ssd":
		ssdCmd(os.Args[2:])
	case "eonet":
		eonetCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "nasa: unknown command %q\n\n", os.Args[1])
		printUsage(os.Stderr)
		os.Exit(1)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `Usage: nasa <command> [flags]

Commands:
  apod      Astronomy Picture of the Day
  neo       Near-Earth Object Web Service
  donki     Space Weather (DONKI)
  images    NASA Image and Video Library
  ssd       Small-Body Database (SSD/CNEOS)
  eonet     Earth Observatory Natural Event Tracker

Global flags:
  --version   Print version and exit
  --help      Show this help message

Use "nasa <command> --help" for more information about a command.
`)
}
