package strava

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
)

func HandleWebhookRequest(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		handleWebhookRegistrationRequest(w, req)
	} else if req.Method == http.MethodPost {
		handleWebhookRequest(w, req)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleWebhookRegistrationRequest(w http.ResponseWriter, req *http.Request) {
	log.Println("Received webhook registration request")

	mode := req.URL.Query().Get("hub.mode")
	token := req.URL.Query().Get("hub.verify_token")
	challenge := req.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == os.Getenv("STRAVA_VERIFY_TOKEN") {
		log.Println("Webhook subscribed successful")
		fmt.Println(w, "{\"hub.challenge\":\""+challenge+"\"}")
	} else {
		log.Println("Webhook subscription missing correct query params")
		w.WriteHeader(http.StatusForbidden)
	}
}

func handleWebhookRequest(w http.ResponseWriter, req *http.Request) {
	var webhook Webhook
	if err := json.NewDecoder(req.Body).Decode(&webhook); err != nil {
		log.Println("Failed to decode webhook", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("Received webhook %s: %s", webhook.AspectType, webhook.ObjectType)
}

func InitiateWebhookRegistration() error {
	var buf bytes.Buffer
	var err error

	log.Println("Creating webhook registration request")

	callbackUrl, err := url.JoinPath(os.Getenv("STRAVA_CALLBACK_BASE_URL"), "/register-strava-webhook")
	writer := multipart.NewWriter(&buf)
	write := func(field, value string) {
		if err != nil {
			return
		}
		err = writer.WriteField(field, value)
	}

	write("client_id", os.Getenv("STRAVA_CLIENT_ID"))
	write("client_secret", os.Getenv("STRAVA_CLIENT_SECRET"))
	write("verify_token", os.Getenv("STRAVA_VERIFY_TOKEN"))
	write("callback_url", callbackUrl)

	if err != nil {
		log.Println("Failed to create webhook registration request")
		return err
	}

	resp, err := http.Post("https://www.strava.com/api/v3/push_subscriptions", writer.FormDataContentType(), &buf)
	if err != nil {
		log.Println("Failed to send webhook registration request")
		return err
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		log.Println(bodyString)
		log.Println("Unexpected webhook registration status code", resp.StatusCode)
		return errors.New("strava webhook registration failed")
	}

	log.Println("Webhook registration request successful")
	return nil
}
