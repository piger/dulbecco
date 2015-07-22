package dulbecco

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/piger/dulbecco/markov"
	"io"
	"log"
	"net/http"
	"strings"
)

// See: https://www.hipchat.com/docs/apiv2/webhooks#room_message

type HipchatRequest struct {
	Event         string
	Item          *HipchatItem
	WebhookId     int    `json:"webhook_id"`
	OauthClientId string `json:"oauth_client_id"`
}

func (hcr *HipchatRequest) GetRoomMessage() string {
	return hcr.Item.Message.Message
}

type HipchatItem struct {
	Message *HipchatMessage
	Room    *HipchatRoom
}

type HipchatUser struct {
	Id          int
	MentionName string `json:"mention_name"`
	Name        string
}

type HipchatMessage struct {
	Date    string
	From    *HipchatUser
	Id      string
	Message string
	Type    string
}

type HipchatRoom struct {
	Id   int
	Name string
}

// web handler stuff

type AppContext struct {
	MarkovDB *markov.MarkovDB
}

type vRequest struct {
	Ctx       *AppContext
	Request   *http.Request
	HCRequest *HipchatRequest
}

type vHandler interface {
	ServeHTTP(w http.ResponseWriter, r *vRequest)
}

type vHandlerFunc func(w http.ResponseWriter, r *vRequest)

func (f vHandlerFunc) ServeHTTP(w http.ResponseWriter, r *vRequest) {
	f(w, r)
}

// Wrapper for our HTTP handlers
func WithRequest(ac *AppContext, h vHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hr, err := decodeHipchatRequest(r.Body)
		if err != nil {
			log.Printf("Cannot decode JSON request: %s\n", err)
			http.Error(w, "ERROR", http.StatusInternalServerError)
			return
		}

		vreq := vRequest{
			Ctx:       ac,
			Request:   r,
			HCRequest: hr,
		}
		h.ServeHTTP(w, &vreq)
	})
}

// strip a prefix *and the following character* (usually a whitespace)
func stripPrefix(s, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix)+1:]
	}
	return s
}

// Contains the field required for a proper response for an Hipchat
// integration.
type CommandResponse struct {
	Color         string `json:"color"`
	Message       string `json:"message"`
	Notify        bool   `json:"notify"`
	MessageFormat string `json:"message_format"`
}

func NewCommandResponse(message string) *CommandResponse {
	cr := CommandResponse{
		Message:       message,
		MessageFormat: "text",
	}
	return &cr
}

// Decode a JSON request from a Hipchat server
func decodeHipchatRequest(br io.Reader) (*HipchatRequest, error) {
	decoder := json.NewDecoder(br)
	var hr HipchatRequest
	if err := decoder.Decode(&hr); err != nil {
		return nil, err
	}
	return &hr, nil
}

// Write a JSON response back to a Hipchat server
func sendJSONResponse(w http.ResponseWriter, resp *CommandResponse) {
	w.Header().Set("Content-Type", "application/json")
	jresp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Cannot encode JSON response: %s\n", err)
		http.Error(w, "ERROR", http.StatusInternalServerError)
		return
	}
	w.Write(jresp)
}

// handler for /pinolo
func talkHandler(w http.ResponseWriter, r *vRequest) {
	var reply string
	message := stripPrefix(r.HCRequest.GetRoomMessage(), "/pinolo")
	if message == "" {
		reply = GetRandomReply()
	} else {
		reply = r.Ctx.MarkovDB.Generate(message)
		if reply == message || len(reply) == 0 {
			reply = GetRandomReply()
		}
	}
	resp := NewCommandResponse(reply)
	sendJSONResponse(w, resp)
}

// Main loop, must be called from a goroutine
func HipchatHandler(address string, mdb *markov.MarkovDB) error {
	ac := &AppContext{
		MarkovDB: mdb,
	}
	r := mux.NewRouter()
	hr := r.PathPrefix("/hipchat/").Subrouter()
	hr.Handle("/talk", WithRequest(ac, vHandlerFunc(talkHandler)))

	http.Handle("/", r)
	return http.ListenAndServe(address, nil)
}
