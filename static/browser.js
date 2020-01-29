// Copyright 2019 Eurac Research. All rights reserved.

// browser sets up the client for filtering and downloading LTER data.
// opts is an object with these keys data - JSON data object
//	stationEl - stations select element
//	measurementEl - measurements select element
//	landuseEl - landuse select element
//	altitudeEl - altitude range element
//	dateEl - date picker element
//	sDateEl - start date element
//	eDateEl - end date element
//	submitEl - submit button element
//	codeEl - code button element
//	mapEl - map element
function browser(opts) {
	const mapMarkers = {};

	function getMaxAltitude() {
		let a = 0;

		opts.data.forEach(function(s) {
			if (s.Altitude >= a) {
				a = s.Altitude
			}
		});

		return Math.round(a/1000)*1000;
	}

	function getMinAltitude() {
		let a = 10000;

		opts.data.forEach(function(s) {
			if (s.Altitude <= a) {
				a = s.Altitude
			}
		});

		return Math.floor(a/100)*100;
	}

	function loadMap() {
		const map = L.map(opts.mapEl, {zoomControl: false}).setView([46.69765764825818, 10.638368502259254], 13);

		L.control.scale({position: "bottomright"}).addTo(map);
		L.control.zoom({position: "bottomright"}).addTo(map);

		const basemap = {
		"Orthophotos South Tyrol (2014/2015/2017)": L.tileLayer.wms('http://geoservices.retecivica.bz.it/geoserver/ows?', {
			layers: 'P_BZ_OF_2014_2015_2017',
			attribution: 'Map data &copy; <a href="http://geoportal.buergernetz.bz.it/geodaten.asp">Geoportal Südtirol</a>, <a href="https://creativecommons.org/publicdomain/zero/1.0/deed.en">CC-0</a>'
		}).addTo(map),

		"Open Street Map": L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png?', {
		attribution: 'Map data &copy; <a href="https://www.openstreetmap.org/">OpenStreetMap</a> contributors, <a href="https://creativecommons.org/licenses/by-sa/2.0/">CC-BY-SA</a>'}),

		"OpenTopoMap": L.tileLayer("https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png", {
		attribution: 'Kartendaten: &copy; <a href="https://openstreetmap.org/copyright">OpenStreetMap</a>-Mitwirkende, SRTM | Kartendarstellung: &copy; <a href="http://opentopomap.org/">OpenTopoMap<a> (<a href="https://creativecommons.org/licenses/by-sa/3.0/">CC-BY-SA</a>)'}),

		"Google Maps Terrain": L.tileLayer('http://{s}.google.com/vt/lyrs=p&x={x}&y={y}&z={z}',{
			maxZoom: 20,
			subdomains:['mt0','mt1','mt2','mt3'],
			attribution: 'Map data &copy; Google.'})
		}

		// Add map layers.
		L.control.layers(basemap, {}, {'collapsed': false}).addTo(map)

		const mapBound = [];
		Object.keys(opts.data).map(function(k) {
			let item = opts.data[k];

			let marker = L.marker([item.Latitude, item.Longitude]).addTo(map);
			let c = document.getElementById("s"+item.ID)

			marker.bindPopup(c, {
			autoPan: true,
			keepInView: true,
			maxWidth: 600});

			marker.bindTooltip(c.getAttribute("data-name"))

			mapMarkers[item.ID] = marker
			mapBound.push(new L.latLng(item.Latitude, item.Longitude));
		});
	}

	// isValidDateRange checks if the selected date range is valid and
	// returns true for a valid range otherwise false.
	function isValidDateRange() {
		var startDate = new Date($(opts.sDateEl).val());
		startDate.setHours(0,0,0,0);

		var endDate = new Date($(opts.eDateEl).val());
		endDate.setFullYear(endDate.getFullYear() - 1);
		endDate.setHours(0,0,0,0);

		if (startDate < endDate) {
			return false;
		}

		return true;
	}

	// toggleDownload enables the download botton if at least one
	// station and one measurement was selected. Moreover it checks
	// if the time range selected is not ore than a year. Otherwise
	// it will be disable it.
	function toggleDownload() {
		if ($(opts.sDateEl).val() == "" || $(opts.eDateEl).val() == "") {
			$(opts.submitEl).attr("disabled", "disabled");
			return
		}

		if (! isValidDateRange()) {
			$(opts.submitEl).attr("disabled", "disabled");
			return
		}

		if ($(opts.stationEl).val() == null) {
			$(opts.submitEl).attr("disabled", "disabled");
			$(opts.codeEl).attr("disabled", "disabled");
			return
		}

		if ($(opts.measurementEl).val() == null) {
			$(opts.submitEl).attr("disabled", "disabled");
			$(opts.codeEl).attr("disabled", "disabled");
			return
		}

		$(opts.submitEl).removeAttr("disabled");
		$(opts.codeEl).removeAttr("disabled");
	}

	// ToggleOptions enables or disables an option.
	// arr is an array of objects with these keys:
	// 	[{ el: "id of select", data: "set of items"}]
	function toggleOptions(arr) {
		arr.forEach(function(item) {
			$(item.el).children('option').map(function() {
				if (item.data.size === 0) {
					$(this).prop('disabled', false);
					return
				}
				if (item.data.has(this.value)) {
					$(this).prop('disabled', false);
					return
				}

				$(this).prop('disabled', true);
				$(this).prop('selected', false);
			});

			$(item.el).multiselect('refresh')
		});
	}

	// filterByMeasurements filters the global opts.data object by the
	// given measurements and returns an object with the following keys:
	// 	stations - set of stations
	//	landuse - set of landuses
	function filterByMeasurements(names) {
		const stations = new Set();
		const landuse = new Set();

		if (! Array.isArray(names)) {
				return {stations, landuse}
		}

		opts.data.forEach(function(o) {
			if (! Array.isArray(o.Measurements)) {
				return {stations, landuse}
			}

			let result = names.every(function(val) {
				return o.Measurements.indexOf(val) >= 0
			});
			if (result) {
				stations.add(o.ID);
				landuse.add(o.Landuse);
			}
		});

		return {stations, landuse};
	}

	// filterByStations filters the global opts.data object by the given
	// stations and returns an object with the following keys:
	// 	measurements - set of measurements
	//	landuse - set of landuses
	function filterByStations(stations) {
		const measurements = new Set();
		const landuse = new Set();

		if (! Array.isArray(stations)) {
			return {measurements, landuse};
		}

		opts.data.forEach(function(o) {
			if (stations.indexOf(o.ID) >= 0) {
				if (Array.isArray(o.Measurements)) {
					o.Measurements.forEach(m => measurements.add(m));
				}
				landuse.add(o.Landuse);
			}
		});

		return {measurements, landuse};
	}

	// filterByLanduse filters the global opts.data object by the given
	// landuses and returns an object with the following keys:
	// 	measurements - set of measurements
	//	stations - set of stations
	function filterByLanduse(landuse) {
		const measurements = new Set();
		const stations = new Set();

		if (! Array.isArray(landuse)) {
			return {measurements, stations};
		}

		opts.data.forEach(function(o) {
			if (landuse.indexOf(o.Landuse) >= 0) {
				if (Array.isArray(o.Measurements)) {
					o.Measurements.forEach(m => measurements.add(m));
				}
				stations.add(o.ID);
			}
		});

		return {measurements, stations};
	}


	// Initialize UI elements

	$(opts.measurementEl).multiselect({
		maxHeight: 400,
		buttonWidth: "100%",
		enableFiltering: true,
		filterBehavior: "both",
		enableRegexFiltering: true,
		enableCaseInsensitiveFiltering: true,
		includeSelectAllOption: true,
		onChange: function() {
			const m = filterByMeasurements($(opts.measurementEl).val());

			toggleOptions([
				{el: opts.stationEl, data: m.stations},
				{el: opts.landuseEl, data: m.landuse}
			]);
			toggleDownload();
		}
	});

	$(opts.stationEl).multiselect({
		maxHeight: 400,
	 	buttonWidth: "100%",
		enableFiltering: true,
		filterBehavior: "both",
		enableRegexFiltering: true,
		enableFullValueFiltering: true,
		enableCaseInsensitiveFiltering: true,
		includeSelectAllOption: true,
		onChange: function() {
			const s = filterByStations($(opts.stationEl).val());

			toggleOptions([
				{el: opts.measurementEl, data: s.measurements},
				{el: opts.landuseEl, data: s.landuse}
			]);
			toggleDownload();

		}
	});

	$(opts.landuseEl).multiselect({
		maxHeight: 400,
		buttonWidth: "100%",
		enableFiltering: true,
		filterBehavior: "both",
		enableRegexFiltering: true,
		enableFullValueFiltering: true,
		enableCaseInsensitiveFiltering: true,
		includeSelectAllOption: true,
		onChange: function() {
			const l = filterByLanduse($(opts.landuseEl).val());

			toggleOptions([
				{el: opts.measurementEl, data: l.measurements},
				{el: opts.stationEl, data: l.stations}
			]);
			toggleDownload();
		}
	});

	$(opts.altitudeEl).ionRangeSlider({
		skin: "round",
		type: "double",
		min: getMinAltitude(),
		max: getMaxAltitude(),
		grid: true,
		onChange: function(data) {
			const stations = new Set();
			const landuse = new Set();
			const measurements = new Set();

			opts.data.forEach(function(item) {
				var marker = mapMarkers[item.ID]
				if (item.Altitude >= data.from && item.Altitude <= data.to) {
					stations.add(item.ID);
					landuse.add(item.Landuse);
					if (Array.isArray(item.Measurements)) {
						item.Measurements.forEach(m => measurements.add(m));
					}

					marker.setOpacity(1.0);
				} else {
					marker.setOpacity(0.4);
				}
			});

			toggleOptions([
				{el: opts.measurementEl, data: measurements},
				{el: opts.stationEl, data: stations},
				{el: opts.landuse, data: landuse},
			]);
		}
	 });


	$(opts.dateEl).datepicker({
		todayHighlight: true,
		format: 'yyyy-mm-dd',
		endDate: new Date()
	}).on('hide', function() {
		toggleDownload();

		if ($(opts.sDateEl).val() == "" || $(opts.eDateEl).val() == "") {
			$(opts.dateEl).popover('show')
			return
		}

		if (isValidDateRange()) {
			$(opts.sDateEl).popover('hide');

			return
		}

		$(opts.sDateEl).popover('show');
	}).on('show', function() {
		$(opts.dateEl).popover('hide');
	});

	loadMap();
}
