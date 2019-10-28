// Copyright 2019 Eurac Research. All rights reserved.

// errorHandler returns an XHR error callback that invokes the given
// browser error callback with the human-readable error string.
function errorHandler(callback) {
	return function(jqXHR, textStatus, errorThrown) {
		if (errorThrown) {
			callback(errorThrown);
			return;
		}
		callback(textStatus);
	}
}

// error displays an modal dialog with the given error.
function error(err) {
	$("#errorDialogContent").html("<p>" + err + "<p>");
	$("#errorDialog").modal('toggle');
}

// ToggleDownload enables the download botton if at least one
// station and one measurement was selected. Moreover it checks
// if the time range selected is not ore than a year. Otherwise
// it will be disable it.
function ToggleDownload(opts) {
	if ($(opts.sDateEl).val() == "" || $(opts.eDateEl).val() == "") {
		$(opts.submitEl).attr("disabled", "disabled");
		return
	}

	if (! ValidDateRange(opts.sDateEl, opts.eDateEl)) {
		$(opts.submitEl).attr("disabled", "disabled");
		return
	}

	if ($(opts.stationEl).val() == null) {
		$(opts.submitEl).attr("disabled", "disabled");
		$(opts.codeEl).attr("disabled", "disabled");
		return
	}

	if ($(opts.fieldEl).val() == null) {
		$(opts.submitEl).attr("disabled", "disabled");
		$(opts.codeEl).attr("disabled", "disabled");
		return
	}

	$(opts.submitEl).removeAttr("disabled");
	$(opts.codeEl).removeAttr("disabled");
}

// ToggleOptions enables/disables an option depending on the presents
// of its value in given string data array.
function ToggleOptions(el, data) {
	$(el).children('option').map(function(){
		if (data.includes(this.value)) {
			$(this).prop('disabled', false);
		} else {
			$(this).prop('disabled', true);
			$(this).prop('selected', false);
		}
	});

	$(el).multiselect('refresh');
}

// ValidDateRange checks if the selected date range is valid and
// returns true for a valid range otherwise false.
function ValidDateRange(sDateEl, eDateEl) {
	var startDate = new Date($(sDateEl).val());
	startDate.setHours(0,0,0,0);

	var endDate = new Date($(eDateEl).val());
	endDate.setFullYear(endDate.getFullYear() - 1);
	endDate.setHours(0,0,0,0);

	if (startDate < endDate) {
		return false;
	}

	return true;
}

// browser sets up the client for filtering and
// downloading data.
// opts is an object with these keys
//	stationEl - stations select element
//	fieldEl - fields select element
//	landuseEl - landuse select element
//	altitudeEl - altitude range element
//	dateEl - date picker element
//	sDateEl - start date element
//	eDateEl - end date element
//	submitEl - submit button element
//    codeEl - code button element
//	mapEl - map element
/// mapData - JSON used for initialize the map and altitude range
function browser(opts) {
	$(opts.fieldEl).multiselect({
		maxHeight: 400,
		buttonWidth: "100%",
		enableFiltering: true,
		includeSelectAllOption: true,
		onChange: function() {
			$.ajax("/api/v1/filter", {
				method: "POST",
				contentType: "application/json",
				data: JSON.stringify({
					fields: $(opts.fieldEl).val(),
				}),
				dataType: "json",
				success: function(data) {
					ToggleOptions(opts.stationEl, data.Stations);
					ToggleOptions(opts.landuseEl, data.Landuse);
					ToggleDownload(opts);
				},
				error: errorHandler(error)
			});
		}
	});

	$(opts.stationEl).multiselect({
		maxHeight: 400,
	 	buttonWidth: "100%",
		enableFiltering: true,
		includeSelectAllOption: true,
		onChange: function() {
			$.ajax("/api/v1/filter", {
				method: "POST",
				contentType: "application/json",
				data: JSON.stringify({
					stations: $(opts.stationEl).val(),
				}),
				dataType: "json",
				success: function(data) {
					ToggleOptions(opts.fieldEl, data.Fields);
					ToggleOptions(opts.landuseEl, data.Landuse);
					ToggleDownload(opts);
				},
				error: errorHandler(error)
			});
		}
	});

	$(opts.landuseEl).multiselect({
		maxHeight: 400,
		buttonWidth: "100%",
		enableFiltering: true,
		includeSelectAllOption: true,
		onChange: function() {
			$.ajax("/api/v1/filter", {
				method: "POST",
				contentType: "application/json",
				data: JSON.stringify({
					landuse: $(opts.landuseEl).val(),
				}),
				dataType: "json",
				success: function(data) {
					ToggleOptions(opts.fieldEl, data.Fields);
					ToggleOptions(opts.stationEl, data.Stations);
					ToggleDownload(opts);
				},
				error: errorHandler(error)
			});
		}
	});

	$(opts.dateEl).datepicker({
		todayHighlight: true,
		format: 'yyyy-mm-dd',
		endDate: new Date()
	}).on('changeDate', function() {
		ToggleDownload(opts);
		if (ValidDateRange(opts.sDateEl, opts.eDateEl)) {
			$(opts.sDateEl).popover('hide');
			return
		}
		$(opts.sDateEl).popover('show');
	}).on('hide', function() {
		if ($(opts.sDateEl).val() == "" || $(opts.eDateEl).val() == "") {
			$(opts.dateEl).popover('show')
		} else {
			$(opts.dateEl).popover('hide')
		}

		ToggleDownload(opts);
		if (ValidDateRange(opts.sDateEl, opts.eDateEl)) {
			$(opts.sDateEl).popover('hide');
			return
		}
		$(opts.sDateEl).popover('show');
	}).on('show', function() {
		$(opts.dateEl).popover('hide');
	});

	// Initalize map.
	var map = L.map(opts.mapEl, {zoomControl: false}).setView([46.69765764825818, 10.638368502259254], 13);

	L.control.scale({position: "bottomright"}).addTo(map);
	L.control.zoom({position: "bottomright"}).addTo(map);

	var basemap = {
		"Open Street Map": L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png?', {
		attribution: 'Map data &copy; <a href="https://www.openstreetmap.org/">OpenStreetMap</a> contributors, <a href="https://creativecommons.org/licenses/by-sa/2.0/">CC-BY-SA</a>'}).addTo(map),

		"Orthophotos South Tyrol (2014/2015/2017)": L.tileLayer.wms('http://geoservices.retecivica.bz.it/geoserver/ows?', {
			layers: 'P_BZ_OF_2014_2015_2017',
			attribution: 'Map data &copy; <a href="http://geoportal.buergernetz.bz.it/geodaten.asp">Geoportal Südtirol</a>, <a href="https://creativecommons.org/publicdomain/zero/1.0/deed.en">CC-0</a>'
		}),

		"Google Maps Terrain": L.tileLayer('http://{s}.google.com/vt/lyrs=p&x={x}&y={y}&z={z}',{
			maxZoom: 20,
			subdomains:['mt0','mt1','mt2','mt3'],
			attribution: 'Map data &copy; Google.'})
	}

	// Add map layers.
	L.control.layers(basemap, {}, {'collapsed': false}).addTo(map)

	var mapMarkers = {};
	var mapBound = [];
	var maxAltitude = 0;
	var minAltitude = 100000;
	Object.keys(opts.mapData).map(function(k) {
		var item = opts.mapData[k];

		var marker = L.marker([item.Latitude, item.Longitude]).addTo(map);
		marker.bindPopup(`<p>
		<strong>Name:</strong> ${item.Name}<br>
		<strong>Altitude:</strong> ${item.Altitude} m
		</p>
		<p><img src="${item.Image}" width="400"></p>`, {
			autoPan: true,
			keepInView: true,
			maxWidth: 600,
			className: "map-popup"
		});

		mapMarkers[item.ID] = marker
		mapBound.push(new L.latLng(item.Latitude, item.Longitude));

		if (item.Altitude >= maxAltitude) {
			maxAltitude = item.Altitude
		}

		if (item.Altitude <= minAltitude) {
			minAltitude = item.Altitude
		}
	});
	map.fitBounds(mapBound, {padding: [50, 50]});

	$(".js-range-slider").ionRangeSlider({
		skin: "round",
		type: "double",
		min: Math.floor(minAltitude/100)*100,
		max: Math.round(maxAltitude/1000)*1000,
		grid: true,
		onChange: function(data) {
			var stations = [];
			var landuse = [];
			opts.mapData.forEach(function(item) {
				var marker = mapMarkers[item.ID]
				if (item.Altitude >= data.from && item.Altitude <= data.to) {
					stations.push(item.ID);
					landuse.push(item.Landuse);
					marker.setOpacity(1.0);
				} else {
					marker.setOpacity(0.4);
				}
			});
			ToggleOptions(opts.stationEl, stations);
			ToggleOptions(opts.landuseEl, landuse);
		}
	 });
}