package quotes

import (
	"database/sql"
	"github.com/blevesearch/bleve"
	"github.com/codegangsta/cli"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strconv"
)

var schemaStmt = `CREATE TABLE if not exists quotes
(id INTEGER NOT NULL, creation_date DATETIME, author VARCHAR, quote TEXT, karma INTEGER, PRIMARY KEY (id))`

type QuotesDB struct {
	DbFile   string
	IndexDir string

	db  *sql.DB
	idx bleve.Index
}

func OpenQuotesDB(ctx *cli.Context) *QuotesDB {
	dbfile := ctx.GlobalString("dbfile")
	indexdir := ctx.GlobalString("indexdir")
	if dbfile == "" || indexdir == "" {
		log.Fatal("You must specify a dbfile and an indexdir")
	}
	qdb := &QuotesDB{DbFile: dbfile, IndexDir: indexdir}
	if err := qdb.Open(); err != nil {
		log.Fatalf("Error opening databases: %s\n", err)
	}

	return qdb
}

type Quote struct {
	Id     int    `json:"id"`
	Author string `json:"author"`
	Quote  string `json:"quote"`
	Karma  int    `json:"karma"`
}

func (q *Quote) Type() string {
	return "quote"
}

func (q *QuotesDB) Open() error {
	var dbExists bool
	if _, err := os.Stat(q.DbFile); err == nil {
		dbExists = true
	}

	db, err := sql.Open("sqlite3", q.DbFile)
	if err != nil {
		return err
	}
	q.db = db

	if !dbExists {
		if err := q.SetupDBSchema(); err != nil {
			return err
		}
	}

	index, err := bleve.Open(q.IndexDir)
	if err == bleve.ErrorIndexPathDoesNotExist {
		indexMapping := buildIndexMapping()
		index, err = bleve.New(q.IndexDir, indexMapping)
		if err != nil {
			return err
		}
		q.idx = index

		// we have to index the db!
		log.Println("Indexing database")
		if err := q.IndexAll(); err != nil {
			return err
		}
		log.Println("Indexing done")
	} else if err != nil {
		return err
	} else {
		q.idx = index
	}

	return nil
}

func (q *QuotesDB) Close() {
	q.db.Close()
	q.idx.Close()
}

func (q *QuotesDB) SetupDBSchema() error {
	if _, err := q.db.Exec(schemaStmt); err != nil {
		return err
	}
	return nil
}

func (q *QuotesDB) IndexAll() error {
	rows, err := q.db.Query("SELECT id, author, quote, karma FROM quotes")
	if err != nil {
		return err
	}
	defer rows.Close()

	var i int
	batch := q.idx.NewBatch()
	for rows.Next() {
		quote := &Quote{}
		if err := rows.Scan(&quote.Id, &quote.Author, &quote.Quote, &quote.Karma); err != nil {
			log.Printf("Error reading quote %+v: %s\n", quote, err)
			// return err
		}
		id := strconv.Itoa(quote.Id)
		batch.Index(id, quote)

		if i > 100 {
			if err := q.idx.Batch(batch); err != nil {
				return err
			}
			i = 0
			batch = q.idx.NewBatch()
		} else {
			i++
		}
	}

	// index also the last batch!
	if i > 0 {
		if err := q.idx.Batch(batch); err != nil {
			return err
		}
	}

	return nil
}

func buildIndexMapping() *bleve.IndexMapping {
	itTextMapping := bleve.NewTextFieldMapping()
	itTextMapping.Analyzer = "it"
	stdTextMapping := bleve.NewTextFieldMapping()
	stdTextMapping.Analyzer = "simple"

	qm := bleve.NewDocumentMapping()
	qm.AddSubDocumentMapping("id", bleve.NewDocumentDisabledMapping())
	qm.AddFieldMappingsAt("author", stdTextMapping)
	qm.AddFieldMappingsAt("quote", itTextMapping)

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("quote", qm)
	mapping.DefaultAnalyzer = "it"
	// mapping.TypeField = "quote"
	return mapping
}
