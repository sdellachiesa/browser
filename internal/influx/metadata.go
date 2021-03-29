// Copyright 2021 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package influx

// publicAllowed is the list of allowed measurements for the public role.
var publicAllowed = []string{
	"air_t_avg",
	"air_rh_avg",
	"wind_dir",
	"wind_speed",
	"wind_speed_avg",
	"wind_speed_max",
	"nr_up_sw_avg",
	"precip_rt_nrt_tot",
	"snow_height",
}

// maintenace is a list of measurement names only intressting for technicians.
var maintenace = []string{
	"RECORD",
	"Batt_V_Avg",
	"Batt_V_Std",
	"Log_T_Avg",
	"Log_T_Std",
	"Batt_V_Min",
	"Box_T_Ref_Avg",
	"Box_T_Ref_Std",
	"Air_T_old_Avg",
	"Air_T_old_Std",
	"Wind_Fail_AxisX_Tot",
	"Wind_Fail_AxisXY_Tot",
	"Wind_Fail_AxisY_Tot",
	"Wind_Fail_MaxGain_Tot",
	"Wind_Fail_NoNewData_Tot",
	"Wind_Fail_NVM_Tot",
	"Wind_Fail_ROM_Tot",
	"Wind_Samples_Tot",
	"NR_Dn_Body_T_Avg",
	"NR_Dn_Body_T_Std",
	"NR_Up_Body_T_Avg",
	"NR_Up_Body_T_Std",
	"NR_Dn_LW0_Avg",
	"NR_Dn_LW0_Std",
	"NR_Body_T_Avg",
	"NR_Body_T_Std",
	"NR_PT100_Rs_Avg",
	"NR_PT100_Rs_Std",
	"NR_Up_LW0_Avg",
	"NR_Up_LW0_Std",
	"SR_Old_Avg",
	"SR_Old_Std",
	"NDVI_Dn_Tilt_Avg",
	"NDVI_Up_Tilt_Avg",
	"NDVI_Dn_Tilt_1_Avg",
	"NDVI_Up_Tilt_1_Avg",
	"PRI_Dn_Tilt_Avg",
	"PRI_Up_Tilt_Avg",
	"PRI_Dn_Tilt_1_Avg",
	"PRI_Up_Tilt_1_Avg",
	"Precip_Bucket_Level_NRT_Lt",
	"Precip_Bucket_Level_NRT_Perc",
	"Precip_Bucket_Level_RT_Lt",
	"Precip_ElectronicUnit_T_Avg",
	"Precip_ElectronicUnit_T_Std",
	"Precip_Heater_Status",
	"Precip_LoadCell_T_Avg",
	"Precip_LoadCell_T_Std",
	"Precip_NRT_Cum",
	"Precip_Pluvio_Status",
	"Precip_Pluvio_V_Avg",
	"Precip_Pluvio_V_Std",
	"Precip_Rim_T_Avg",
	"Precip_Rim_T_Std",
	"Snow_Dist",
	"Snow_Dist_Std",
	"Snow_Dist0",
	"Snow_Dist0_Std",
	"Snow_Quality",
	"Snow_Quality_Std",
	"Soil_Surf_T_mV_Avg",
	"Soil_Surf_T_mV_Std",
	"Soil_Surf_Body_T_Avg",
	"Soil_Surf_Body_T_Std",
	"Soil_Surf_T0_Avg",
	"Soil_Surf_T0_Std",
	"SWC_Wave_PA_02_Avg",
	"SWC_Wave_PA_02_Std",
	"SWC_Wave_PA_05_Avg",
	"SWC_Wave_PA_05_Std",
	"SWC_Wave_PA_20_Avg",
	"SWC_Wave_PA_20_Std",
	"SWC_Wave_PA_50_Avg",
	"SWC_Wave_PA_50_Std",
	"SWC_Wave_PA_05_1_Avg",
	"SWC_Wave_PA_05_1_Std",
	"SWC_Wave_PA_40_Avg",
	"SWC_Wave_PA_40_Std",
	"SWC_Wave_VR_02_Avg",
	"SWC_Wave_VR_02_Std",
	"SWC_Wave_VR_05_Avg",
	"SWC_Wave_VR_05_Std",
	"SWC_Wave_VR_20_Avg",
	"SWC_Wave_VR_20_Std",
	"SWC_Wave_VR_50_Avg",
	"SWC_Wave_VR_50_Std",
	"SWC_Wave_VR_05_1_Avg",
	"SWC_Wave_VR_05_1_Std",
	"SWC_Wave_VR_40_Avg",
	"SWC_Wave_VR_40_Std",
	"SWC_Wave_PA_A_02_Avg",
	"SWC_Wave_PA_A_02_Std",
	"SWC_Wave_PA_A_05_Avg",
	"SWC_Wave_PA_A_05_Std",
	"SWC_Wave_PA_A_20_Avg",
	"SWC_Wave_PA_A_20_Std",
	"SWC_Wave_PA_A_50_Avg",
	"SWC_Wave_PA_A_50_Std",
	"SWC_Wave_PA_B_02_Avg",
	"SWC_Wave_PA_B_02_Std",
	"SWC_Wave_PA_B_05_Avg",
	"SWC_Wave_PA_B_05_Std",
	"SWC_Wave_PA_B_20_Avg",
	"SWC_Wave_PA_B_20_Std",
	"SWC_Wave_PA_C_02_Avg",
	"SWC_Wave_PA_C_02_Std",
	"SWC_Wave_PA_C_05_Avg",
	"SWC_Wave_PA_C_05_Std",
	"SWC_Wave_PA_C_20_Avg",
	"SWC_Wave_PA_C_20_Std",
	"SWC_Wave_PA_C_50_Avg",
	"SWC_Wave_PA_C_50_Std",
	"SWC_Wave_VR_A_02_Avg",
	"SWC_Wave_VR_A_02_Std",
	"SWC_Wave_VR_A_05_Avg",
	"SWC_Wave_VR_A_05_Std",
	"SWC_Wave_VR_A_20_Avg",
	"SWC_Wave_VR_A_20_Std",
	"SWC_Wave_VR_A_50_Avg",
	"SWC_Wave_VR_A_50_Std",
	"SWC_Wave_VR_B_02_Avg",
	"SWC_Wave_VR_B_02_Std",
	"SWC_Wave_VR_B_05_Avg",
	"SWC_Wave_VR_B_05_Std",
	"SWC_Wave_VR_B_20_Avg",
	"SWC_Wave_VR_B_20_Std",
	"SWC_Wave_VR_C_02_Avg",
	"SWC_Wave_VR_C_02_Std",
	"SWC_Wave_VR_C_05_Avg",
	"SWC_Wave_VR_C_05_Std",
	"SWC_Wave_VR_C_20_Avg",
	"SWC_Wave_VR_C_20_Std",
	"SWC_Wave_VR_C_50_Avg",
	"SWC_Wave_VR_C_50_Std",
	"LWmV_Max",
	"LWmV_Min",
	"LWmV_Std",
}
