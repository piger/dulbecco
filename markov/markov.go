package markov

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/jmhodges/levigo"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
)

var (
	MaxPhraseLength = 50
)

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

func MakeKey(ngram []string) ([]byte, error) {
	return json.Marshal(ngram)
}

type MarkovDB struct {
	Order int
	Db    *levigo.DB
	mutex sync.Mutex
}

func NewMarkovDB(order int, dbfile string) (*MarkovDB, error) {
	opts := levigo.NewOptions()
	opts.SetCache(levigo.NewLRUCache(3 << 29))
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

func (mdb *MarkovDB) ReadSentence(sentence string) {
	tokens, err := tokenize(mdb.Order, sentence)
	if err != nil {
		// log.Printf("ReadSentence error: %s\n", err)
		return
	}
	for _, token := range tokens {
		ngram := token[0 : len(token)-1]
		follow := token[len(token)-1]
		key, err := MakeKey(ngram)
		if err != nil {
			log.Printf("ReadSentence error: %s\n", err)
			return
		}
		if err := mdb.Put(key, follow); err != nil {
			log.Printf("ReadSentence put error: %s\n", err)
		}
	}
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
	for _, word := range words {
		if word == value {
			return nil
		}
	}
	words = append(words, value)
	newwords, err := json.Marshal(words)
	if err != nil {
		return err
	}

	// write the new value to the db
	// concurrent access is OK as long as we don't write the same key at the same time,
	// so better use a lock
	mdb.mutex.Lock()
	defer mdb.mutex.Unlock()

	err = mdb.Db.Put(wo, key, newwords)
	return err
}

func (mdb *MarkovDB) Generate(seed string) string {
	ro := levigo.NewReadOptions()
	defer ro.Close()

	var phrases []string

	tokens, err := tokenize(mdb.Order, seed)
	if err != nil {
		return ""
	}
	for _, token := range tokens {
		ngram := token[0 : len(token)-1]
		// fmt.Printf("ngram = %q\n", ngram)

		phrase := mdb.Goo(ngram)
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

func (mdb *MarkovDB) Goo(ngramKey []string) string {
	key, err := MakeKey(ngramKey)
	if err != nil {
		log.Println("Goo error (MakeKey):", err)
		return ""
	}
	var result []string = make([]string, len(ngramKey))
	copy(result, ngramKey)

	// fmt.Printf("result (1): %q (%q)\n", result, ngramKey)

	for i := 0; i < MaxPhraseLength; i++ {
		followWord, err := mdb.GetRandom(key)
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
			break
		}
	}
	return strings.Join(result, " ")
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

	// fmt.Printf("random for %q: %q\n", key, word)

	return word, nil
}

func (mdb *MarkovDB) Close() {
	mdb.Db.Close()
}

func TestMarkov() {
	mdb, err := NewMarkovDB(2, "petodb")
	if err != nil {
		log.Fatal(err)
	}
	defer mdb.Close()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		if strings.HasPrefix(text, "quit") {
			break
		}
		// mdb.ReadSentence(text)
		mdb.Generate(text)
	}
}

func ReadStdin(dbpath string, order int) {
	mdb, err := NewMarkovDB(order, dbpath)
	if err != nil {
		log.Fatal(err)
	}
	defer mdb.Close()

	i := 1
	var buf string

	reader := bufio.NewReader(os.Stdin)
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Print(err)
			break
		}

		text = strings.TrimRight(text, "\n")
		if text == "" {
			text = "\n"
		}

		if i%2 == 0 {
			fmt.Printf("Put %s -> %q\n", buf, text)
			mdb.Put([]byte(buf), text)
		} else {
			buf = text
		}

		i++
	}
}

// To read from a IRC log file with a format like:
//
//    Dec 19 15:24:41 <user>    hello world!
//
// You can cleanup the log running:
//
// awk '$4 ~ /^</ { print }' < irc.log | cut -d '>' -f 2- | sed 's/^_TAB_//'
//
// Where _TAB_ is 'C-v TAB' from a bash shell.

func ReadFile(dbpath, filename string, order int) error {
	mdb, err := NewMarkovDB(order, dbpath)
	if err != nil {
		log.Fatal(err)
	}
	defer mdb.Close()

	var reader *bufio.Reader

	if filename == "-" {
		reader = bufio.NewReader(os.Stdin)
	} else {
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		reader = bufio.NewReader(file)
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		mdb.ReadSentence(line)
	}
}
