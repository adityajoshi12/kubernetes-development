package main

import (
	"kubectl-decode/cmd"
	"log"
)

func main() {

	if err := cmd.NewDecodeCMD().Execute(); err != nil {
		log.Fatal(err)
	}
}
