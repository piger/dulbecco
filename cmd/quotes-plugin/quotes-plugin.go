package main

import (
	"github.com/codegangsta/cli"
	"github.com/piger/dulbecco/quotes"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "pinolo-quotes"
	app.Usage = "pinolo quotes plugin"
	app.Version = "0.1.2"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "dbfile",
			Value: "",
			Usage: "Path to DB file",
		},
		cli.StringFlag{
			Name:  "indexdir",
			Value: "",
			Usage: "Path to index directory",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "search",
			Usage:  "search a quote",
			Action: quotes.CmdSearch,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "page",
					Value: 1,
					Usage: "Page number",
				},
			},
		},
		{
			Name:   "get",
			Usage:  "get quote by ID",
			Action: quotes.CmdGetQuote,
		},
		{
			Name:   "random",
			Usage:  "get random quote",
			Action: quotes.CmdGetRandomQuote,
		},
		{
			Name:   "add",
			Usage:  "add a quote",
			Action: quotes.CmdAddQuote,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "author",
					Value:  "",
					Usage:  "Quote author",
					EnvVar: "IRC_NICKNAME",
				},
			},
		},
	}

	app.Run(os.Args)
}
