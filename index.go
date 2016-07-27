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

type LastThreeRequest struct {
	ReqId    string `json:"reqId"`
	TimeZone string `json:"timeZone"`
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
		case PAYLOAD_YES:
			//Get last three day's photos from NASA
			reqBody := fmt.Sprintf(`{"reqId":"%s","timeZone":"CET"}`, NewV4().String())
			response, err := msgBot.Client.Post("http://nasa-photo-dev4.appspot.com/last_three_list", "application/json", bytes.NewBufferString(reqBody))

			if err != nil {
				msgBot.Send(user, NewMessage("Sick, sorry I have some internal problems. :("), NotificationTypeRegular)
				cxt.Errorf(fmt.Sprintf("%v", err))
				hasErrorCh <- true
				return
			}

			if response != nil {
				defer response.Body.Close()
			}

			body, err := ioutil.ReadAll(response.Body)

			if err != nil {
				msgBot.Send(user, NewMessage("Sick, sorry I have some internal problems. :("), NotificationTypeRegular)
				cxt.Errorf(fmt.Sprintf("%v", err))
				hasErrorCh <- true
				return
			}
			if response.StatusCode == http.StatusOK {
				photos := Photos{}
				json.Unmarshal(body, &photos)

				for _, res := range photos.Result {
					msgBot.Send(user, NewMessage(fmt.Sprintf("Photo of %s", res.Date)), NotificationTypeRegular)
					msgBot.Send(user, NewImageMessage(res.Urls.HD), NotificationTypeRegular)
				}
			} else {
				cxt.Errorf(fmt.Sprintf("Status: %v", response.StatusCode))
			}
		}
		hasErrorCh <- false
	}

	msgBot.Handler(w, r)
}
