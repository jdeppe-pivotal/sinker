package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"log"
)

func init() {
	app.Commands = append(
		app.Commands,
		cli.Command{
			Name:      "list",
			ShortName: "l",
			Usage:     "list <target dir>",
			Action:    List,
		},
	)
}

func List(c *cli.Context) {
	client, err := getSftpClient(c)
	if err != nil {
		log.Fatal(err)
	}

	walker := client.Walk(c.Args()[0])
	for walker.Step() {
		fmt.Printf("%s\n", walker.Path())
	}
}
