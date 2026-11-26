// Package holidays looks up public holidays by the civil date in a location.
package holidays

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/xand-stapleton/meetlet-lib/places"
)

//go:embed zones.tab
var zoneTable string

var countries = func() map[string]string {
	result := make(map[string]string)
	for line := range strings.SplitSeq(zoneTable, "\n") {
		if fields := strings.Fields(line); len(fields) >= 3 && !strings.HasPrefix(line, "#") {
			result[fields[2]] = fields[0]
		}
	}
	return result
}()

const endpoint = "https://date.nager.at/api/v3/PublicHolidays"

var publicCalendars = Client{endpoint: endpoint, http: &http.Client{Timeout: 5 * time.Second}}

type Holiday struct {
	Date string
	Name string
}

type Summary struct {
	Next         Holiday
	OnMeetingDay []Holiday
	Available    bool
}

func (s Summary) NextLabel() string {
	if s.Next.Date != "" {
		return "Next public holiday · " + s.Next.Date + " · " + s.Next.Name
	}
	if s.Available {
		return "No upcoming public holiday found"
	}
	return "Public holiday data unavailable"
}

type entry struct {
	Holiday
	Global bool
	Types  []string
}

type yearKey struct {
	country string
	year    int
}

type calendar struct {
	ready   chan struct{}
	entries []entry
	ok      bool
	expires time.Time
}

type Client struct {
	endpoint string
	http     *http.Client
	mu       sync.Mutex
	years    map[yearKey]*calendar
}

// Lookup uses the shared Nager.Date cache. It can make network requests.
func Lookup(place places.Location, now, meeting time.Time) Summary {
	return publicCalendars.lookup(place, now, meeting)
}

func (c *Client) lookup(place places.Location, now, meeting time.Time) Summary {
	loc, err := place.Load()
	if err != nil {
		return Summary{}
	}
	zone := place.Zone
	if zone == "Local" {
		zone = loc.String()
		if path, err := filepath.EvalSymlinks("/etc/localtime"); zone == "Local" && err == nil {
			_, zone, _ = strings.Cut(path, "/zoneinfo/")
		}
	}
	country := countries[zone]
	if country == "" {
		return Summary{}
	}
	now, meeting = now.In(loc), meeting.In(loc)
	today, meetingDate := now.Format(time.DateOnly), meeting.Format(time.DateOnly)
	var result Summary
	years := []int{now.Year(), now.Year() + 1}
	if !slices.Contains(years, meeting.Year()) {
		years = append(years, meeting.Year())
	}
	for _, year := range years {
		// Once the next holiday is known, only the meeting's year can add information.
		if year != meeting.Year() && result.Next.Date != "" {
			continue
		}
		entries, ok := c.year(country, year)
		if year == now.Year() {
			result.Available = ok
		}
		for _, holiday := range entries {
			if !slices.Contains(holiday.Types, "Public") || !holiday.Global {
				continue
			}
			if result.Available && year <= now.Year()+1 && holiday.Date >= today && (result.Next.Date == "" || holiday.Date < result.Next.Date) {
				result.Next = holiday.Holiday
			}
			if holiday.Date == meetingDate && !slices.Contains(result.OnMeetingDay, holiday.Holiday) {
				result.OnMeetingDay = append(result.OnMeetingDay, holiday.Holiday)
			}
		}
	}
	return result
}

func (c *Client) year(country string, year int) ([]entry, bool) {
	key := yearKey{country, year}
	c.mu.Lock()
	cached := c.years[key]
	if cached != nil && (cached.expires.IsZero() || time.Now().Before(cached.expires)) {
		c.mu.Unlock()
		<-cached.ready
		return cached.entries, cached.ok
	}
	cached = &calendar{ready: make(chan struct{})}
	if c.years == nil {
		c.years = make(map[yearKey]*calendar)
	}
	c.years[key] = cached
	c.mu.Unlock()

	response, err := c.http.Get(fmt.Sprintf("%s/%d/%s", c.endpoint, year, country))
	if err == nil {
		defer response.Body.Close()
		if response.StatusCode == http.StatusOK {
			cached.ok = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&cached.entries) == nil
		}
	}
	ttl := 24 * time.Hour
	if !cached.ok {
		cached.entries = nil
		ttl = time.Minute
	}
	c.mu.Lock()
	cached.expires = time.Now().Add(ttl)
	close(cached.ready)
	c.mu.Unlock()
	return cached.entries, cached.ok
}
