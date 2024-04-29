package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/lukasjoc/act/internal/parse"
)

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
	}
	for _, item := range (*module).Items {
		switch item.Type() {
		case parse.ModuleItemActor:
			fmt.Println("[STMT ACTOR]: ", item.(parse.ActorStmt).Ident)
		case parse.ModuleItemSend:
			fmt.Println("[STMT SEND]: ", item.(parse.SendStmt).ActorIdent)
		case parse.ModuleItemShow:
			fmt.Println("[STMT SHOW]: ", item.(parse.ShowStmt).ActorIdent)
		default:
			panic("we cant print that type yet")
		}
	}
}
