package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"log"
	"net/http"
	"os"
	"time"
)

var config = Configuration{
	SlackName:       os.Getenv("SLACK_NAME"),
	SlackIconUrl:    os.Getenv("SLACK_ICON_URL"),
	SlackWebHookUrl: os.Getenv("SLACK_WEBHOOK_URL"),
	SlackChannel:    os.Getenv("SLACK_CHANNEL"),
	Host:            os.Getenv("HOST"),
	VoicemailAudio:  os.Getenv("VOICEMAIL_AUDIO"),
	ElksUserName:    os.Getenv("ELKS_USERNAME"),
	ElksPassword:    os.Getenv("ELKS_PASSWORD"),
	AWSAccessKey:    os.Getenv("AWS_ACCESS_KEY"),
	AWSSecretKey:    os.Getenv("AWS_SECRET_KEY"),
	S3BucketName:    os.Getenv("S3_BUCKET_NAME"),
}

func main() {
	AWSAuth := aws.Auth{
		AccessKey: config.AWSAccessKey,
		SecretKey: config.AWSSecretKey,
	}
	region := aws.EUWest

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
		log.Print("Incoming message...")
		err := r.ParseForm()
		check(err)
		from := r.FormValue("from")
		wav := r.FormValue("wav")
		if len(wav) == 0 || len(from) == 0 {
			log.Print("Bad request")
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		log.Print("Incoming message from " + from)
		log.Print("Retrieving audio file " + wav + "...")
		req, err := http.NewRequest("GET", wav, nil)
		req.SetBasicAuth(config.ElksUserName, config.ElksPassword)
		client := &http.Client{}
		resp, err := client.Do(req)
		check(err)
		defer resp.Body.Close()

		log.Print("Downloading...")
		filebytes := make([]byte, resp.ContentLength)
		buffer := bufio.NewReader(resp.Body)
		_, err = buffer.Read(filebytes)
		check(err)

		log.Print("Connecting to AWS S3...")
		connection := s3.New(AWSAuth, region)
		bucket := connection.Bucket(config.S3BucketName)

		path := "voicemail/" + time.Now().Format("20060102150405") + ".wav"
		log.Print("Writing " + path + " to " + config.S3BucketName + "...")
		err = bucket.Put(path, filebytes, "audio/wav", s3.ACL("public-read"))
		check(err)

		log.Print("Posting message to Slack, channel " + config.SlackChannel + "...")
		payload := SlackPayload{
			UserName: config.SlackName,
			IconUrl:  config.SlackIconUrl,
			Text:     "New voice message from " + from + " <" + bucket.URL(path) + ">!",
			Channel:  config.SlackChannel,
		}

		json_payload, _ := json.Marshal(payload)
		slack_req, err := http.NewRequest("POST", config.SlackWebHookUrl, bytes.NewBuffer([]byte(json_payload)))
		req.Header.Set("Content-Type", "application/json")

		slack_client := &http.Client{}
		slack_resp, err := slack_client.Do(slack_req)
		check(err)
		defer slack_resp.Body.Close()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(nil)
	})

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	http.ListenAndServe(":"+port, nil)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type Configuration struct {
	SlackName       string
	SlackIconUrl    string
	SlackWebHookUrl string
	SlackChannel    string
	Host            string
	VoicemailAudio  string
	ElksUserName    string
	ElksPassword    string
	AWSAccessKey    string
	AWSSecretKey    string
	S3BucketName    string
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
