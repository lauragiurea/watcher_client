package api

import "time"

type Monitor struct {
	ID               uint64    `json:"id"`
	Name             string    `json:"name"`
	URL              string    `json:"url"`
	CSSSelector      *string   `json:"css_selector,omitempty"`
	FrequencySeconds int       `json:"frequency_seconds"`
	NotifyEmail      bool      `json:"notify_email"`
	NotifyEmailAddr  *string   `json:"notify_email_address,omitempty"`
	Active           bool      `json:"active"`
	LastStatus       *string   `json:"last_status,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedAt        time.Time `json:"created_at"`
}

type ChangeEvent struct {
	ID             uint64    `json:"id"`
	MonitorID      uint64    `json:"monitor_id"`
	RunID          uint64    `json:"run_id"`
	HTTPStatusPrev *int      `json:"http_status_prev,omitempty"`
	HTTPStatusCurr *int      `json:"http_status_curr,omitempty"`
	HTMLPrev       *string   `json:"html_prev,omitempty"`
	HTMLCurr       *string   `json:"html_curr,omitempty"`
	HTMLDiff       *string   `json:"html_diff,omitempty"`
	ScreenshotCurr *string   `json:"screenshot_curr,omitempty"`
	ScreenshotPrev *string   `json:"screenshot_prev,omitempty"`
	ScreenshotDiff *string   `json:"screenshot_diff,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
