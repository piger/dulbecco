package main

import (
	"flag"
	"github.com/boltdb/bolt"
	"github.com/jmhodges/levigo"
	"log"
	"os"
)

var (
	inFilename  = flag.String("in", "", "Input file")
	outFilename = flag.String("out", "", "Output file")
)

var bucketName []byte = []byte("markov")

func fileExists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		return false
	}
	return true
}

func migrateData(a, b string) error {
	var err error

	// open LevelDB file
	opts := levigo.NewOptions()
	opts.SetCache(levigo.NewLRUCache(3 << 29))
	opts.SetCreateIfMissing(true)
	ldb, err := levigo.Open(a, opts)
	if err != nil {
		return err
	}

	// Open BoltDB and create bucket
	bdb, err := bolt.Open(b, 0644, nil)
	if err != nil {
		return err
	}
	err = bdb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error creating bucket: %s", err)
	}

	// create the leveldb iterator
	ro := levigo.NewReadOptions()
	ro.SetFillCache(false)
	it := ldb.NewIterator(ro)
	defer it.Close()

	// do the migration!
	err = bdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

		for it.SeekToFirst(); it.Valid(); it.Next() {
			perr := b.Put(it.Key(), it.Value())
			if perr != nil {
				log.Printf("Error PUT key %q: %s\n", it.Key(), err)
				return err
			}
		}

		if gerr := it.GetError(); gerr != nil {
			log.Printf("Error reading leveldb: %s\n", gerr)
			return gerr
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error migrating data: %s\n", err)
	}

	return nil
}

func main() {
	flag.Parse()

	if *inFilename == "" || *outFilename == "" {
		flag.Usage()
		os.Exit(1)
	}

	if fileExists(*outFilename) {
		log.Fatalf("Output file already exists: %s\n", *outFilename)
	}

	if err := migrateData(*inFilename, *outFilename); err != nil {
		log.Fatalf("Error migrating data: %s\n", err)
	}
}
