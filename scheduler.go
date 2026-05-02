package main

import (
	"strings"
	"time"
)

func startScheduler(times []string, onReminder func(targetTime string)) {
	go func() {
		triggered := map[string]bool{}

		check := func(now time.Time) {
			currentHHMM := now.Format("15:04")
			today := now.Format("2006-01-02")

			for _, target := range times {
				if currentHHMM != target {
					continue
				}

				key := today + " " + target
				if triggered[key] {
					continue
				}

				triggered[key] = true
				onReminder(target)
			}

			for key := range triggered {
				if !strings.HasPrefix(key, today) {
					delete(triggered, key)
				}
			}
		}

		check(time.Now())

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for now := range ticker.C {
			check(now)
		}
	}()
}
