package push

import "time"

// Subscription is a Web Push subscription registered for a user's device.
// Mirrors the browser's PushSubscription JSON shape plus the owning user.
type Subscription struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	P256DH    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SubscriptionRequest is the body the frontend sends when registering the
// device (`SwPush.requestSubscription()` → PushSubscriptionJSON).
type SubscriptionRequest struct {
	Endpoint       string           `json:"endpoint" binding:"required"`
	ExpirationTime *time.Time       `json:"expirationTime"`
	Keys           SubscriptionKeys `json:"keys"`
}

// SubscriptionKeys holds the ECDH public key (p256dh) and auth secret
// required by the Web Push protocol to encrypt outgoing messages.
type SubscriptionKeys struct {
	P256DH string `json:"p256dh" binding:"required"`
	Auth   string `json:"auth" binding:"required"`
}
