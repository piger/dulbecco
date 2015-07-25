package markov

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var (
	MaxPhraseLength = 50
)

const bucketName = "markov"

func BucketName() []byte {
	return []byte(bucketName)
}

// Create a key suitable for MarkovDB
func MakeKey(ngram []string) ([]byte, error) {
	return json.Marshal(ngram)
}

func tokenize(order int, sentence string) ([][]string, error) {
	var result [][]string

	words := strings.Fields(sentence)
	if len(words) < order {
		return result, fmt.Errorf("Sentence too short for order %d\n", order)
	}
	words = append(words, "\n")

	for i := 0; i < len(words)-order; i++ {
		ngram := words[i : i+order+1]
		result = append(result, ngram)
	}
	return result, nil
}

type MarkovDB struct {
	Order int
	Db    *bolt.DB
	mutex sync.Mutex
}

func NewMarkovDB(order int, dbfile string) (*MarkovDB, error) {
	opts := bolt.Options{Timeout: 1 * time.Second}
	db, err := bolt.Open(dbfile, 0600, &opts)
	if err != nil {
		return nil, err
	}

	mdb := &MarkovDB{
		Order: order,
		Db:    db,
	}
	if err := mdb.createBucket(bucketName); err != nil {
		return nil, err
	}

	return mdb, nil
}

func (mdb *MarkovDB) createBucket(name string) error {
	err := mdb.Db.Update(func(tx *bolt.Tx) error {
		_, berr := tx.CreateBucketIfNotExists([]byte(name))
		return berr
	})
	return err
}

// Learn from the given sentence
func (mdb *MarkovDB) ReadSentence(sentence string) {
	tokens, err := tokenize(mdb.Order, sentence)
	if err != nil {
		return
	}

	mdb.mutex.Lock()
	defer mdb.mutex.Unlock()

	err = mdb.Db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(BucketName())

		for _, token := range tokens {
			ngram := token[0 : len(token)-1]
			follow := token[len(token)-1]
			key, err := MakeKey(ngram)
			if err != nil {
				return err
			}
			if err := mdb.updateValue(key, follow, bkt); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("ReadSentence error: %s\n", err)
	}
}

func (mdb *MarkovDB) updateValue(key []byte, value string, bkt *bolt.Bucket) error {
	var jsonwords []byte

	// get existing value for key
	jsonwords = bkt.Get(key)

	// deserialize existing value, if any
	var words []string
	if jsonwords != nil && len(jsonwords) > 0 {
		err := json.Unmarshal(jsonwords, &words)
		if err != nil {
			return err
		}
	}

	// short circuit if the value is already there
	for _, word := range words {
		if word == value {
			return nil
		}
	}

	// append the new word to the value and serialize the result
	words = append(words, value)
	newwords, err := json.Marshal(words)
	if err != nil {
		return err
	}

	return bkt.Put(key, newwords)
}

// Generate a phrase
func (mdb *MarkovDB) Generate(seed string) string {
	var phrases []string

	tokens, err := tokenize(mdb.Order, seed)
	if err != nil {
		return ""
	}
	for _, token := range tokens {
		ngram := token[0 : len(token)-1]
		// fmt.Printf("ngram = %q\n", ngram)

		phrase, err := mdb.generatePhrase(ngram)
		if err != nil {
			log.Printf("Generate error: %s\n", err)
			return ""
		}

		if phrase == seed {
			continue
		}
		phrases = append(phrases, phrase)
		// fmt.Printf("* phrase %d: %s\n", i, phrase)
	}

	var result string
	for _, phrase := range phrases {
		if len(phrase) > len(result) {
			result = phrase
		}
	}

	// fmt.Printf("%s\n", result)

	return result
}

func (mdb *MarkovDB) generatePhrase(ngramKey []string) (string, error) {
	key, err := MakeKey(ngramKey)
	if err != nil {
		return "", err
	}

	var result []string = make([]string, len(ngramKey))
	copy(result, ngramKey)

	// fmt.Printf("result (1): %q (%q)\n", result, ngramKey)

	err = mdb.Db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(BucketName())

		for i := 0; i < MaxPhraseLength; i++ {
			followWord, err := mdb.getRandom(key, bkt)
			if err != nil || followWord == "\n" {
				// fmt.Printf("no follow for %s\n", key)
				break
			}
			if followWord == "\n" {
				break
			}
			result = append(result, followWord)
			ngramKey = append(ngramKey[1:], followWord)
			key, err = MakeKey(ngramKey)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return strings.Join(result, " "), nil
}

// NOTE: the following function runs inside a bolt.Tx!

func (mdb *MarkovDB) getRandom(key []byte, bkt *bolt.Bucket) (string, error) {
	var jsonwords []byte

	// get existing value for key
	jsonwords = bkt.Get(key)

	// deserialize existing value, if any
	var words []string
	if jsonwords != nil && len(jsonwords) > 0 {
		err := json.Unmarshal(jsonwords, &words)
		if err != nil {
			return "", err
		}
	}

	return words[rand.Intn(len(words))], nil
}

func (mdb *MarkovDB) Close() error {
	return mdb.Db.Close()
}
