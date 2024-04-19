package api

import (
	"encoding/json"
	"glooko/internal/domain"
	"glooko/internal/ports"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type API struct {
	userRepo   ports.UserRepository
	deviceRepo ports.DeviceRepository
}

func NewAPI(userRepo ports.UserRepository, deviceRepo ports.DeviceRepository) *API {
	return &API{
		userRepo:   userRepo,
		deviceRepo: deviceRepo,
	}
}

func (api *API) Routes() *chi.Mux {
	r := chi.NewRouter()

	// User routes
	r.Route("/users", func(r chi.Router) {
		r.Post("/", api.CreateUser)
		r.Get("/{id}", api.GetUser)
	})

	// Device routes
	r.Route("/devices", func(r chi.Router) {
		r.Post("/", api.CreateDevice)
		r.Get("/{id}", api.GetDevice)
	})

	return r
}

// User handlers
func (api *API) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var user domain.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := api.userRepo.Save(ctx, user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (api *API) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := chi.URLParam(r, "id")
	user, err := api.userRepo.FindByID(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// Device handlers
func (api *API) CreateDevice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var device domain.Device

	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := api.deviceRepo.Save(ctx, device); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(device)
}

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
