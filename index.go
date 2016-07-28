package index

import (
	"appengine"
	"net/http"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

const (
	API_INDEX_START = 2
	API_INDEX_END   = 6
	ERR             = "Sick, sorry I have some internal problems. :(. Try again or later."
	DONE            = ":) Done!"

	//Buttons
	BTN_PAYLOAD_YES      = "Yes, show last 3 days photos."
	BTN_PAYLOAD_NO       = "No, thanks."
	BTN_START_THIS_MONTH = "This month photos please."
	BTN_STOP_THIS_MONTH  = "No more photos, thanks."

	//Payloads
	PAYLOAD_YES              = "POSTBACK_BUTTON_YES"
	PAYLOAD_NO               = "POSTBACK_BUTTON_NO"
	PAYLOAD_START_THIS_MONTH = "POSTBACK_BUTTON_START_THIS_MONTH"
	PAYLOAD_STOP_THIS_MONTH  = "POSTBACK_BUTTON_STOP_THIS_MONTH"
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
		thisMonthBtn := NewPostbackButton(BTN_START_THIS_MONTH, PAYLOAD_START_THIS_MONTH)
		msg.AddButton(yesBtn, thisMonthBtn, NoBtn)

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
				success, photos := get3DaysPhotos(cxt, msgBot.Client, i)
				if success {
					for _, res := range photos.Result {
						msgBot.Send(user, NewMessage(fmt.Sprintf("Photo of %s", res.Date)), NotificationTypeRegular)
						msgBot.Send(user, NewImageMessage(res.Urls.HD), NotificationTypeRegular)
					}
					msgBot.Send(user, NewMessage(DONE), NotificationTypeRegular)
					//Asking for getting photos of this month.
					now := time.Now()
					year := now.Year()
					month := now.Month()
					msg := NewButtonTemplate(fmt.Sprintf("Get whole photo list this month (%s/%d)?", month, year))
					yesBtn := NewPostbackButton(BTN_START_THIS_MONTH, PAYLOAD_START_THIS_MONTH)
					NoBtn := NewPostbackButton(BTN_STOP_THIS_MONTH, PAYLOAD_STOP_THIS_MONTH)
					msg.AddButton(yesBtn, NoBtn)

					msgBot.Send(user, msg, NotificationTypeRegular)
					hasErrorCh <- false
					return
				}
			}
			//Some errors happened before, otherwise you can't arrive here.
			msgBot.Send(user, NewMessage(ERR), NotificationTypeRegular)
			hasErrorCh <- true
		case PAYLOAD_STOP_THIS_MONTH:
			msgBot.Send(user, NewMessage("Sad! But still happy to see you here. :) "), NotificationTypeRegular)
			hasErrorCh <- true
		case PAYLOAD_START_THIS_MONTH:
			for i := API_INDEX_START; i <= API_INDEX_END; i++ {
				success, photos := getMonthPhotos(cxt, msgBot.Client, i)
				if success {
					var pMsg *GenericTemplate
					i := 0
					for _, res := range photos.Result {
						if i%5 == 0 { //Every 5 box to show.
							if pMsg != nil {
								cxt.Infof(fmt.Sprintf("Send:%v", pMsg))
								msgBot.Send(user, *pMsg, NotificationTypeRegular)
							}
							msg := NewGenericTemplate()
							pMsg = &msg
						}
						element := Element{
							Title:    res.Title,
							Url:      res.Urls.HD,
							ImageUrl: res.Urls.HD,
							Subtitle: res.Date,
						}
						pMsg.AddElement(element)
						cxt.Infof(fmt.Sprintf("Add:%v", element))
						i++
					}
					msgBot.Send(user, NewMessage(DONE), NotificationTypeRegular)
					hasErrorCh <- false
					return
				}
			}
			//Some errors happened before, otherwise you can't arrive here.
			msgBot.Send(user, NewMessage(ERR), NotificationTypeRegular)
			hasErrorCh <- true
		}
	}
	msgBot.Handler(w, r)
}

func get3DaysPhotos(cxt appengine.Context, httpClient *http.Client, apiIndex int) (success bool, photos *Photos) {
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

func getMonthPhotos(cxt appengine.Context, httpClient *http.Client, apiIndex int) (success bool, photos *Photos) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	response, err := httpClient.Post(fmt.Sprintf("http://nasa-photo-dev%d.appspot.com/month_list", apiIndex), "application/json", bytes.NewBufferString(fmt.Sprintf(`{"reqId":"%s","year":%d, "month" : %d, "timeZone":"CET"}`, NewV4().String(), year, month)))
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
