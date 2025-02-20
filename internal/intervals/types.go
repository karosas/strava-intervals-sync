package intervals

import "time"

type SportType string

const SportTypeRun SportType = "Run"

type AthleteSportSettings struct {
	MaximumHeartRate   int      `json:"max_hr"`
	ThresholdHeartRate int      `json:"lthr"`
	HeartRateZones     []int    `json:"hr_zones"`
	HeartRateZoneNames []string `json:"hr_zone_names"`
	// ThresholdPace in m/s
	ThresholdPace float32 `json:"threshold_pace"`
	// PaceZones in form of percentage of `ThresholdPace`, e.g. `77.5` would be 77.5%
	PaceZones     []float32 `json:"pace_zones"`
	PaceZoneNames []string  `json:"pace_zone_names"`
}

type Activity struct {
	StravaId string `json:"strava_id"`
	// PairedEventId as far as I'm aware, refers to Workout.Id. It might however to also refer to maybe planned races in calendar?
	PairedEventId int       `json:"paired_event_id"`
	StartDate     time.Time `json:"start_date"`
	Distance      float32   `json:"distance"`
	MovingTime    float32   `json:"moving_time"`
}

type Workout struct {
	Id         int         `json:"id"`
	Name       string      `json:"name"`
	WorkoutDoc *WorkoutDoc `json:"workout_doc"`
}

type WorkoutDoc struct {
	Steps    *[]WorkoutStep `json:"steps"`
	Distance float32        `json:"distance"`
	Duration float32        `json:"duration"`
}

// WorkoutStep is dynamic, there are 2 dynamic parts:
// - Object representing unit (or range) for that step. It will have different key based on the unit (e.g. `hr` or `pace`)
//   - Check WorkoutStepUnit for details about its properties
//
// - Text can be nil
// - Distance / Duration can be nil (at least one will be not nil), based whether the step is based on distance or time
//   - If step is based on distance, but WorkoutStepUnit is `pace` then there'll be `Duration` calculated as well
type WorkoutStep struct {
	Distance    float32          `json:"distance"`
	Duration    float32          `json:"duration"`
	Text        string           `json:"text"`
	HeartRate   *WorkoutStepUnit `json:"hr"`
	Pace        *WorkoutStepUnit `json:"pace"`
	Steps       *[]WorkoutStep   `json:"steps"`
	Repetitions int              `json:"reps"`
}

type WorkoutStepUnit struct {
	Start float32 `json:"start"`
	End   float32 `json:"end"`
	// Units values I have observed so far are `%lthr`, `%pace`
	Units string  `json:"units"`
	Value float32 `json:"value"`
}
