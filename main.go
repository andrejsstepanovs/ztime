package main

import (
	"fmt"
	"os"

	"github.com/astepanovs/ztime/cmd"
	"github.com/astepanovs/ztime/db"
)

func main() {
	conn, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "time: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	store := db.NewStore(conn)
	cmd.Execute(store)
}
