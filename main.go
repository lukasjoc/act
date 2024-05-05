package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/lukasjoc/act/internal/parse"
	"github.com/lukasjoc/act/internal/runtime"
)

func debugPrintItems(module parse.Module) {
	for _, item := range module {
		switch s := item.(type) {
		case parse.ActorStmt:
			fmt.Printf("[ACTOR] %v %v\n", s.Ident, s.State)
			for _, action := range s.Actions {
				fmt.Printf("  [ACTION] %v\n", action)
			}
		case parse.SendStmt:
			fmt.Printf("[SEND] %v %v %v\n", s.SendPid, s.Message, s.Args)
		case parse.SpawnStmt:
			fmt.Printf("[SPAWN] %v %v\n", s.PidIdent, s.Scope)
		default:
		}
	}
}

func main() {
	// !TODO: better filepath validation of input
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %v ./path/to/file.act\n", os.Args[0])
		os.Exit(1)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	module, err := parse.New(bufio.NewReader(f))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Could not parse file: %v\n", err)
		os.Exit(1)
	}
	debugPrintItems(module)

	env := runtime.New(module)
	if err := env.Exec(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR RUNTIME]: %v\n", err)
		os.Exit(1)
	}
}
