package settings

import (
	"github.com/xand-stapleton/meetlet-lib/places"
	"testing"
)

func TestLocalSourceAtLaunch(t *testing.T) {
	c := Default()
	c.Source = "London"
	first := c.withLocalSource("Europe/Berlin")
	if first.Locations[0].Name != "Bonn" || first.Source != "Bonn" || len(first.Locations) != len(c.Locations) {
		t.Fatal(first)
	}
	first.Source = "London"
	if first.RemoveLocation("Bonn") == nil {
		t.Fatal("removed pinned zone")
	}
	c.Locations = append(c.Locations, places.Location{Name: "Local time", Zone: "UTC"})
	away := c.withLocalSource("Asia/Tokyo")
	if away.Origin().Zone != "Local" || away.Validate() != nil {
		t.Fatal(away)
	}
	back := away.withLocalSource("Europe/London")
	if back.Origin().Name != "London" || len(back.Locations) != len(c.Locations) {
		t.Fatal(back)
	}
}
