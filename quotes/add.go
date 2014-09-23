package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"strconv"
	"strings"
	"time"
)

func cmdAddQuote(ctx *cli.Context) {
	author := ctx.String("author")
	if author == "" {
		fmt.Println("You must specify an author")
		return
	}
	quoteText := strings.Join(ctx.Args(), " ")

	stmt, err := qdb.db.Prepare("INSERT INTO quotes(creation_date, author, quote, karma) VALUES (?, ?, ?, ?)")
	if err != nil {
		fmt.Printf("error preparing SQL query: %s", err)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(time.Now(), author, quoteText, 0)
	if err != nil {
		fmt.Printf("error executing SQL query: %s", err)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		fmt.Println("cannot get the inserted quote ID")
		return
	}
	strId := strconv.Itoa(int(id))

	quote := &Quote{Id: int(id), Author: author, Quote: quoteText, Karma: 0}
	qdb.idx.Index(strId, quote)

	fmt.Printf("Added quote %d\n", id)
}
