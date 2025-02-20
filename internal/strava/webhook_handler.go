package strava

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
)

func HandleWebhookRegistrationRequest(w http.ResponseWriter, req *http.Request) {
	log.Println("Received webhook registration request")

	mode := req.URL.Query().Get("hub.mode")
	token := req.URL.Query().Get("hub.verify_token")
	challenge := req.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == os.Getenv("STRAVA_VERIFY_TOKEN") {
		if _, err := w.Write([]byte("{\"hub.challenge\":\"" + challenge + "\"}")); err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		log.Println("Webhook subscribed successful")

	} else {
		log.Println("Webhook subscription missing correct query params")
		w.WriteHeader(http.StatusForbidden)
	}
}

func ShouldProcessWebhook(w http.ResponseWriter, req *http.Request) (shouldProcess bool, activityId int64) {
	var webhook Webhook
	if err := json.NewDecoder(req.Body).Decode(&webhook); err != nil {
		log.Println("Failed to decode webhook", err)
		w.WriteHeader(http.StatusBadRequest)
		return false, 0
	}

	if strconv.Itoa(int(webhook.OwnerId)) == os.Getenv("STRAVA_CLIENT_ATHLETE_ID") &&
		(webhook.AspectType == WebhookAspectTypeCreate || webhook.AspectType == WebhookAspectTypeUpdate) &&
		webhook.ObjectType == WebhookObjectTypeActivity {
		return true, webhook.ObjectId
	}

	return false, 0
}
