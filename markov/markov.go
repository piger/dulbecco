package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jmhodges/levigo"
	"math/rand"
	"os"
	"strings"
	"time"
)

var (
	dbfile   = flag.String("db", "markov.db", "Path to database file")
	order    = flag.Int("order", 2, "N-Gram order")
	generate = flag.Bool("gen", false, "Generate some blabla")
)

const beginningKey = "*BEGINNING*"

func MakeKey(ngram []string) ([]byte, error) {
	return json.Marshal(ngram)
}

type MarkovDB struct {
	Order int
	Db    *levigo.DB
}

func NewMarkovDB(order int, dbfile string) (*MarkovDB, error) {
	opts := levigo.NewOptions()
	opts.SetCache(levigo.NewLRUCache(3 << 30))
	opts.SetCreateIfMissing(true)
	db, err := levigo.Open(dbfile, opts)
	if err != nil {
		return nil, err
	}

	mdb := &MarkovDB{
		Order: order,
		Db:    db,
	}

	return mdb, nil
}

func (mdb *MarkovDB) ReadFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		words := strings.Fields(scanner.Text())
		if len(words) < mdb.Order {
			continue
		}
		words = append(words, "\n")

		beginning := words[0:mdb.Order]
		err := mdb.PutBeginning(beginning)
		if err != nil {
			return err
		}

		for i := 0; i < len(words)-mdb.Order-1; i++ {
			ngram := words[i : i+mdb.Order]
			follow := words[i+mdb.Order]
			fmt.Printf("%v -> %s\n", ngram, follow)
			key, err := MakeKey(ngram)
			if err != nil {
				return err
			}
			mdb.Put(key, follow)
		}
		fmt.Println(".")
	}

	return nil
}

func (mdb *MarkovDB) Put(key []byte, value string) error {
	ro := levigo.NewReadOptions()
	wo := levigo.NewWriteOptions()
	defer ro.Close()
	defer wo.Close()

	// get existing value for key
	jsonwords, err := mdb.Db.Get(ro, key)
	if err != nil {
		return err
	}

	// deserialize existing value, if any
	var words []string
	if jsonwords != nil && string(jsonwords) != "" {
		err = json.Unmarshal(jsonwords, &words)
		if err != nil {
			return err
		}
	}

	// append the new word to the value and serialize the result
	words = append(words, value)
	newwords, err := json.Marshal(words)
	if err != nil {
		return err
	}

	// write the new value to the db
	err = mdb.Db.Put(wo, key, newwords)
	return err
}

type JsonBeginnings struct {
	Beginnings []*JsonBeginnings
}

type JsonBeginning struct {
	Ngrams []string
}

func (mdb *MarkovDB) PutBeginning(ngram []string) error {
	ro := levigo.NewReadOptions()
	wo := levigo.NewWriteOptions()
	defer ro.Close()
	defer wo.Close()

	jsondata, err := mdb.Db.Get(ro, []byte(beginningKey))
	if err != nil {
		return err
	}

	var jsonbeginnings [][]string
	if jsondata != nil && string(jsondata) != "" {
		err = json.Unmarshal(jsondata, &jsonbeginnings)
		if err != nil {
			return err
		}
	}
	jsonbeginnings = append(jsonbeginnings, ngram)

	newbeginnings, err := json.Marshal(jsonbeginnings)
	if err != nil {
		return nil
	}

	// fmt.Printf("newbeginnings = %s\n", newbeginnings)

	err = mdb.Db.Put(wo, []byte(beginningKey), newbeginnings)
	return err
}

func (mdb *MarkovDB) Generate() error {
	ro := levigo.NewReadOptions()
	defer ro.Close()

	jsonbeginnings, err := mdb.Db.Get(ro, []byte(beginningKey))
	if err != nil {
		return err
	}

	// fmt.Printf("jsonbeginnings = %s\n", jsonbeginnings)

	var beginnings [][]string
	err = json.Unmarshal(jsonbeginnings, &beginnings)
	if err != nil {
		return err
	}

	ngramKey := beginnings[rand.Intn(len(beginnings))]
	// fmt.Printf("ngramKey = %s\n", ngramKey)
	key, err := MakeKey(ngramKey)
	if err != nil {
		return err
	}

	for i := 0; i < 20; i++ {
		followWord, err := mdb.GetRandom(key)
		if err != nil {
			// no value for this key!
			break
		}
		fmt.Printf("%s ", followWord)
		ngramKey = append(ngramKey[1:], followWord)
		key, err = MakeKey(ngramKey)
		if err != nil {
			break
		}
	}

	fmt.Println()

	return nil
}

func (mdb *MarkovDB) GetRandom(key []byte) (string, error) {
	ro := levigo.NewReadOptions()
	defer ro.Close()

	jsondata, err := mdb.Db.Get(ro, key)
	if err != nil {
		return "", nil
	}

	var follows []string
	if jsondata == nil || string(jsondata) == "" {
		return "", fmt.Errorf("no value for this key: %s\n", key)
	}
	err = json.Unmarshal(jsondata, &follows)
	if err != nil {
		return "", err
	}

	word := follows[rand.Intn(len(follows))]

	return word, nil
}

func (mdb *MarkovDB) Close() {
	mdb.Db.Close()
}

func main() {
	flag.Parse()

	mdb, err := NewMarkovDB(*order, *dbfile)
	if err != nil {
		fmt.Print(err)
		return
	}

	if *generate {
		rand.Seed(time.Now().UnixNano())

		err = mdb.Generate()
		if err != nil {
			fmt.Print(err)
			return
		}
	} else {
		for _, filename := range flag.Args() {
			if err := mdb.ReadFile(filename); err != nil {
				fmt.Print(err)
				return
			}
		}
	}

	mdb.Close()
}
