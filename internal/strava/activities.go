package strava

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strava-intervals-description-sync/internal/strava/persistence"
	"strava-intervals-description-sync/internal/util"
)

func GetActivity(id int64) (*Activity, error) {
	getActivityFunc := func() (*http.Response, error) {
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
	}

	shouldRetryFunc := func(resp *http.Response, err error) bool {
		return resp.StatusCode == http.StatusUnauthorized
	}

	beforeRetryFunc := func(resp *http.Response, err error) error {
		return RefreshToken()
	}

	resp, err := util.SendHttpRequestWithExpRetry(getActivityFunc, shouldRetryFunc, beforeRetryFunc, 1)

	if err != nil {
		log.Println("Failed to get activity", err)
		return nil, err
	}

	var activity *Activity
	if err = json.NewDecoder(resp.Body).Decode(&activity); err != nil {
		log.Println("Failed to decode activity ", id)
		return nil, err
	}

	// No clue what goes on here, but I have successfully received empty bodies before
	if activity.SportType == "" && activity.Name == "" {
		log.Println("Failed to get activity")
		return nil, errors.New("failed to get activity")
	}

	return activity, nil
}

func UpdateActivity(id int64, activity *UpdatableActivity) error {
	updatefunc := func() (*http.Response, error) {
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
	}

	shouldRetryFunc := func(resp *http.Response, err error) bool {
		return resp.StatusCode == http.StatusUnauthorized
	}

	beforeRetryFunc := func(resp *http.Response, err error) error {
		return RefreshToken()
	}

	resp, err := util.SendHttpRequestWithExpRetry(updatefunc, shouldRetryFunc, beforeRetryFunc, 1)

	if err != nil {
		log.Println("Failed to update activity")
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Println("Failed to update activity", resp.StatusCode)
	}

	return nil
}
