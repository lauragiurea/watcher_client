package api

import "time"

type Monitor struct {
	ID               uint64    `json:"id"`
	Name             string    `json:"name"`
	URL              string    `json:"url"`
	CSSSelector      *string   `json:"css_selector,omitempty"`
	RenderJS         bool      `json:"render_js"`
	FrequencySeconds int       `json:"frequency_seconds"`
	NotifyEmail      bool      `json:"notify_email"`
	NotifyEmailAddr  *string   `json:"notify_email_address,omitempty"`
	Active           bool      `json:"active"`
	LastStatus       *string   `json:"last_status,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedAt        time.Time `json:"created_at"`
}

type ChangeEvent struct {
	ID                uint64    `json:"id"`
	MonitorID         uint64    `json:"monitor_id"`
	RunID             uint64    `json:"run_id"`
	TextDiffHTML      *string   `json:"text_diff_html,omitempty"`
	ContentURL        *string   `json:"content_url,omitempty"`
	HTMLSnapshotURL   *string   `json:"html_snapshot_url,omitempty"`
	ScreenshotURL     *string   `json:"screenshot_url,omitempty"`
	ScreenshotPrevURL *string   `json:"screenshot_prev_url,omitempty"`
	ScreenshotDiffURL *string   `json:"screenshot_diff_url,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}
