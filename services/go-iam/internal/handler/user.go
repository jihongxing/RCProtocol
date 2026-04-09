package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"rcprotocol/services/go-iam/internal/model"
	"rcprotocol/services/go-iam/internal/repo"
)

// UserHandler handles HTTP requests for user CRUD operations.
type UserHandler struct {
	repo repo.UserRepository
}

// NewUserHandler creates a UserHandler with the given repository.
func NewUserHandler(r repo.UserRepository) *UserHandler {
	return &UserHandler{repo: r}
}

type createUserRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// Create handles POST /users.
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if !isValidEmail(req.Email) {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid email format")
		return
	}

	if len(req.Password) < 8 {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		return
	}

	user := &model.User{
		UserID:       uuid.New().String(),
		Email:        req.Email,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
		Status:       "active",
	}

	if err := h.repo.Create(r.Context(), user); err != nil {
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "CONFLICT", "email already exists")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create user")
		return
	}

	WriteSuccess(w, http.StatusCreated, user)
}

// List handles GET /users.
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)

	users, total, err := h.repo.List(r.Context(), page, pageSize)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list users")
		return
	}

	WriteList(w, users, page, pageSize, total)
}

// GetByID handles GET /users/{id}.
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user")
		return
	}

	WriteSuccess(w, http.StatusOK, user)
}

type updateUserRequest struct {
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

// Update handles PUT /users/{id}.
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	user, err := h.repo.Update(r.Context(), id, req.DisplayName, req.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update user")
		return
	}

	WriteSuccess(w, http.StatusOK, user)
}

// Delete handles DELETE /users/{id} — soft-deletes by setting status to "disabled".
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Disable(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to disable user")
		return
	}

	WriteSuccess(w, http.StatusOK, map[string]string{"status": "disabled"})
}
