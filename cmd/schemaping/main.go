package main

import "github.com/rubensantoniorosa2704/schemaping-worker/cmd/schemaping/cli"

var version = "dev"

func main() {
	cli.Version = version
	cli.Execute()
}
