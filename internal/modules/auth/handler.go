package auth

import (
	"errors"
	"strconv"

	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ─── Public ────────────────────────────────────────────────────────

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	user, err := h.service.Register(&req)
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			utils.Conflict(c, err.Error())
			return
		}
		utils.InternalError(c, "internal server error")
		return
	}

	utils.Created(c, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	token, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCreds) {
			utils.Unauthorized(c, err.Error())
			return
		}
		utils.InternalError(c, "internal server error")
		return
	}

	utils.Success(c, token)
}

// ─── Protected (authenticated) ─────────────────────────────────────

func (h *Handler) GetProfile(c *gin.Context) {
	userID, _ := c.Get("userID")
	id, ok := userID.(int64)
	if !ok {
		utils.Unauthorized(c, "invalid token")
		return
	}

	user, err := h.service.GetUserProfile(id)
	if err != nil {
		utils.NotFound(c, err.Error())
		return
	}

	utils.Success(c, user)
}

// ─── Admin (role-gated) ────────────────────────────────────────────

func (h *Handler) CreateUser(c *gin.Context) {
	callerLevelRaw, _ := c.Get("role_level")
	callerLevel, ok := callerLevelRaw.(float64)
	if !ok {
		utils.Unauthorized(c, "invalid token")
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	user, err := h.service.CreateUser(int(callerLevel), &req)
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			utils.Conflict(c, err.Error())
			return
		}
		if errors.Is(err, ErrInsufficientRole) || errors.Is(err, ErrCannotCreateAdmin) {
			utils.Unauthorized(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Created(c, user)
}

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.service.ListUsers()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, users)
}

func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.service.store.ListRoles()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, roles)
}

func (h *Handler) UpdateUserActive(c *gin.Context) {
	callerIDRaw, _ := c.Get("userID")
	callerID, ok := callerIDRaw.(int64)
	if !ok {
		utils.Unauthorized(c, "invalid token")
		return
	}

	callerLevelRaw, _ := c.Get("role_level")
	callerLevel, ok2 := callerLevelRaw.(float64)
	if !ok2 {
		utils.Unauthorized(c, "invalid token")
		return
	}

	targetID, err := ParseUserID(c)
	if err != nil {
		utils.BadRequest(c, "invalid user id")
		return
	}

	var req struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if err := h.service.UpdateUserActive(callerID, int(callerLevel), targetID, req.Active); err != nil {
		switch {
		case errors.Is(err, ErrCannotDeactivateSelf):
			utils.BadRequest(c, err.Error())
		case errors.Is(err, ErrCannotDeactivateSA):
			utils.BadRequest(c, err.Error())
		default:
			utils.InternalError(c, err.Error())
		}
		return
	}

	utils.Success(c, gin.H{"active": req.Active})
}

// ─── Auth helpers exposed for middleware ───────────────────────────

// GetCallerLevel extracts the role level from a JWT-authenticated request.
func (h *Handler) GetCallerLevel(c *gin.Context) int {
	raw, _ := c.Get("role_level")
	if level, ok := raw.(float64); ok {
		return int(level)
	}
	return 99 // deny if not found
}

// ParseUserID extracts the numeric user id from a route param.
func ParseUserID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}
