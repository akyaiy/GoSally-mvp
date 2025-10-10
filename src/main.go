// Package main used only for calling cmd.Execute()
package main

import (
	"github.com/akyaiy/GoSally-mvp/src/cmd"
	_ "modernc.org/sqlite"
)

func main() {
	cmd.Execute()
}
