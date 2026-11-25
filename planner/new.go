package planner

import (
	"fmt"
	"github.com/xand-stapleton/meetlet-lib/settings"
	"strings"
	"time"
)

// NewMeeting starts at the current minute in the configured source location.
func NewMeeting(cfg settings.Config, now time.Time) (Meeting, error) {
	origin := cfg.Origin()
	loc, err := origin.Load()
	if err != nil {
		return Meeting{}, err
	}
	meeting := Meeting{
		At: now.Truncate(time.Minute).In(loc), Origin: origin, Locations: cfg.Locations,
		Title: "Meeting", Duration: time.Hour,
	}
	return meeting, meeting.Validate()
}

// SaveLocations persists the selection and keeps the same meeting instant in the new source.
func (m *Meeting) SaveLocations(path string, cfg settings.Config) error {
	if err := settings.Save(path, cfg); err != nil {
		return err
	}
	m.Origin = cfg.Origin()
	loc, _ := m.Origin.Load() // Save validated every location.
	m.At = m.At.In(loc)
	m.Locations = cfg.Locations
	return nil
}

func (m *Meeting) SetTitle(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("enter a meeting title")
	}
	m.Title = value
	return nil
}

func (m *Meeting) SetDuration(value string) error {
	d, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || d <= 0 || d > 168*time.Hour {
		return fmt.Errorf("use a duration such as 45m or 1h (maximum 168h)")
	}
	m.Duration = d
	return nil
}

// ShiftedDate preserves the calendar day calculation across clock changes.
func (m Meeting) ShiftedDate(days int) string {
	date, _ := time.Parse(time.DateOnly, m.At.Format(time.DateOnly))
	return date.AddDate(0, 0, days).Format(time.DateOnly)
}
