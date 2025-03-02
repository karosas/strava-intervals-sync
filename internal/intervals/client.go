package intervals

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strava-intervals-description-sync/internal/util"
	"strconv"
	"time"
)

func FindActivity(stravaActivityId int64, from *time.Time, to *time.Time) (*Activity, error) {
	findActivityFunc := func() (*http.Response, error) {
		client := &http.Client{}

		fromFmt := from.Format("2006-01-02T15:04:05")
		toFmt := to.Format("2006-01-02T15:04:05")
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://intervals.icu/api/v1/athlete/%s/activities?oldest=%s&newest=%s",
			os.Getenv("INTERVALS_ATHLETE_ID"), fromFmt, toFmt), nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth("API_KEY", os.Getenv("INTERVALS_API_KEY"))
		return client.Do(req)
	}

	shouldRetryFunc := func(resp *http.Response, err error) bool {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var activities []*Activity
			if err = json.NewDecoder(resp.Body).Decode(&activities); err != nil {
				return false
			}

			for _, activity := range activities {
				if activity.StravaId == strconv.Itoa(int(stravaActivityId)) {
					return false
				}
			}
			b, _ := io.ReadAll(resp.Body)
			log.Println(string(b))
			return true
		}

		log.Println("Unexpected status code:", resp.StatusCode)

		return true
	}

	log.Println("Attempting to find intervals.icu activity")
	resp, err := util.SendHttpRequestWithExpRetry(findActivityFunc, shouldRetryFunc,
		func(r *http.Response, err error) error { return nil }, 10)

	if err != nil {
		return nil, err
	}

	var activities []*Activity
	if err = json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		return nil, err
	}

	_ = json.NewEncoder(os.Stdout).Encode(activities)

	for _, activity := range activities {
		if activity.StravaId == strconv.Itoa(int(stravaActivityId)) {
			return activity, nil
		}
	}

	return nil, errors.New("couldn't find matching activity")
}

// GetAthleteSportSettings fetches 'setting' like hr/pace zones from intervals.icu
func GetAthleteSportSettings(sportType SportType) (*AthleteSportSettings, error) {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://intervals.icu/api/v1/athlete/%s/sport-settings/%s",
		os.Getenv("INTERVALS_ATHLETE_ID"), sportType), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("API_KEY", os.Getenv("INTERVALS_API_KEY"))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Println("Received unexpected status code fetching intervals activity", resp.StatusCode)

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Println(string(bodyBytes))
		return nil, errors.New("unexpected status code fetching intervals activity")
	}

	var athleteSettings *AthleteSportSettings
	if err = json.NewDecoder(resp.Body).Decode(&athleteSettings); err != nil {
		return nil, err
	}

	return athleteSettings, nil
}

func FindWorkoutForActivity(intervalsActivity *Activity) (*Workout, error) {
	activityYear, activityMonth, activityDay := intervalsActivity.StartDate.Date()
	workoutFrom := time.Date(activityYear, activityMonth, activityDay, 0, 0, 0, 0, time.UTC)
	workoutTo := time.Date(activityYear, activityMonth, activityDay+1, 0, 0, 0, 0, time.UTC)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://intervals.icu/api/v1/athlete/%s/eventsjson?oldest=%s&newest=%s",
		os.Getenv("INTERVALS_ATHLETE_ID"), workoutFrom.Format("2006-01-02T15:04:05"), workoutTo.Format("2006-01-02T15:04:05")), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("API_KEY", os.Getenv("INTERVALS_API_KEY"))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Println("Received unexpected status code fetching intervals workout", resp.StatusCode)

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Println(string(bodyBytes))
		return nil, errors.New("unexpected status code fetching intervals workout")
	}

	var workouts []*Workout
	if err = json.NewDecoder(resp.Body).Decode(&workouts); err != nil {
		return nil, err
	}

	_ = json.NewEncoder(os.Stdout).Encode(workouts)

	for _, workout := range workouts {
		if workout.Id == intervalsActivity.PairedEventId {
			return workout, nil
		}
	}

	// If workout has not auto paired with activity in intervals, then let's try finding one whose
	// duration or distance is within 5% of workout's duration or distance
	for _, workout := range workouts {
		if math.Abs(float64(workout.WorkoutDoc.Distance)-float64(intervalsActivity.Distance)) < float64(workout.WorkoutDoc.Distance)*0.05 ||
			math.Abs(float64(workout.WorkoutDoc.Duration)-float64(intervalsActivity.MovingTime)) < float64(workout.WorkoutDoc.Duration)*0.05 {
			return workout, nil
		}
	}

	return nil, errors.New("couldn't find workout for activity")
}
