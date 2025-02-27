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

func InitiateWebhookRegistration() error {
	sub, err := getSubscription()
	if err != nil {
		log.Println("Failed to fetch Strava webhook subscription")
		return err
	}

	if sub != nil {
		desiredCallbackUrl, _ := getWebhookCallbackUrl()
		if sub.CallbackUrl == desiredCallbackUrl {
			log.Println("Found correct existing Strava webhook subscription")
			return nil
		} else {
			log.Println("Found existing Strava webhook subscription with incorrect webhook url, recreating..")
			err = deleteSubscription(sub)
			if err != nil {
				log.Println("Failed to delete Strava webhook subscription")
				return err
			}
		}
	}

	if err = createSubscription(); err != nil {
		log.Println("Failed to create Strava webhook subscription")
		return err
	}

	return nil
}

func createSubscription() error {
	var buf bytes.Buffer
	log.Println("Creating webhook registration request")

	callbackUrl, err := getWebhookCallbackUrl()
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
		log.Printf("Failed to create webhook registration request: %s", err)
		return err
	}

	err = writer.Close()
	if err != nil {
		log.Printf("Failed to close form writer: %s", err)
		return err
	}

	resp, err := http.Post("https://www.strava.com/api/v3/push_subscriptions", writer.FormDataContentType(), &buf)
	if err != nil {
		log.Printf("Failed to send webhook registration request: %s", err)
		return err
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		log.Printf("Webhook registration request successful")
		return nil
	}

	log.Println("Unexpected webhook registration status code", resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	if err == nil {
		log.Println(string(b))
	}
	return errors.New("strava webhook registration failed")
}

func getSubscription() (*subscription, error) {
	resp, err := http.Get(fmt.Sprintf("https://www.strava.com/api/v3/push_subscriptions?client_id=%s&client_secret=%s",
		os.Getenv("STRAVA_CLIENT_ID"), os.Getenv("STRAVA_CLIENT_SECRET")))

	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var subs []subscription
		if err := json.Unmarshal(bodyBytes, &subs); err != nil {
			log.Printf("Failed to unmarshal response body: %s", err)
			return nil, err
		}

		if len(subs) > 0 {
			return &subs[0], nil
		}

		return nil, nil
	}

	return nil, nil
}

func deleteSubscription(sub *subscription) error {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("https://www.strava.com/api/v3/push_subscriptions/%v?client_id=%s&client_secret=%s",
			sub.Id, os.Getenv("STRAVA_CLIENT_ID"), os.Getenv("STRAVA_CLIENT_SECRET")), nil)
	if err != nil {
		log.Printf("Failed to create delete subscription request: %s", err)
		return err
	}
	_, err = client.Do(req)
	if err != nil {
		log.Printf("Failed to delete subscription request: %s", err)
	}

	return nil
}

func getWebhookCallbackUrl() (string, error) {
	return url.JoinPath(os.Getenv("STRAVA_CALLBACK_BASE_URL"), WebhookUrl)
}
