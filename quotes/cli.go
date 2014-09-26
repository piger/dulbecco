package main

import (
	"github.com/codegangsta/cli"
	"log"
	"os"
)

func OpenQuotesDB(ctx *cli.Context) *QuotesDB {
	dbfile := ctx.GlobalString("dbfile")
	indexdir := ctx.GlobalString("indexdir")
	if dbfile == "" || indexdir == "" {
		log.Fatal("You must specify a dbfile and an indexdir")
	}
	qdb := NewQuotesDB(dbfile, indexdir)
	if err := qdb.Open(); err != nil {
		log.Fatalf("Error opening databases: %s\n", err)
	}

	return qdb
}

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
