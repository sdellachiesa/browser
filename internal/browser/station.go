package browser

import (
	"encoding/json"
	"strconv"

	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
)

// Station defines metadata about a physical LTER station.
// We use a custom type and not the SnipeIT locations type since,
// first we do not need all SnipeIT fields and moreover we map some
// fields do custom onse since they aren't supported or have an other
// name inside SnipeIT.
type Station struct {
	ID        string
	Name      string
	Landuse   string
	Image     string
	Altitude  int64
	Latitude  float64
	Longitude float64
}

// UnmarshalJSON is a custom JSON unmarshaler which maps SnipeIT
// location fields to the right station field with the right name.
func (s *Station) UnmarshalJSON(b []byte) error {
	var l snipeit.Location
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}

	s.ID = strconv.FormatInt(l.ID, 10)
	s.Name = l.Name
	s.Landuse = l.Currency
	s.Image = l.Image
	s.Altitude, _ = strconv.ParseInt(l.Zip, 10, 64)
	s.Latitude, _ = strconv.ParseFloat(l.Address, 64)
	s.Longitude, _ = strconv.ParseFloat(l.Address2, 64)

	return nil
}
