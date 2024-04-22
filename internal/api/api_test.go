package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"glooko/internal/domain"
	"glooko/internal/mocks"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func setupAPI() *API {
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	userRepo := new(mocks.UserRepository)
	deviceRepo := new(mocks.DeviceRepository)
	readingsRepo := new(mocks.ReadingRepository)

	return NewAPI(log, userRepo, deviceRepo, readingsRepo)
}

func TestGetUserOverview(t *testing.T) {
	apiInstance := setupAPI()
	readingsRepo := apiInstance.readingsRepo.(*mocks.ReadingRepository)

	userID := "1234567890abcdef12345678"
	userObjectID, _ := primitive.ObjectIDFromHex(userID)

	testCases := []struct {
		name        string
		userID      string
		start       string
		end         string
		setupMock   func()
		expectCode  int
		expectLen   int
		expectError bool
	}{
		{
			name:   "Valid Request",
			userID: userID,
			start:  "2006-04-06",
			end:    "2006-04-20",
			setupMock: func() {
				start, end, _ := parseDates("2006-04-06", "2006-04-20")
				readings := []domain.Reading{
					{
						UserID: userObjectID,
						Day:    start,
						Readings: []domain.ReadingEntry{
							{Time: start.Add(8 * time.Hour), Value: 100},
							{Time: start.Add(9 * time.Hour), Value: 105},
						},
						MinValue: 100,
						MaxValue: 105,
						AvgValue: 102.5,
					},
				}
				readingsRepo.On("FetchReadings", mock.Anything, userID, start, end).Return(readings, nil)
			},
			expectCode:  http.StatusOK,
			expectLen:   1,
			expectError: false,
		},
		{
			name:        "Invalid Date Format",
			userID:      userID,
			start:       "April 06, 2006",
			end:         "April 20, 2006",
			setupMock:   func() {},
			expectCode:  http.StatusBadRequest,
			expectError: true,
		},
		{
			name:   "No Data Available",
			userID: userID,
			start:  "2006-04-01",
			end:    "2006-04-05",
			setupMock: func() {
				start, end, _ := parseDates("2006-04-01", "2006-04-05")
				readingsRepo.On("FetchReadings", mock.Anything, userID, start, end).Return([]domain.Reading{}, nil)
			},
			expectCode:  http.StatusOK,
			expectLen:   0,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock behavior
			tc.setupMock()

			url := fmt.Sprintf("/users/%s/overview?start=%s&end=%s", tc.userID, tc.start, tc.end)
			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			w := httptest.NewRecorder()
			r := chi.NewRouter()
			r.Mount("/", apiInstance.Routes())
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectCode, w.Code)

			if !tc.expectError {
				var response UserOverviewResponse
				err = json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Len(t, response.Overview, tc.expectLen)
			}
		})
	}
}

func TestGetUserOverview_MultipleDevices(t *testing.T) {
	apiInstance := setupAPI()
	readingsRepo := apiInstance.readingsRepo.(*mocks.ReadingRepository) // Cast to mocked type

	userID := "1234567890abcdef12345678"
	userObjectID, _ := primitive.ObjectIDFromHex(userID)

	// Set up dates
	start, end, err := parseDates("2006-04-06", "2006-04-20")
	assert.NoError(t, err)

	readings := []domain.Reading{
		{
			UserID:   userObjectID,
			DeviceID: primitive.NewObjectID(),
			Day:      start,
			Readings: []domain.ReadingEntry{
				{Time: start.Add(8 * time.Hour), Value: 100},
				{Time: start.Add(9 * time.Hour), Value: 105},
			},
			MinValue: 100,
			MaxValue: 105,
			AvgValue: 102.5,
		},
		{
			UserID:   userObjectID,
			DeviceID: primitive.NewObjectID(),
			Day:      start,
			Readings: []domain.ReadingEntry{
				{Time: start.Add(10 * time.Hour), Value: 110},
				{Time: start.Add(11 * time.Hour), Value: 115},
				{Time: start.Add(25 * time.Hour), Value: 115}, // next day
			},
			MinValue: 110,
			MaxValue: 115,
			AvgValue: 112.5,
		},
	}

	readingsRepo.On("FetchReadings", mock.Anything, userID, start, end).Return(readings, nil)

	url := fmt.Sprintf("/users/%s/overview?start=%s&end=%s", userID, "2006-04-06", "2006-04-20")
	req, err := http.NewRequest("GET", url, nil)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	r := chi.NewRouter()
	r.Mount("/", apiInstance.Routes())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response UserOverviewResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Len(t, response.Overview, 1)
	assert.Equal(t, userID, response.Overview[0].UserID)
	assert.Equal(t, 4, len(response.Overview[0].Measure))
	assert.Equal(t, 100, response.Overview[0].Metrics.MinValue)
	assert.Equal(t, 115, response.Overview[0].Metrics.MaxValue)
	assert.Equal(t, 107.5, response.Overview[0].Metrics.AvgValue)
}

func TestGetDevicesOverview(t *testing.T) {
	apiInstance := setupAPI()
	readingsRepo := apiInstance.readingsRepo.(*mocks.ReadingRepository)

	userID := "1234567890abcdef12345678"

	testCases := []struct {
		name        string
		userID      string
		days        int
		setupMock   func()
		expectCode  int
		expectLen   int
		expectError bool
	}{
		{
			name:   "Valid Request",
			userID: userID,
			days:   30,
			setupMock: func() {
				start := time.Now().AddDate(0, 0, -30)
				deviceOverviews := []domain.DayDeviceCounts{
					{
						Day: start,
						Devices: []domain.DeviceCount{
							{DeviceID: "device1", Count: 10},
							{DeviceID: "device2", Count: 15},
						},
					},
				}
				readingsRepo.On("FetchDevicesOverview", mock.Anything, userID, 30).Return(deviceOverviews, nil)
			},
			expectCode:  http.StatusOK,
			expectLen:   1,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock()

			url := fmt.Sprintf("/users/%s/devices-overview", tc.userID)
			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			w := httptest.NewRecorder()
			r := chi.NewRouter()
			r.Mount("/", apiInstance.Routes())
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectCode, w.Code)

			if !tc.expectError {
				var response []DayDeviceCounts
				err = json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Len(t, response, tc.expectLen)
			}
		})
	}
}
