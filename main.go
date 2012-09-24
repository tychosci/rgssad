package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 2 {
		usage()
		os.Exit(1)
	}

	cmd := flag.Arg(0)
	filename := flag.Arg(1)

	if cmd != "list" && cmd != "save" {
		usage()
		os.Exit(1)
	}

	rgssad, err := Extract(filename)
	if err != nil {
		fatal(err)
	}
	defer rgssad.Close()

	if cmd == "list" {
		rgssad.Show()
	} else if cmd == "save" {
		rgssad.Save()
	}
}

func usage() {
	fmt.Println("usage: rgssad [list|save] filename")
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "rgssad: %v\n", err)
	os.Exit(1)
}
