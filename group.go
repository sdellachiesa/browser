// Copyright 2021 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package browser

const (
	AirTemperature Group = iota
	RelativeHumidity
	SoilTemperature
	SoilTemperatureDepth00
	SoilTemperatureDepth02
	SoilTemperatureDepth05
	SoilTemperatureDepth10
	SoilTemperatureDepth20
	SoilTemperatureDepth40
	SoilTemperatureDepth50
	SoilWaterContent
	SoilWaterContentDepth02
	SoilWaterContentDepth05
	SoilWaterContentDepth20
	SoilWaterContentDepth40
	SoilWaterContentDepth50
	SoilElectricalConductivity
	SoilElectricalConductivityDepth02
	SoilElectricalConductivityDepth05
	SoilElectricalConductivityDepth20
	SoilElectricalConductivityDepth40
	SoilElectricalConductivityDepth51
	SoilDielectricPermittivity
	SoilDielectricPermittivityDepth02
	SoilDielectricPermittivityDepth05
	SoilDielectricPermittivityDepth20
	SoilDielectricPermittivityDepth40
	SoilDielectricPermittivityDepth52
	SoilWaterPotential
	SoilWaterPotentialDepth05
	SoilWaterPotentialDepth20
	SoilWaterPotentialDepth40
	SoilWaterPotentialDepth50
	SoilHeatFlux
	SoilSurfaceTemperature
	WindSpeed
	WindSpeedAvg
	WindSpeedMax
	WindDirection
	Precipitation
	PrecipitationTotal
	PrecipitationIntensity
	SnowHeight
	LeafWetnessDuration
	SunshineDuration
	PhotosyntheticallyActiveRadiation
	PhotosyntheticallyActiveRadiationTotal
	PhotosyntheticallyActiveRadiationDiffuse
	PhotosyntheticallyActiveRadiationAtSoilLevel
	NDVIRadiations
	PRIRadiations
	ShortWaveRadiation
	ShortWaveRadiationIncoming
	ShortWaveRadiationOutgoing
	LongWaveRadiation
	LongWaveRadiationIncoming
	LongWaveRadiationOutgoing
	NoGroup
)

// Group combines multiple measurements to a single entity.
type Group uint8

func (g Group) String() string {
	switch g {
	default:
		return "No Group"
	case AirTemperature:
		return "Air Temperature"
	case RelativeHumidity:
		return "Relative Humidity"
	case SoilTemperature:
		return "Soil Temperature"
	case SoilWaterContent:
		return "Soil Water Content"
	case SoilElectricalConductivity:
		return "Soil Electrical Conductivity"
	case SoilDielectricPermittivity:
		return "Soil Dielectric Permittivity"
	case SoilWaterPotential:
		return "Soil Water Potential"
	case SoilHeatFlux:
		return "Soil Heat Flux"
	case SoilSurfaceTemperature:
		return "Soil Surface Temperature"
	case WindSpeed:
		return "Wind Speed"
	case WindDirection:
		return "Wind Direction"
	case Precipitation:
		return "Precipitation"
	case SnowHeight:
		return "Snow Height"
	case LeafWetnessDuration:
		return "Leaf Wetness Duration"
	case SunshineDuration:
		return "Sunshine Duration"
	case PhotosyntheticallyActiveRadiation:
		return "Photosynthetically Active Radiation"
	case NDVIRadiations:
		return "NDVI Radiations"
	case PRIRadiations:
		return "PRI Radiations"
	case ShortWaveRadiation:
		return "Short Wave Radiation"
	case LongWaveRadiation:
		return "Long Wave Radiation"
	case SoilTemperatureDepth00:
		return "0 cm"
	case SoilTemperatureDepth02, SoilWaterContentDepth02, SoilElectricalConductivityDepth02, SoilDielectricPermittivityDepth02:
		return "2 cm"
	case SoilTemperatureDepth05, SoilWaterContentDepth05, SoilElectricalConductivityDepth05, SoilDielectricPermittivityDepth05, SoilWaterPotentialDepth05:
		return "5 cm"
	case SoilTemperatureDepth10:
		return "10 cm"
	case SoilTemperatureDepth20, SoilWaterContentDepth20, SoilElectricalConductivityDepth20, SoilDielectricPermittivityDepth20, SoilWaterPotentialDepth20:
		return "20 cm"
	case SoilTemperatureDepth40, SoilWaterContentDepth40, SoilElectricalConductivityDepth40, SoilDielectricPermittivityDepth40, SoilWaterPotentialDepth40:
		return "40 cm"
	case SoilTemperatureDepth50, SoilWaterContentDepth50, SoilWaterPotentialDepth50:
		return "50 cm"
	case SoilElectricalConductivityDepth51:
		return "51 cm"
	case SoilDielectricPermittivityDepth52:
		return "52 cm"
	case WindSpeedAvg:
		return "Average"
	case WindSpeedMax:
		return "Max"
	case PrecipitationTotal:
		return "Total"
	case PrecipitationIntensity:
		return "Intensity"
	case PhotosyntheticallyActiveRadiationTotal:
		return "Total Incoming"
	case PhotosyntheticallyActiveRadiationDiffuse:
		return "Diffuse Incoming"
	case PhotosyntheticallyActiveRadiationAtSoilLevel:
		return "At Soil Level Incoming"
	case ShortWaveRadiationIncoming, LongWaveRadiationIncoming:
		return "Incoming"
	case ShortWaveRadiationOutgoing, LongWaveRadiationOutgoing:
		return "Outgoing"
	}
}

// Public returns the group name as string for the public user.
func (g Group) Public() string {
	switch g {
	default:
		return g.String()
	case WindSpeedAvg:
		return "Wind Speed"
	case WindSpeedMax:
		return "Wind Max"
	case ShortWaveRadiationOutgoing:
		return "Global Radiation"
	case PrecipitationTotal:
		return "Precipitation"
	}
}

// SubGroups will return a list of sub groups. An empty slice indicates that no
// sub groups are defined.
func (g Group) SubGroups() []Group {
	switch g {
	default:
		return []Group{}

	case SoilTemperature:
		return []Group{
			SoilTemperatureDepth00,
			SoilTemperatureDepth02,
			SoilTemperatureDepth05,
			SoilTemperatureDepth10,
			SoilTemperatureDepth20,
			SoilTemperatureDepth40,
			SoilTemperatureDepth50,
		}

	case SoilWaterContent:
		return []Group{
			SoilWaterContentDepth02,
			SoilWaterContentDepth05,
			SoilWaterContentDepth20,
			SoilWaterContentDepth40,
			SoilWaterContentDepth50,
		}

	case SoilElectricalConductivity:
		return []Group{
			SoilElectricalConductivityDepth02,
			SoilElectricalConductivityDepth05,
			SoilElectricalConductivityDepth20,
			SoilElectricalConductivityDepth40,
			SoilElectricalConductivityDepth51,
		}

	case SoilDielectricPermittivity:
		return []Group{
			SoilDielectricPermittivityDepth02,
			SoilDielectricPermittivityDepth05,
			SoilDielectricPermittivityDepth20,
			SoilDielectricPermittivityDepth40,
			SoilDielectricPermittivityDepth52,
		}

	case SoilWaterPotential:
		return []Group{
			SoilWaterPotentialDepth05,
			SoilWaterPotentialDepth20,
			SoilWaterPotentialDepth40,
			SoilWaterPotentialDepth50,
		}

	case WindSpeed:
		return []Group{WindSpeedAvg, WindSpeedMax}

	case Precipitation:
		return []Group{PrecipitationTotal, PrecipitationIntensity}

	case PhotosyntheticallyActiveRadiation:
		return []Group{
			PhotosyntheticallyActiveRadiationTotal,
			PhotosyntheticallyActiveRadiationDiffuse,
			PhotosyntheticallyActiveRadiationAtSoilLevel,
		}

	case ShortWaveRadiation:
		return []Group{ShortWaveRadiationIncoming, ShortWaveRadiationOutgoing}

	case LongWaveRadiation:
		return []Group{LongWaveRadiationIncoming, LongWaveRadiationOutgoing}

	}
}

type GroupType uint8

const (
	ParentGroup GroupType = iota
	SubGroup
)

func GroupsByType(t GroupType) []Group {
	switch t {
	default:
		return []Group{
			AirTemperature,
			RelativeHumidity,
			SoilTemperature,
			SoilWaterContent,
			SoilElectricalConductivity,
			SoilDielectricPermittivity,
			SoilWaterPotential,
			SoilHeatFlux,
			SoilSurfaceTemperature,
			WindSpeed,
			WindDirection,
			Precipitation,
			SnowHeight,
			LeafWetnessDuration,
			SunshineDuration,
			PhotosyntheticallyActiveRadiation,
			NDVIRadiations,
			PRIRadiations,
			ShortWaveRadiation,
			LongWaveRadiation,
		}
	case SubGroup:
		return []Group{
			SoilTemperatureDepth00,
			SoilTemperatureDepth02,
			SoilTemperatureDepth05,
			SoilTemperatureDepth10,
			SoilTemperatureDepth20,
			SoilTemperatureDepth40,
			SoilTemperatureDepth50,
			SoilWaterContentDepth02,
			SoilWaterContentDepth05,
			SoilWaterContentDepth20,
			SoilWaterContentDepth40,
			SoilWaterContentDepth50,
			SoilElectricalConductivityDepth02,
			SoilElectricalConductivityDepth05,
			SoilElectricalConductivityDepth20,
			SoilElectricalConductivityDepth40,
			SoilElectricalConductivityDepth51,
			SoilDielectricPermittivityDepth02,
			SoilDielectricPermittivityDepth05,
			SoilDielectricPermittivityDepth20,
			SoilDielectricPermittivityDepth40,
			SoilDielectricPermittivityDepth52,
			SoilWaterPotentialDepth05,
			SoilWaterPotentialDepth20,
			SoilWaterPotentialDepth40,
			SoilWaterPotentialDepth50,
			WindSpeedAvg,
			WindSpeedMax,
			PrecipitationTotal,
			PrecipitationIntensity,
			PhotosyntheticallyActiveRadiationTotal,
			PhotosyntheticallyActiveRadiationDiffuse,
			PhotosyntheticallyActiveRadiationAtSoilLevel,
			ShortWaveRadiationIncoming,
			ShortWaveRadiationOutgoing,
			LongWaveRadiationIncoming,
			LongWaveRadiationOutgoing,
		}
	}
}

// GroupsByRole will return a list of groups for the given role.
func GroupsByRole(r Role) []Group {
	if r == Public {
		return []Group{
			AirTemperature,
			RelativeHumidity,
			WindDirection,
			WindSpeedAvg,
			WindSpeedMax,
			ShortWaveRadiationOutgoing,
			PrecipitationTotal,
			SnowHeight,
		}
	}

	return GroupsByType(ParentGroup)
}

// AppendGroupIfMissing will append the given to group to the given slice if it
// is missing.
func AppendGroupIfMissing(slice []Group, g Group) []Group {
	for _, el := range slice {
		if el == g {
			return slice
		}
	}
	return append(slice, g)
}

// FilterGroupsByRole will filter the give groups by the given role returning
// only groups the role is allowed to access.
func FilterGroupsByRole(groups []Group, r Role) []Group {
	var filtered []Group

	for _, group := range groups {
		if present(group, GroupsByRole(r)) {
			filtered = append(filtered, group)
		}
	}

	return filtered
}

// present checks if the given group is present in the given slice of groups.
func present(g Group, groups []Group) bool {
	for _, group := range groups {
		if g == group {
			return true
		}
	}
	return false
}
