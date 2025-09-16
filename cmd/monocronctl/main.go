package main

import (
	"context"

	"encore.app/cmd/monocronctl/cmd"
)

func main() {
	cmd.Execute(context.Background())
}
