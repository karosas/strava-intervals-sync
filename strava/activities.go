package strava

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strava-intervals-description-sync/strava/persistence"
)

func sendRequestWithRetry(sendRequest func() (*http.Response, error)) (*http.Response, error) {
	resp, err := sendRequest()

	if err != nil {
		// refresh access token and try again
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			log.Println("Strava token expired, refreshing")
			tempErr := RefreshToken()
			if tempErr != nil {
				log.Println("Failed to refresh Strava token")
				return nil, tempErr
			}

			resp, err = sendRequest()
			if err != nil {
				log.Println("Retry with refreshed token failed")
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return resp, err
}

func GetActivity(id int64) (*Activity, error) {
	resp, err := sendRequestWithRetry(func() (*http.Response, error) {
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://www.strava.com/api/v3/activities/%v", id), nil)
		if err != nil {
			return nil, err
		}
		token, err := persistence.ReadAccessToken()
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)

		return resp, err
	})

	if err != nil {
		log.Println("Failed to get activity")
		return nil, err
	}

	var activity *Activity
	if err = json.NewDecoder(resp.Body).Decode(&activity); err != nil {
		log.Println("Failed to decode activity ", id)
		return nil, err
	}

	return activity, nil
}

func UpdateActivity(id int64, activity *UpdatableActivity) error {
	resp, err := sendRequestWithRetry(func() (*http.Response, error) {
		client := &http.Client{}
		jsonBody, err := json.Marshal(activity)
		if err != nil {
			return nil, err
		}
		log.Println(string(jsonBody))
		req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("https://www.strava.com/api/v3/activities/%v", id), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		if err != nil {
			return nil, err
		}
		token, err := persistence.ReadAccessToken()
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return client.Do(req)
	})

	if err != nil {
		log.Println("Failed to update activity")
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Println("Failed to update activity", resp.StatusCode)
	}

	return nil
}
