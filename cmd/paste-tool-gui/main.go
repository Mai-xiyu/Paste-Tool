package main

import (
	"context"
	"os"

	"github.com/Mai-xiyu/Paste-Tool/internal/gui"
)

func main() {
	if err := gui.Run(context.Background()); err != nil {
		os.Exit(1)
	}
}
