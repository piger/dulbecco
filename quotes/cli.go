package main

import (
	"github.com/codegangsta/cli"
	"os"
)

var qdb *QuotesDB

func OpenQuotesDB(ctx *cli.Context) error {
	qdb = NewQuotesDB(ctx.GlobalString("dbfile"), ctx.GlobalString("indexdir"))
	return qdb.Open()
}

func main() {
	app := cli.NewApp()
	app.Name = "pinolo-quotes"
	app.Usage = "pinolo quotes plugin"
	app.Version = "0.1.1"
	app.Before = OpenQuotesDB

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "dbfile",
			Value: "db.sqlite",
			Usage: "Path to DB file",
		},
		cli.StringFlag{
			Name:  "indexdir",
			Value: "idx",
			Usage: "Path to index directory",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "search",
			Usage:  "search a quote",
			Action: cmdSearch,
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
			Action: cmdGetQuote,
		},
		{
			Name:   "random",
			Usage:  "get random quote",
			Action: cmdGetRandomQuote,
		},
		{
			Name:   "add",
			Usage:  "add a quote",
			Action: cmdAddQuote,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "author",
					Value: "",
					Usage: "Quote author",
				},
			},
		},
	}

	app.Run(os.Args)
}
