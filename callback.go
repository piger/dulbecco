package dulbecco

import (
	"fmt"
	"strings"
	"reflect"
	"math/rand"
	"crypto/sha1"
)

func (c *Connection) AddCallback(name string, callback func(*Message)) string {
	name = strings.ToUpper(name)

	if _, ok := c.events[name]; !ok {
		c.events[name] = make(map[string]func(*Message))
	}

	hash := sha1.New()
	rawId := []byte(fmt.Sprintf("%v%d", reflect.ValueOf(callback).Pointer(), rand.Int63()))
	hash.Write(rawId)
	id := fmt.Sprintf("%x", hash.Sum(nil))
	c.events[name][id] = callback
	return id
}

func (c *Connection) RemoveCallback(name string, id string) bool {
	name = strings.ToUpper(name)

	if event, ok := c.events[name]; ok {
		if _, ok := event[id]; ok {
			delete(c.events[name], id)
			return true
		}
		fmt.Printf("No callback found in %s with id %s\n", name, id)
		return false
	}

	fmt.Printf("Event not found: %s\n", name)
	return false
}

func (c *Connection) RunCallbacks(message *Message) {
	if callbacks, ok := c.events[message.Cmd]; ok {
		for _, callback := range(callbacks) {
			go callback(message)
		}
	}

	// catch-all handlers
	if callbacks, ok := c.events["*"]; ok {
		for _, callback := range(callbacks) {
			go callback(message)
		}
	}
}

func (c *Connection) SetupCallbacks() {
	c.events = make(map[string]map[string]func(*Message))

	c.AddCallback("INIT", c.h_INIT)
	c.AddCallback("001", c.h_001)
}


// callbacks
func (c *Connection) h_INIT(message *Message) {
	fmt.Printf("Running handler h_INIT\n")

	c.Nick(c.nickname)
	c.User(c.username, c.realname)
}

// "Connected" event upon receiving numeric 001
func (c *Connection) h_001(message *Message) {
	c.Join("#puzza")
}
