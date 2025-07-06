package intervals

import (
	"fmt"
	"log"
	"math"
	"time"
)

// GenerateDescription iterates Workout steps and generates text summary for it
func (w *Workout) GenerateDescription(sportSettings *AthleteSportSettings) string {
	var summary string

	for i, doc := range *w.WorkoutDoc.Steps {
		if i > 0 {
			summary += "\n"
		}
		summary += doc.generateSummaryLineOrBlock(sportSettings)
	}

	return summary
}

// generateSummaryLineOrBlock takes a single step and generates summary depending on what type it is:
// - Repetitions (e.g. repeat step X 3 times)
//   - Recursively calls this function for each child step (haven't tested or seen multiple level nested repeats)
//
// - HeartRate - hr base workout step
// - Pace - pace based workout step
func (w *WorkoutStep) generateSummaryLineOrBlock(sportSettings *AthleteSportSettings) string {
	var result string
	if w.Repetitions > 0 && len(*w.Steps) > 0 {
		result += fmt.Sprintf("%dX:\n", w.Repetitions)
		for i, doc := range *w.Steps {
			result += fmt.Sprintf("- %s", doc.generateSummaryLineOrBlock(sportSettings))
			if i != len(*w.Steps)-1 {
				result += "\n"
			}
		}
	} else if w.HeartRate != nil {
		result = w.generateHeartRateDescriptionLine(sportSettings)
	} else if w.Pace != nil {
		result = w.generatePaceDescriptionLine(sportSettings)
	} else {
		log.Println("Unexpected step type")
	}
	return result
}

func (w *WorkoutStep) generateHeartRateDescriptionLine(sportSettings *AthleteSportSettings) string {
	durationOrDistance := w.calculationDurationOrDistanceText()

	var zoneDetails string
	var zoneName string

	switch w.HeartRate.Units {
	// just the value of hr zone as integer
	case "hr_zone":
		zoneName = fmt.Sprintf("Z%d", int(w.HeartRate.Value))
		// % of max hr
	case "%hr":
		if w.HeartRate.Value > 0 {
			zoneName, zoneDetails = w.calculateSingleHrZoneAndDetails(sportSettings, sportSettings.MaximumHeartRate)
		} else {
			zoneName, zoneDetails = w.calculateRangeHrZoneAndDetails(sportSettings, sportSettings.MaximumHeartRate)
		}
	case "%lthr":
		if w.HeartRate.Value > 0 {
			zoneName, zoneDetails = w.calculateSingleHrZoneAndDetails(sportSettings, sportSettings.ThresholdHeartRate)
		} else {
			zoneName, zoneDetails = w.calculateRangeHrZoneAndDetails(sportSettings, sportSettings.ThresholdHeartRate)
		}
	default:
		return ""
	}

	return fmt.Sprintf("%s @ %s (%s)", durationOrDistance, zoneName, zoneDetails)
}

func calculateHeartRateZone(hrValue float32, sportSettings *AthleteSportSettings) int {
	for i, hrZoneUpperValue := range sportSettings.HeartRateZones {
		if hrValue <= float32(hrZoneUpperValue) {
			return i + 1 // 0 index == Z1
		}
	}

	log.Println("Could not find a value for the heart rate zone")
	return 1
}

func (w *WorkoutStep) calculateSingleHrZoneAndDetails(sportSettings *AthleteSportSettings, sourceHrValue int) (zone string, details string) {
	hrValue := w.HeartRate.Value / 100 * float32(sourceHrValue)
	hrZone := calculateHeartRateZone(hrValue, sportSettings)
	return fmt.Sprintf("Z%d", hrZone), fmt.Sprintf("%d", int(hrValue))
}

func (w *WorkoutStep) calculateRangeHrZoneAndDetails(sportSettings *AthleteSportSettings, sourceHrValue int) (zone string, details string) {
	hrStart := w.HeartRate.Start / 100 * float32(sourceHrValue)
	hrEnd := w.HeartRate.End / 100 * float32(sourceHrValue)

	hrZoneStart := calculateHeartRateZone(hrStart, sportSettings)
	hrZoneEnd := calculateHeartRateZone(hrEnd, sportSettings)
	var zoneName string
	if hrZoneStart == hrZoneEnd {
		zoneName = fmt.Sprintf("Z%d", hrZoneStart)
	} else {
		zoneName = fmt.Sprintf("Z%d-Z%d", hrZoneStart, hrZoneEnd)
	}
	return zoneName, fmt.Sprintf("%d-%d bpm", int(hrStart), int(hrEnd))
}

func (w *WorkoutStep) generatePaceDescriptionLine(sportSettings *AthleteSportSettings) string {
	durationOrDistance := w.calculationDurationOrDistanceText()

	var zoneDetails string
	var zoneName string

	switch w.Pace.Units {
	// just the value of hr zone as integer
	case "pace_zone":
		zoneName = fmt.Sprintf("Z%d", int(w.HeartRate.Value))
		// % of max hr
	case "%pace":
		if w.Pace.Value > 0 {
			paceValueMinPerKm := 1 / ((w.Pace.Value / 100 * sportSettings.ThresholdPace) / 1000 * 60)
			paceDuration := time.Duration(paceValueMinPerKm * float32(time.Minute))
			paceZone := calculatePaceZone(w.Pace.Value, sportSettings)
			zoneName = fmt.Sprintf("Z%d", paceZone)
			zoneDetails = fmt.Sprintf("%s min/km",
				time.Unix(0, 0).UTC().Add(paceDuration).Format("04:05"))
		} else {
			startPaceValueMinPerKm := 1 / ((w.Pace.Start / 100 * sportSettings.ThresholdPace) / 1000 * 60)
			startPaceValueDuration := time.Duration(startPaceValueMinPerKm * float32(time.Minute))
			endPaceValueMinPerKm := 1 / ((w.Pace.End / 100 * sportSettings.ThresholdPace) / 1000 * 60)
			endPaceValueDuration := time.Duration(endPaceValueMinPerKm * float32(time.Minute))

			startPaceZone := calculatePaceZone(w.Pace.Start, sportSettings)
			endPaceZone := calculatePaceZone(w.Pace.End, sportSettings)
			if startPaceZone == endPaceZone {
				zoneName = fmt.Sprintf("Z%d", startPaceZone)
			} else {
				zoneName = fmt.Sprintf("Z%d-Z%d", startPaceZone, endPaceZone)
			}
			zoneDetails = fmt.Sprintf("%s-%s min/km",
				time.Unix(0, 0).UTC().Add(startPaceValueDuration).Format("04:05"),
				time.Unix(0, 0).UTC().Add(endPaceValueDuration).Format("04:05"))
		}
	default:
		return ""
	}

	return fmt.Sprintf("%s @ Pace %s (%s)", durationOrDistance, zoneName, zoneDetails)
}

func calculatePaceZone(pacePercentage float32, sportSettings *AthleteSportSettings) int {
	for i, paceZoneUpperPercentage := range sportSettings.PaceZones {
		if pacePercentage <= paceZoneUpperPercentage {
			return i + 1 // 0 index == Z1
		}
	}

	log.Println("Could not find a value for the pace zone")
	return 1
}

func (w *WorkoutStep) calculationDurationOrDistanceText() string {
	distanceText := ""
	durationText := ""

	if w.Distance > 0 {
		if w.Distance < 1000 {
			distanceText = fmt.Sprintf("%dm", int(w.Distance))
		} else {
			distanceText = fmt.Sprintf("%.2gkm", float32(w.Distance)/1000)
		}
	}
	if w.Duration > 0 {
		if w.Duration < 60 {
			durationText = fmt.Sprintf("%ds", int(roundNum(w.Duration)))
		} else {
			duration := time.Duration(roundNum(w.Duration) * float32(time.Second))
			durationText = time.Unix(0, 0).UTC().Add(duration).Format("04:05min")
		}
	}

	if distanceText == "" {
		return durationText
	} else if durationText == "" {
		return distanceText
	} else {
		// If both duration and distance is set (intervals usually calculates missing one after activity and workout is matched)
		// it's not really possible to figure out which was the "original" one (at least I haven't found anything in
		// API data that would indicate it).
		// So I'm just checking if distance is a round number which in most cases should indicate that it was the originally
		// intended interval format
		if int(w.Distance)%100 == 0 {
			return distanceText
		}
		return fmt.Sprintf("%s / %s", durationText, distanceText)
	}
}

func roundNum(num float32) float32 {
	return float32(math.Round(float64(num)/10) * 10)
}
