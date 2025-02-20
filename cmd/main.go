package main

import (
	"context"
	"errors"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"os/signal"
	intervals2 "strava-intervals-description-sync/internal/intervals"
	strava2 "strava-intervals-description-sync/internal/strava"
	"strings"
	"syscall"
	"time"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	server := &http.Server{
		Addr: ":5001",
	}

	http.HandleFunc(strava2.WebhookUrl, handleWebhookRequest)
	http.HandleFunc(strava2.InitiateAuthenticationUrl, strava2.HandleAuthentication)
	http.HandleFunc(strava2.AuthenticationCallbackUrl, strava2.HandleAuthenticationCallback)

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped serving new connections.")
	}()

	if err = strava2.InitiateWebhookRegistration(); err != nil {
		if err = server.Shutdown(context.Background()); err != nil {
			log.Fatalf("HTTP shutdown error: %v", err)
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}

	log.Println("Stopped serving new connections.")
}

func handleWebhookRequest(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		strava2.HandleWebhookRegistrationRequest(w, req)
	} else if req.Method == http.MethodPost {
		shouldProcess, stravaActivityId := strava2.ShouldProcessWebhook(w, req)
		if shouldProcess {
			log.Println("Received webhook to process for activity id ", stravaActivityId)
			// start goroutine not to keep request open for too long
			go syncActivities(stravaActivityId)
		}
	}
}

const SummarySeparator string = "---Workout Summary---"

func syncActivities(stravaActivityId int64) {
	stravaActivity, err := strava2.GetActivity(stravaActivityId)
	if err != nil {
		log.Println("Error getting strava activity ", err)
		return
	}

	if strings.Contains(stravaActivity.Description, SummarySeparator) {
		log.Println("Activity already contains summary")
		return
	}

	from := stravaActivity.StartDateLocal.Add(-1 * time.Hour)
	to := stravaActivity.StartDateLocal.Add(time.Hour)

	intervalsActivity, err := intervals2.FindActivity(stravaActivityId, &from, &to)
	if err != nil {
		log.Println("Error getting intervals activity ", err)
		return
	}
	intervalsWorkout, err := intervals2.FindWorkoutForActivity(intervalsActivity)
	if err != nil {
		log.Println("Error getting intervals workout ", err)
		return
	}
	athleteSportSettings, err := intervals2.GetAthleteSportSettings(intervals2.SportTypeRun)
	if err != nil {
		log.Println("Error getting athleteSportSettings ", err)
		return
	}

	workoutSummary := intervalsWorkout.GenerateDescription(athleteSportSettings)

	updatableActivity := &strava2.UpdatableActivity{}
	if stravaActivity.Description == "" {
		updatableActivity.Description = SummarySeparator + "\n" + workoutSummary
	} else {
		updatableActivity.Description = stravaActivity.Description + "\n" + SummarySeparator + "\n" + workoutSummary
	}

	if err = strava2.UpdateActivity(stravaActivityId, updatableActivity); err != nil {
		log.Println("Error updating strava activity ", err)
	}
}
