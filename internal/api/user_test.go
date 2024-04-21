package api

import (
	"encoding/json"
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
)

func setupAPI() *API {
	userRepo := new(mocks.UserRepository)
	deviceRepo := new(mocks.DeviceRepository)
	readingsRepo := new(mocks.ReadingRepository)

	return NewAPI(userRepo, deviceRepo, readingsRepo)
}

func TestGetUserOverview(t *testing.T) {
	apiInstance := setupAPI()
	readingsRepo := apiInstance.readingsRepo.(*mocks.ReadingRepository) // Type assert to mocked type

	// Setup expected data returned by the mock
	userID := "1234567890abcdef12345678"
	userObjectID, _ := primitive.ObjectIDFromHex(userID)
	start, end := time.Now().AddDate(0, 0, -14), time.Now() // Last 14 days

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

	// Create the router
	r := chi.NewRouter()
	r.Mount("/", apiInstance.Routes())

	// Create a request to the endpoint
	req, err := http.NewRequest("GET", "/users/"+userObjectID.String()+"/overview?start="+start.Format("2006-01-02")+"&end="+end.Format("2006-01-02"), nil)
	assert.NoError(t, err)

	// Record the response
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, w.Code)

	// Check the body
	var response UserOverviewResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	// Further assertions on the response content
	assert.Len(t, response.Overview, 1)
	assert.Equal(t, userID, response.Overview[0].UserID)
	assert.Equal(t, 100, response.Overview[0].Metrics.MinValue)
	assert.Equal(t, 105, response.Overview[0].Metrics.MaxValue)
	assert.Equal(t, 102.5, response.Overview[0].Metrics.AvgValue)
}
