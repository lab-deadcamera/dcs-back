package push

import (
	"database/sql"
)

// Store persists Web Push subscriptions for each user.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a subscription, or refreshes the keys/user-agent when the
// same (user, endpoint) re-registers (browser rotates keys / updates SW).
func (s *Store) Create(sub *Subscription) error {
	_, err := s.db.Exec(`
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, user_agent)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, endpoint) DO UPDATE
		SET p256dh = EXCLUDED.p256dh,
		    auth = EXCLUDED.auth,
		    user_agent = EXCLUDED.user_agent,
		    updated_at = NOW()`,
		sub.UserID, sub.Endpoint, sub.P256DH, sub.Auth, sub.UserAgent)
	return err
}

// ListByUser returns every device subscribed by the user.
func (s *Store) ListByUser(userID int64) ([]Subscription, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, endpoint, p256dh, auth,
		       COALESCE(user_agent, ''), created_at, updated_at
		FROM push_subscriptions
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.Endpoint, &sub.P256DH, &sub.Auth,
			&sub.UserAgent, &sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// DeleteByEndpoint removes one device subscription for a user.
func (s *Store) DeleteByEndpoint(userID int64, endpoint string) error {
	_, err := s.db.Exec(
		`DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`,
		userID, endpoint)
	return err
}
