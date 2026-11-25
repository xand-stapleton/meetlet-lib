package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/xand-stapleton/meetlet-lib/places"
)

// WithLocalSource prepares a session with the system zone first. If a saved
// place uses that zone, keep its label. Otherwise "Local" follows the OS even
// on systems whose zoneinfo file does not identify an IANA name.
func (c Config) WithLocalSource() Config {
	zone := time.Local.String()
	if zone == "Local" {
		path := strings.TrimPrefix(os.Getenv("TZ"), ":")
		if path == "" {
			path, _ = filepath.EvalSymlinks("/etc/localtime")
		}
		if _, name, ok := strings.Cut(path, "/zoneinfo/"); ok {
			zone = name
		}
	}
	return c.withLocalSource(zone)
}

func (c Config) withLocalSource(zone string) Config {
	locations := slices.Clone(c.Locations)
	// A previously saved system row follows the user when travelling.
	locations = slices.DeleteFunc(locations, func(p places.Location) bool { return p.Zone == "Local" })
	i := slices.IndexFunc(locations, func(p places.Location) bool { return p.Zone == zone })
	local := places.Location{Name: "Local time", Zone: "Local"}
	if i >= 0 {
		local = locations[i]
		locations = slices.Delete(locations, i, i+1)
	} else {
		for n := 2; slices.ContainsFunc(locations, func(p places.Location) bool { return strings.EqualFold(p.Name, local.Name) }); n++ {
			local.Name = fmt.Sprintf("Local time (%d)", n)
		}
	}
	c.Locations = append([]places.Location{local}, locations...)
	c.Source, c.localSource = local.Name, local.Name
	return c
}

func (c Config) IsLocalSource(name string) bool { return name == c.localSource }
