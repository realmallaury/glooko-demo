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

	r.Use(api.LoggingMiddleware)

	// home endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		respondWithJSON(w, "Glooko API")
	})

	// User routes
	r.Route("/users", func(r chi.Router) {
		r.Get("/{id}/overview", api.GetUserOverview)
		r.Get("/{id}/devices-overview", api.GetDevicesOverview)
	})

	return r
}

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

// DeviceCount represents the count of readings for a specific device on a given day.
type DeviceCount struct {
	DeviceID string `json:"deviceId"`
	Count    int    `json:"count"`
}

// DayDeviceCounts aggregates the readings count for multiple devices on a specific day.
type DayDeviceCounts struct {
	Day     time.Time     `json:"day"`
	Devices []DeviceCount `json:"devices"`
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

func (api *API) GetDevicesOverview(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	ctx := r.Context()

	err := api.validate.Var(userID, "required")
	if err != nil {
		api.log.Errorf("validation error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results, err := api.readingsRepo.FetchDevicesOverview(ctx, userID, 30) // Last 30 days
	if err != nil {
		api.log.Errorf("Failed to fetch device overview: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Map domain results to response struct
	response := make([]DayDeviceCounts, len(results))
	for i, dayCounts := range results {
		deviceCounts := make([]DeviceCount, len(dayCounts.Devices))
		for j, device := range dayCounts.Devices {
			deviceCounts[j] = DeviceCount{
				DeviceID: device.DeviceID,
				Count:    device.Count,
			}
		}

		// sort devices by id
		sort.Slice(deviceCounts, func(i, j int) bool {
			return deviceCounts[i].DeviceID < deviceCounts[j].DeviceID
		})

		response[i] = DayDeviceCounts{
			Day:     dayCounts.Day,
			Devices: deviceCounts,
		}
	}
	respondWithJSON(w, response)
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

func (api *API) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			api.log.Infow("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytesWritten", ww.BytesWritten(),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

type wrapResponseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int
}

func NewWrapResponseWriter(w http.ResponseWriter, protoMajor int) *wrapResponseWriter {
	// Default the status code to 200
	return &wrapResponseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (wr *wrapResponseWriter) WriteHeader(code int) {
	wr.status = code
	wr.ResponseWriter.WriteHeader(code)
}

func (wr *wrapResponseWriter) Write(b []byte) (int, error) {
	size, err := wr.ResponseWriter.Write(b)
	wr.bytesWritten += size
	return size, err
}

func (wr *wrapResponseWriter) Status() int {
	return wr.status
}

func (wr *wrapResponseWriter) BytesWritten() int {
	return wr.bytesWritten
}
