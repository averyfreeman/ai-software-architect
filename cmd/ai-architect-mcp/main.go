package main

import (
	"context"
	"fmt"
	"os"

	"github.com/averyfreeman/git-bbq/internal/mcpserver"
)

func main() {
	if err := mcpserver.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "ai-architect-mcp:", err)
		os.Exit(1)
	}
}
