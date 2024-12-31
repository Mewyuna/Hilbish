package util

import (
	"io"
)

type Streams struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin io.Reader
}
