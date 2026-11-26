package holidays

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xand-stapleton/meetlet-lib/places"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRegionalHolidaysAreExcludedForLondon(t *testing.T) {
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		payload := `[{"date":"2026-04-06","name":"Easter Monday","global":false,"counties":["GB-ENG"],"types":["Public"]},{"date":"2026-12-25","name":"Christmas Day","global":true,"types":["Public"]}]`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload))}, nil
	})}
	c := Client{endpoint: "https://holidays.example", http: client}
	london := places.Location{Name: "London", Zone: "Europe/London"}
	day := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	got := c.lookup(london, day, day)
	if got.Next.Name != "Christmas Day" || len(got.OnMeetingDay) != 0 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}

func TestPublicHolidaysUseLocalDates(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		var payload string
		switch r.URL.Path {
		case "/2026/JP":
			payload = `[{"date":"2026-12-31","name":"Observance","global":true,"types":["Observance"]}]`
		case "/2027/JP":
			payload = `[{"date":"2027-01-01","name":"New Year's Day","global":true,"types":["Public"]},{"date":"2027-01-01","name":"Regional","global":false,"counties":["JP-13"],"types":["Public"]}]`
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload))}, nil
	})}
	c := Client{endpoint: "https://holidays.example", http: client}
	tokyo := places.Location{Name: "Tokyo", Zone: "Asia/Tokyo"}
	now := time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC)
	meeting := now.Add(4 * time.Hour)
	for range 2 {
		got := c.lookup(tokyo, now, meeting)
		if got.Next.Date != "2027-01-01" || len(got.OnMeetingDay) != 1 || got.OnMeetingDay[0] != got.Next {
			t.Fatalf("unexpected summary: %+v", got)
		}
	}
	if requests != 2 {
		t.Fatalf("expected two cached year requests, got %d", requests)
	}
}
