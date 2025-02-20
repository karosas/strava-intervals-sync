package strava

import "time"

const (
	InitiateAuthenticationUrl string = "/strava/auth"
	AuthenticationCallbackUrl string = "/strava/auth/callback"
	WebhookUrl                string = "/strava/webhook"
)

const (
	WebhookObjectTypeActivity = "activity"
	WebhookAspectTypeCreate   = "create"
	WebhookAspectTypeUpdate   = "update"
)

type Webhook struct {
	AspectType string `json:"aspect_type"`
	ObjectType string `json:"object_type"`
	ObjectId   int64  `json:"object_id"`
	OwnerId    int64  `json:"owner_id"`
}

type subscription struct {
	Id          int32  `json:"id"`
	CallbackUrl string `json:"callback_url"`
}

type Activity struct {
	Description    string    `json:"description"`
	Name           string    `json:"name"`
	Commute        bool      `json:"commute"`
	Trainer        bool      `json:"trainer"`
	HideFromHome   bool      `json:"hide_from_home"`
	SportType      string    `json:"sport_type"`
	GearId         string    `json:"gear_id"`
	StartDate      time.Time `json:"start_date"`
	StartDateLocal time.Time `json:"start_date_local"`
}

type UpdatableActivity struct {
	Description string `json:"description"`
}

type tokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	Athlete      tokenAthlete `json:"athlete"`
}

type tokenAthlete struct {
	Id int32 `json:"id"`
}
