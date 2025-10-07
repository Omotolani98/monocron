package main

import (
	"context"
	"fmt"

	"github.com/Omotolani98/monocrond/cmd"
)

func main() {
	fmt.Println("Welcome to Monocrond")
	cmd.Execute(context.Background())
}
