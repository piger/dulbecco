package quotes

import (
	"database/sql"
	"fmt"
	"github.com/codegangsta/cli"
)

func CmdGetQuote(ctx *cli.Context) {
	qdb := OpenQuotesDB(ctx)

	id := ctx.Args().First()
	if id == "" {
		fmt.Println("ma de che?")
		return
	}
	quote, err := qdb.getQuote(id)
	if err == sql.ErrNoRows {
		fmt.Println("Te stai popo che a sbàja")
		return
	} else if err != nil {
		fmt.Printf("error getting quote: %s\n", err)
		return
	}
	fmt.Printf("%d: %s\n", quote.Id, quote.Quote)
}

func CmdGetRandomQuote(ctx *cli.Context) {
	qdb := OpenQuotesDB(ctx)

	quote, err := qdb.getRandomQuote()
	if err != nil {
		fmt.Printf("error getting quote: %s\n", err)
		return
	}
	fmt.Printf("%d: %s\n", quote.Id, quote.Quote)
}

func (q *QuotesDB) getQuote(id string) (*Quote, error) {
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

func (q *QuotesDB) getRandomQuote() (*Quote, error) {
	quote := &Quote{}
	row := q.db.QueryRow("SELECT id, author, quote, karma FROM quotes ORDER BY RANDOM() LIMIT 1")
	err := row.Scan(&quote.Id, &quote.Author, &quote.Quote, &quote.Karma)
	return quote, err
}
