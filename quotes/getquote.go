package main

import (
	"database/sql"
	"fmt"
	"github.com/codegangsta/cli"
)

func cmdGetQuote(ctx *cli.Context) {
	qdb := OpenQuotesDB(ctx)

	id := ctx.Args().First()
	if id == "" {
		fmt.Println("ma de che?")
		return
	}
	quote, err := qdb.GetQuote(id)
	if err == sql.ErrNoRows {
		fmt.Println("Te stai popo che a sbàja")
		return
	} else if err != nil {
		fmt.Printf("error getting quote: %s\n", err)
		return
	}
	fmt.Printf("%d: %s\n", quote.Id, quote.Quote)
}

func cmdGetRandomQuote(ctx *cli.Context) {
	qdb := OpenQuotesDB(ctx)

	quote, err := qdb.GetRandomQuote()
	if err != nil {
		fmt.Printf("error getting quote: %s\n", err)
		return
	}
	fmt.Printf("%d: %s\n", quote.Id, quote.Quote)
}

func (q *QuotesDB) GetQuote(id string) (*Quote, error) {
	stmt, err := q.db.Prepare("SELECT id, author, quote, karma FROM quotes WHERE id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(id)
	quote := &Quote{}
	err = row.Scan(&quote.Id, &quote.Author, &quote.Quote, &quote.Karma)

	return quote, err
}

func (q *QuotesDB) GetRandomQuote() (*Quote, error) {
	quote := &Quote{}
	row := q.db.QueryRow("SELECT id, author, quote, karma FROM quotes ORDER BY RANDOM() LIMIT 1")
	err := row.Scan(&quote.Id, &quote.Author, &quote.Quote, &quote.Karma)
	return quote, err
}
