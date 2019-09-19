package browser

// TODO: This should be replace when we introduce i18n
func MapLanduse(key string) string {
	switch key {
	case "pa":
		return "Pasture"
	case "me":
		return "Meadow"
	case "fo":
		return "Forest"
	case "sf":
		return "SapFlow"
	case "de":
		return "Dendrometer"
	case "ro":
		return "Rock"
	case "bs":
		return "Bare soil"
	default:
		return key
	}
}

// TODO: This should be replace when we introduce i18n
func MapMeasurements(key string) string {
	switch key {
	case "air_rh_avg":
		return "Relative Humidity"
	case "air_t_avg":
		return "Air Temperature"
	case "wind_dir":
		return "Wind Direction"
	case "wind_speed_avg":
		return "Wind Speed"
	case "wind_speed_max":
		return "Wind Gust"
	default:
		return key
	}
}
