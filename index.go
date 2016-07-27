package index

import (
	"appengine"
	"net/http"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

const (
	API_INDEX_START = 2
	API_INDEX_END   = 6
	ERR             = "Sick, sorry I have some internal problems. :(. Try again or later."
	//Buttons
	BTN_PAYLOAD_YES = "Yes, show something."
	BTN_PAYLOAD_NO  = "No, thanks"

	//Payloads
	PAYLOAD_YES = "POSTBACK_BUTTON_YES"
	PAYLOAD_NO  = "POSTBACK_BUTTON_NO"
)

type Photos struct {
	Status int     `json:"status"`
	ReqId  string  `json:"reqId"`
	Result []Photo `json:"result"`
}

type Photo struct {
	ReqId       string `json:"reqId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Urls        Urls   `json:"urls"`
	Type        string `json:"type"`
}

type Urls struct {
	Normal string `json:"normal"`
	HD     string `json:"hd"`
}

func init() {
	http.HandleFunc("/webhook", handleWebhook)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	msgBot := NewMessengerBot(r, ACCESS_TOKEN, VERIFY_TOKEN)
	msgBot.MessageReceived = func(e Event, mo MessageOpts, rm ReceivedMessage, hasErrorCh chan bool) {
		cxt := appengine.NewContext(r)
		defer func() {
			if err := recover(); err != nil {
				cxt.Errorf("Unknown error recover: %v", err)
				hasErrorCh <- true
			}
		}()
		user := NewUserFromId(mo.Sender.ID)
		msg := NewButtonTemplate("Welcome to use NasaPic, do you want to get last three days photos from NASA?")
		yesBtn := NewPostbackButton(BTN_PAYLOAD_YES, PAYLOAD_YES)
		NoBtn := NewPostbackButton(BTN_PAYLOAD_NO, PAYLOAD_NO)
		msg.AddButton(yesBtn, NoBtn)

		msgBot.Send(user, msg, NotificationTypeRegular)
		hasErrorCh <- false
	}

	msgBot.Postback = func(e Event, mo MessageOpts, pb Postback, hasErrorCh chan bool) {
		cxt := appengine.NewContext(r)
		defer func() {
			if err := recover(); err != nil {
				cxt.Errorf("Unknown error recover: %v", err)
				hasErrorCh <- true
			}
		}()
		user := NewUserFromId(mo.Sender.ID)
		switch pb.Payload {
		case PAYLOAD_NO:
			msgBot.Send(user, NewMessage("Sad! But still happy to see you here. :) "), NotificationTypeRegular)
			hasErrorCh <- false
		case PAYLOAD_YES:
			for i := API_INDEX_START; i <= API_INDEX_END; i++ {
				success, photos := getPhotos(cxt, msgBot.Client, i)
				if success {
					for _, res := range photos.Result {
						msgBot.Send(user, NewMessage(fmt.Sprintf("Photo of %s", res.Date)), NotificationTypeRegular)
						msgBot.Send(user, NewImageMessage(res.Urls.HD), NotificationTypeRegular)
					}
					hasErrorCh <- false
					return
				}
			}
			//Some error happened before, otherwise you can't arrive here.
			msgBot.Send(user, NewMessage(ERR), NotificationTypeRegular)
			hasErrorCh <- true
		}
	}
	msgBot.Handler(w, r)
}

func getPhotos(cxt appengine.Context, httpClient *http.Client, apiIndex int) (success bool, photos *Photos) {
	response, err := httpClient.Post(fmt.Sprintf("http://nasa-photo-dev%d.appspot.com/last_three_list", apiIndex), "application/json", bytes.NewBufferString(fmt.Sprintf(`{"reqId":"%s","timeZone":"CET"}`, NewV4().String())))
	if err != nil {
		success = false
		photos = nil
		cxt.Errorf(fmt.Sprintf("Error: %v", err))
		return
	}
	if response != nil {
		defer response.Body.Close()
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		success = false
		photos = nil
		cxt.Errorf(fmt.Sprintf("Error: %v", err))
		return
	}
	if response.StatusCode == http.StatusOK {
		success = true
		photos = &Photos{}
		json.Unmarshal(body, photos)
	} else {
		success = false
		photos = nil
		cxt.Errorf(fmt.Sprintf("Status: %v", response.StatusCode))
	}
	return
}
