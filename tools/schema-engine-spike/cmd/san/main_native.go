//go:build !js

package main

import (
	"fmt"
	"os"
)

func main() {
	b, err := result()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(append(b, '\n'))
}
