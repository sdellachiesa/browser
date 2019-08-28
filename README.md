# LTER Browser application

See [Design Document](https://gitlab.inf.unibz.it/lter/design/blob/master/browser.md).

## TODOs

* [x] ~~Accessible via web browser from external and internal.~~ [https://browser.lter.eurac.edu](https://browser.lter.eurac.edu)
* [ ] Implement ACL System with the restriction on
	* [ ] Stations
	* [ ] Parameter/Measurements
	* [ ] Time
	* [ ] Types
* [x] ~~Query and download *LTER data* in CSV file format.~~
* [x] ~~The timestamp in the output CSV file format should be in the format: `YYYY-MM-DD hh:mm:ss`~~
* [x] ~~Filter data on:~~
	* [x] ~~stations name (e.g. M4, M2, ...)~~
	* [x] ~~measurement name (e.g. Air relative humidity average, Wind speed average, ...)~~
	* [x] ~~land use~~
	* [x] ~~altitude~~
	* [x] ~~specific date range in the following format: `YYYY-MM-DD hh:mm:ss`~~
* [x] ~~The predefined parameters UI elements (dropdown) which are connected (e.g. station name and land use) will automatically be updated on change.~~
* [ ] Downsample data on a hourly, daily, monthly or yearly basis.
* [ ] Download *LTER metadata* as a separate file in one of the following formats: PDF, HTML, JSON
* [ ] Download code templates in R or Python of the defined user query.
* [x] ~~Interactive map showing all stations with the possibility to click on a station for getting additional information.~~
* [x] ~~Display data disclaimer and data license.~~
* [ ] View or download a glossary of the measured parameters.


