package main

import (
	"fmt"
	"os"

	"github.com/astepanovs/ztime/cmd"
	"github.com/astepanovs/ztime/db"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	conn, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ztime: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	store := db.NewStore(conn)
	cmd.Execute(store, version)
}
