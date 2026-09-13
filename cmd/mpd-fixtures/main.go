package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/leaanthony/mpress/internal/mpdconformance"
)

func main() {
	root := flag.String("root", ".", "MPress repository root")
	flag.Parse()
	absolute, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := mpdconformance.Regenerate(absolute); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Regenerated MPD HTML fixtures and viewer")
}
