package markov

import (
	"bufio"
	"os"
	"strings"
)

// To read from a IRC log file with a format like:
//
//    Dec 19 15:24:41 <user>    hello world!
//
// You can cleanup the log running:
//
// awk '$4 ~ /^</ { print }' < irc.log | cut -d '>' -f 2- | sed 's/^_TAB_//'
//
// Where _TAB_ is 'C-v TAB' from a bash shell.

// TODO: this use one transaction per line read

func (mdb *MarkovDB) LearnFromFile(dbpath, filename string, order int) error {
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
