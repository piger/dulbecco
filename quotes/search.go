package main

import (
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/codegangsta/cli"
	"strings"
)

func cmdSearch(ctx *cli.Context) {
	page := ctx.Int("page")
	args := strings.Join(ctx.Args(), " ")

	qdb.SearchQuote(args, page)
}

func (q *QuotesDB) SearchQuote(qstring string, page int) error {
	// query := bleve.NewQueryStringQuery(qstring)
	query := bleve.NewMatchQuery(qstring).SetField("quote")
	var from int
	if page > 1 {
		from = (page - 1) * maxResultsPerSearch
	}
	request := bleve.NewSearchRequestOptions(query, maxResultsPerSearch, from, false)
	request.Fields = append(request.Fields, []string{"id", "quote"}...)
	results, err := q.idx.Search(request)
	if err != nil {
		return err
	}

	if len(results.Hits) > 0 {
		totPages := float64(results.Total) / float64(maxResultsPerSearch)
		fmt.Printf("%d matches, showing page %d of %2.f\n", results.Total, page, totPages)

		for _, hit := range results.Hits {
			fmt.Printf("%s: %s\n", hit.ID, hit.Fields["quote"])
		}

		// fmt.Println(results)
	} else {
		fmt.Println("No matches")
	}

	return nil
}
