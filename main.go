package main

import (
	"context"

	"github.com/Omotolani98/monocron-runner/cmd"
)

func main() {
	cmd.Execute(context.Background())
}
