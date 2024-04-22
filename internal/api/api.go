package api

import (
	"encoding/json"
	"glooko/internal/domain"
	"glooko/internal/ports"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type API struct {
	log          *zap.SugaredLogger
	userRepo     ports.UserRepository
	deviceRepo   ports.DeviceRepository
	readingsRepo ports.ReadingRepository
	validate     *validator.Validate
}

func NewAPI(log *zap.SugaredLogger, userRepo ports.UserRepository, deviceRepo ports.DeviceRepository, readingsRepo ports.ReadingRepository) *API {
	return &API{
		log:          log,
		userRepo:     userRepo,
		deviceRepo:   deviceRepo,
		readingsRepo: readingsRepo,
		validate:     validator.New(),
	}
}

func (api *API) Routes() *chi.Mux {
	r := chi.NewRouter()

	// home endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Glooko demo API"))
	})

	// User routes
	r.Route("/users", func(r chi.Router) {
		r.Get("/{id}/overview", api.GetUserOverview)
	})

	// Device routes
	r.Route("/devices", func(r chi.Router) {
		r.Get("/{id}", api.GetDevice)
	})

	return r
}

// User handlers
type UserOverviewParams struct {
	ID    string `validate:"required"`
	Start string `validate:"omitempty,datetime=2006-01-02"`
	End   string `validate:"omitempty,datetime=2006-01-02"`
}

// Measure represents a single glucose measurment.
type Measure struct {
	Time  time.Time `json:"time"`
	Value int       `json:"value"`
}

// DailyMetrics holds aggregated metrics for a day.
type DailyMetrics struct {
	MinValue int     `json:"minValue"`
	MaxValue int     `json:"maxValue"`
	AvgValue float64 `json:"avgValue"`
}

// DailyReadings contains all readings for a specific day along with calculated metrics.
type DailyReadings struct {
	UserID  string       `json:"userId"`
	Day     string       `json:"day"`
	Measure []Measure    `json:"readings"`
	Metrics DailyMetrics `json:"metrics"`
}

// DailyReadingsResponse holds the response data for the user overview.
type UserOverviewResponse struct {
	Overview []DailyReadings `json:"overview"`
}

func (api *API) GetUserOverview(w http.ResponseWriter, r *http.Request) {
	log := api.log.With("method", "GetUserOverview")

	params := UserOverviewParams{
		ID:    chi.URLParam(r, "id"),
		Start: r.URL.Query().Get("start"),
		End:   r.URL.Query().Get("end"),
	}

	err := api.validate.Struct(params)
	if err != nil {
		log.Errorf("validation error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	start, end, err := parseDates(params.Start, params.End)
	if err != nil {
		log.Errorf("invalid date range: %v", err)
		http.Error(w, "Invalid date range: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	readings, err := api.readingsRepo.FetchReadings(ctx, params.ID, start, end)
	if err != nil {
		log.Errorf("failed to fetch readings: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	aggregatedData := aggregateReadings(readings)
	respondWithJSON(w, UserOverviewResponse{
		Overview: aggregatedData,
	})
}

// Device handlers
func (api *API) GetDevice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	device, err := api.deviceRepo.FindByID(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(device)
}

func respondWithJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// parseDates parses start and end date strings and ensures the date range includes
// the full day from 00:00:00 of the start day to 23:59:59 of the end day.
func parseDates(startStr, endStr string) (start, end time.Time, err error) {
	if startStr != "" {
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			return
		}
		start = start.Truncate(24 * time.Hour) // Ensure time is at 00:00:00
	} else {
		// Default to the last 14 days, starting from 00:00:00
		start = time.Now().Truncate(24*time.Hour).AddDate(0, 0, -14)
	}

	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return
		}
		// Set to end of the day by adding one full day minus one nanosecond
		end = end.Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)
	} else {
		// Default to the current date, ending at 23:59:59
		end = time.Now().Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)
	}

	return
}

func aggregateReadings(readings []domain.Reading) []DailyReadings {
	dailyReadingsMap := make(map[string]DailyReadings)

	for _, reading := range readings {
		dayKey := reading.Day.Format("2006-01-02") // Using a string date as the key
		daily, exists := dailyReadingsMap[dayKey]
		if !exists {
			daily = DailyReadings{
				UserID:  reading.UserID.Hex(),
				Day:     dayKey,
				Measure: []Measure{},
				Metrics: DailyMetrics{},
			}
		}

		for _, entry := range reading.Readings {
			// Only append if the entry is in the date range
			if entry.Time.Format("2006-01-02") != dayKey {
				continue
			}

			daily.Measure = append(daily.Measure, Measure{
				Time:  entry.Time,
				Value: entry.Value,
			})
		}

		// Update the map after appending
		dailyReadingsMap[dayKey] = daily
	}

	// Recalculate metrics after all readings are processed for each day
	for dayKey, daily := range dailyReadingsMap {
		allValues := make([]int, len(daily.Measure))
		for i, measure := range daily.Measure {
			allValues[i] = measure.Value
		}

		if len(allValues) > 0 {
			daily.Metrics.MinValue, daily.Metrics.MaxValue = minMax(allValues)
			daily.Metrics.AvgValue = average(allValues)
			dailyReadingsMap[dayKey] = daily
		}
	}

	// Convert map to slice, sort and return
	var combinedReadings []DailyReadings
	for _, daily := range dailyReadingsMap {
		combinedReadings = append(combinedReadings, daily)
	}

	sort.Slice(combinedReadings, func(i, j int) bool {
		return combinedReadings[i].Day < combinedReadings[j].Day
	})

	return combinedReadings
}

func minMax(values []int) (min, max int) {
	min, max = values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

func average(values []int) float64 {
	total := 0
	for _, v := range values {
		total += v
	}
	return float64(total) / float64(len(values))
}
