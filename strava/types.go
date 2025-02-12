package strava

type Webhook struct {
	AspectType string `json:"aspect_type"`
	ObjectType string `json:"object_type"`
	OwnerId    string `json:"owner_id"`
}
