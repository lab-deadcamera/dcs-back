package file

import (
	"log"
	"time"
)

func (s *Service) StartPurgeCron() {
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			delay := next.Sub(now)
			log.Printf("[file-cron] next temp purge at %s (in %s)", next.Format("2006-01-02 15:04:05"), delay)
			time.Sleep(delay)

			log.Println("[file-cron] running temp purge...")
			if err := s.PurgeExpiredTemp(); err != nil {
				log.Printf("[file-cron] purge error: %v", err)
			} else {
				log.Println("[file-cron] temp purge completed")
			}

			time.Sleep(24 * time.Hour)
		}
	}()
}
