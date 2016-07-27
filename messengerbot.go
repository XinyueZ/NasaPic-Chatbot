package index

import (
	"net/http"

	"appengine"
	"appengine/urlfetch"
)

// Messenger is the main service which handles all callbacks from facebook
// Events are delivered to handlers if they are specified
type MessengerBot struct {
	MessageReceived  MessageReceivedHandler
	MessageDelivered MessageDeliveredHandler
	Postback         PostbackHandler
	Authentication   AuthenticationHandler
	Error            ErrorHandler

	VerifyToken string
	AppSecret   string // Optional: For validating integrity of messages
	AccessToken string
	PageId      string // Optional: For setting welcome message
	Debug       bool
	Client      *http.Client
}

func NewMessengerBot(r *http.Request, token string, verifyToken string) *MessengerBot {
	cxt := appengine.NewContext(r)
	gaeClient := urlfetch.Client(cxt)

	return &MessengerBot{
		AccessToken: token,
		VerifyToken: verifyToken,
		Debug:       false,
		Client:      gaeClient,
	}
}
