package push

import (
	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register handles POST /api/v1/push/subscriptions.
// Body: PushSubscriptionJSON ({ endpoint, expirationTime, keys: { p256dh, auth } }).
func (h *Handler) Register(c *gin.Context) {
	var req SubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	userID := userIDFromContext(c)
	if userID <= 0 {
		utils.Unauthorized(c, "missing user id")
		return
	}

	if err := h.svc.Register(userID, &req, c.Request.UserAgent()); err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, gin.H{"subscribed": true})
}

// Unregister handles DELETE /api/v1/push/subscriptions?endpoint=...
func (h *Handler) Unregister(c *gin.Context) {
	endpoint := c.Query("endpoint")
	if endpoint == "" {
		utils.BadRequest(c, "endpoint is required")
		return
	}

	userID := userIDFromContext(c)
	if userID <= 0 {
		utils.Unauthorized(c, "missing user id")
		return
	}

	if err := h.svc.Unregister(userID, endpoint); err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Message(c, "unsubscribed")
}

// Test handles POST /api/v1/push/test — sends a test notification to the
// authenticated user's devices (diagnostic helper).
func (h *Handler) Test(c *gin.Context) {
	userID := userIDFromContext(c)
	if userID <= 0 {
		utils.Unauthorized(c, "missing user id")
		return
	}

	n, err := h.svc.SendTest(userID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if n == 0 {
		utils.Message(c, "no push subscriptions registered for this user")
		return
	}
	utils.Success(c, gin.H{"sent_to": n})
}

// userIDFromContext reads the JWT user id set by the auth middleware.
func userIDFromContext(c *gin.Context) int64 {
	if v, ok := c.Get("userID"); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}
