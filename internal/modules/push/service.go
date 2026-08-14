package push

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// Service owns subscription persistence and Web Push delivery.
// VAPID keys are read from config; when they are absent the service is a
// no-op so the app keeps working without push configured.
type Service struct {
	store           *Store
	vapidPublicKey  string
	vapidPrivateKey string
	vapidSubject    string
	client          *http.Client
}

func NewService(store *Store, vapidPublicKey, vapidPrivateKey, vapidSubject string) *Service {
	return &Service{
		store:           store,
		vapidPublicKey:  vapidPublicKey,
		vapidPrivateKey: vapidPrivateKey,
		vapidSubject:    vapidSubject,
		client:          &http.Client{Timeout: 15 * time.Second},
	}
}

// Enabled reports whether VAPID is configured and pushes can actually be sent.
func (s *Service) Enabled() bool {
	return s.vapidPublicKey != "" && s.vapidPrivateKey != "" && s.vapidSubject != ""
}

// Register stores a new device subscription for the user.
func (s *Service) Register(userID int64, req *SubscriptionRequest, userAgent string) error {
	return s.store.Create(&Subscription{
		UserID:    userID,
		Endpoint:  req.Endpoint,
		P256DH:    req.Keys.P256DH,
		Auth:      req.Keys.Auth,
		UserAgent: userAgent,
	})
}

// Unregister removes one device subscription for the user.
func (s *Service) Unregister(userID int64, endpoint string) error {
	return s.store.DeleteByEndpoint(userID, endpoint)
}

// SendTest delivers a test notification to every device subscribed by the
// user (diagnostic helper for the /push/test endpoint). Returns how many
// subscriptions were targeted.
func (s *Service) SendTest(userID int64) (int, error) {
	if !s.Enabled() {
		return 0, fmt.Errorf("push not configured: set PUSH_VAPID_PUBLIC_KEY, PUSH_VAPID_PRIVATE_KEY and PUSH_VAPID_SUBJECT")
	}

	subs, err := s.store.ListByUser(userID)
	if err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}

	payload, err := buildPayload("🔔 Push test", "If you see this, the whole chain works.", map[string]string{
		"type": "push-test",
	})
	if err != nil {
		return 0, err
	}

	for _, sub := range subs {
		s.sendOne(&sub, payload)
	}
	return len(subs), nil
}

// SendToUser pushes a notification to every device subscribed by the user.
// Never returns an error to the caller — delivery problems are logged and a
// gone/expired subscription is cleaned up. Safe to run in a goroutine.
func (s *Service) SendToUser(userID int64, title, body string, data map[string]string) {
	if !s.Enabled() {
		return
	}

	subs, err := s.store.ListByUser(userID)
	if err != nil {
		log.Printf("[push] list subscriptions user=%d: %v", userID, err)
		return
	}

	payload, err := buildPayload(title, body, data)
	if err != nil {
		log.Printf("[push] build payload: %v", err)
		return
	}

	for _, sub := range subs {
		s.sendOne(&sub, payload)
	}
}

// sendOne delivers an already-encoded payload to a single subscription.
func (s *Service) sendOne(sub *Subscription, payload []byte) {
	resp, err := webpush.SendNotificationWithContext(context.Background(), payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256DH,
			Auth:   sub.Auth,
		},
	}, &webpush.Options{
		HTTPClient:      s.client,
		Subscriber:      s.vapidSubject,
		VAPIDPublicKey:  s.vapidPublicKey,
		VAPIDPrivateKey: s.vapidPrivateKey,
		TTL:             60,
	})
	if err != nil {
		log.Printf("[push] send to %s: %v", sub.Endpoint, err)
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		return
	case http.StatusNotFound, http.StatusGone:
		// The browser unsubscribed or the push service dropped the endpoint.
		log.Printf("[push] subscription gone (%d), removing %s", resp.StatusCode, sub.Endpoint)
		if delErr := s.store.DeleteByEndpoint(sub.UserID, sub.Endpoint); delErr != nil {
			log.Printf("[push] remove gone subscription: %v", delErr)
		}
	default:
		log.Printf("[push] send to %s returned %d", sub.Endpoint, resp.StatusCode)
	}
}

// buildPayload encodes the Web Push body in the format Angular's ngsw-worker
// understands: { notification: { title, body, icon, badge, data } }.
func buildPayload(title, body string, data map[string]string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"notification": map[string]interface{}{
			"title": title,
			"body":  body,
			"icon":  "/assets/icons/192x192.png",
			"badge": "/assets/icons/192x192.png",
			"data":  data,
		},
	})
}
