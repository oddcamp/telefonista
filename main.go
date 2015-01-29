package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

var config = Configuration{
	SlackName:       os.Getenv("SLACK_NAME"),
	SlackIconUrl:    os.Getenv("SLACK_ICON_URL"),
	SlackWebHookUrl: os.Getenv("SLACK_WEBHOOK_URL"),
	SlackChannel:    os.Getenv("SLACK_CHANNEL"),
	Host:            os.Getenv("HOST"),
	VoicemailAudio:  os.Getenv("VOICEMAIL_AUDIO"),
}

func main() {
	http.HandleFunc("/incoming_call", func(w http.ResponseWriter, r *http.Request) {

		i := IncomingResponse{
			Play: config.VoicemailAudio,
			Next: struct {
				Record string `json:"record"`
			}{
				Record: config.Host + "/voicemail",
			},
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(i)
	})

	http.HandleFunc("/voicemail", func(w http.ResponseWriter, r *http.Request) {
		var err error
		err = r.ParseForm()
		if err != nil {
		}
		from := r.FormValue("from")
		wav := r.FormValue("wav")

		payload := SlackPayload{
			UserName: config.SlackName,
			IconUrl:  config.SlackIconUrl,
			Text:     "New voice message from " + from + " <" + wav + ">!",
			Channel:  config.SlackChannel,
		}

		json_payload, _ := json.Marshal(payload)
		req, err := http.NewRequest("POST", config.SlackWebHookUrl, bytes.NewBuffer([]byte(json_payload)))
		req.Header.Set("X-Custom-Header", "myvalue")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(nil)
	})

	http.ListenAndServe(":8080", nil)
}

type Configuration struct {
	SlackName       string
	SlackIconUrl    string
	SlackWebHookUrl string
	SlackChannel    string
	Host            string
	VoicemailAudio  string
}

type SlackPayload struct {
	UserName string `json:"username"`
	IconUrl  string `json:"icon_url"`
	Text     string `json:"text"`
	Channel  string `json:"channel"`
}

type IncomingResponse struct {
	Play string `json:"play"`
	Next struct {
		Record string `json:"record"`
	} `json:"next"`
}
