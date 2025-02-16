package main

import (
	"context"
	"errors"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strava-intervals-description-sync/intervals"
	"strava-intervals-description-sync/strava"
	"strings"
	"syscall"
	"time"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	server := &http.Server{
		Addr: ":5001",
	}

	http.HandleFunc(strava.WebhookUrl, handleWebhookRequest)
	http.HandleFunc(strava.InitiateAuthenticationUrl, strava.HandleAuthentication)
	http.HandleFunc(strava.AuthenticationCallbackUrl, strava.HandleAuthenticationCallback)

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped serving new connections.")
	}()

	if err = strava.InitiateWebhookRegistration(); err != nil {
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
		strava.HandleWebhookRegistrationRequest(w, req)
	} else if req.Method == http.MethodPost {
		shouldProcess, stravaActivityId := strava.ShouldProcessWebhook(w, req)
		if shouldProcess {
			log.Println("Received webhook to process for activity id ", stravaActivityId)
			// start goroutine not to keep request open for too long
			go syncActivities(stravaActivityId)
		}
	}
}

const SummarySeparator string = "\n--------\nWorkout Summary\n--------\n"

func syncActivities(stravaActivityId int64) {
	stravaActivity, err := strava.GetActivity(stravaActivityId)
	if err != nil {
		log.Println("Error getting strava activity ", err)
		return
	}

	if strings.Contains(stravaActivity.Description, SummarySeparator) {
		log.Println("Activity already contains summary")
		return
	}

	from := stravaActivity.StartDateLocal.Add(0)
	to := stravaActivity.StartDateLocal.Add(0)

	intervalsActivity, err := intervals.FindActivity(stravaActivityId, &from, &to)
	if err != nil {
		log.Println("Error getting intervals activity ", err)
		return
	}
	intervalsWorkout, err := intervals.FindWorkoutForActivity(intervalsActivity)
	if err != nil {
		log.Println("Error getting intervals workout ", err)
		return
	}
	athleteSportSettings, err := intervals.GetAthleteSportSettings(intervals.SportTypeRun)
	if err != nil {
		log.Println("Error getting athleteSportSettings ", err)
		return
	}

	workoutSummary := intervalsWorkout.GenerateDescription(athleteSportSettings)

	updatableActivity := &strava.UpdatableActivity{
		Description: stravaActivity.Description + SummarySeparator + workoutSummary,
	}
	if err = strava.UpdateActivity(stravaActivityId, updatableActivity); err != nil {
		log.Println("Error updating strava activity ", err)
	}
}
