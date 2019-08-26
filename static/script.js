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

// FormatDate formats the given date to yyyy-mm-dd.
function FormatDate(date) {
	return date.toISOString().slice(0, 10)
}

// SetDefaultDate sets the give dat value on the given
// element.
function SetDefaultDate(el, date) {
	$(el).val(FormatDate(date))
	$(el).datepicker().on("hide", function() {
		if ($(this).val() != "") {
			return
		}
		$(this).val(FormatDate(date))
	});
}

// ToggleDownload enables the download botton if at least one
// station and one measurement was selected. Moreover it checks
// if the time range selected is not ore than a year. Otherwise 
// it will be disable it.
function ToggleDownload(opts) {
	var startDate = new Date($(opts.sDateEl).val())
	var maxDate = new Date(new Date().setFullYear(new Date().getFullYear()-1));
	
	if (startDate < maxDate) {
		$(opts.submitEl).attr("disabled", "disabled");
		return
	}
	
	if ($(opts.stationEl).val() == null) {
		$(opts.submitEl).attr("disabled", "disabled");
		return
	}

	if ($(opts.fieldEl).val() == null) {
		$(opts.submitEl).attr("disabled", "disabled");
		return
	}

	$(opts.submitEl).removeAttr("disabled");
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

// ToggleOptionsForNumbers enables/disables an option depending
// on the presents of its value in given int data array.
function ToggleOptionsForNumbers(el, data) {
	$(el).children('option').map(function(){
		var v = Number(this.value)
		if (data.includes(v)) {
			$(this).prop('disabled', false);
		} else {
			$(this).prop('disabled', true);
			$(this).prop('selected', false);
		}
	});
	
	$(el).multiselect('refresh');
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
//  tooltipWrapperEl - tooltip wrapper element for submit button
//	mapEl - map element
/// mapData - JSON used for initialize the map and altitude range
function browser(opts) {
	$(opts.fieldEl).multiselect({
		maxHeight: 400,
		buttonWidth: "100%",
		enableFiltering: true,
		onChange: function() {
			$.ajax("/api/v1/update", {
				method: "POST",
				data: JSON.stringify({
					stations: $(opts.stationEl).val(),
					landuse: $(opts.landuseEl).val(),
					fields: $(opts.fieldEl).val(),
				}),	
				dataType: "json",
				success: function(data) {
					ToggleOptionsForNumbers(opts.stationEl, data.Stations);
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
		onChange: function() {
			$.ajax("/api/v1/update", {
				method: "POST",
				data: JSON.stringify({
					//landuse: $(opts.landuseEl).val(),
					stations: $(opts.stationEl).val(),
					//fields: $(opts.fieldEl).val(),
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
		onChange: function() {
			$.ajax("/api/v1/update", {
				method: "POST",
				data: JSON.stringify({
					//stations: $(opts.stationEl).val(),
					landuse: $(opts.landuseEl).val(),
					//fields: $(opts.fieldEl).val(),
				}),	
				dataType: "json",
				success: function(data) {
					ToggleOptions(opts.fieldEl, data.Fields);
					ToggleOptionsForNumbers(opts.stationEl, data.Stations);
					ToggleDownload(opts);
				},
				error: errorHandler(error)
			});
		}
	});


	var endDate = new Date()
	var startDate = new Date(new Date().setMonth(new Date().getMonth()-6));
	var maxDate = new Date(new Date().setFullYear(new Date().getFullYear()-1));
	
	$(opts.dateEl).datepicker({
		todayHighlight: true,
		format: 'yyyy-mm-dd',
		endDate: endDate
	}).on("changeDate", function(){
		var startDate = new Date($(opts.sDateEl).val())
		var maxDate = new Date(new Date().setFullYear(new Date().getFullYear()-1));

		if (startDate < maxDate) {
			$(opts.sDateEl).popover({
				placement: 'top',
				trigger: 'manual',
			});
			$(opts.sDateEl).popover('show');
		} else {
			$(opts.sDateEl).popover('hide');
		}
		ToggleDownload(opts);
	});
	SetDefaultDate(opts.sDateEl, startDate)
	SetDefaultDate(opts.eDateEl, endDate)

	// Initalize map.
	var map = L.map(opts.mapEl, {zoomControl: false}).setView([46.69765764825818, 10.638368502259254], 13);
	L.control.scale({position: "bottomright"}).addTo(map);
	L.control.zoom({position: "bottomright"}).addTo(map);
	L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png?', {
		attribution: 'Map data &copy; <a href="https://www.openstreetmap.org/">OpenStreetMap</a> contributors, <a href="https://creativecommons.org/licenses/by-sa/2.0/">CC-BY-SA</a>'
	}).addTo(map);
	
	var mapMarkers = {};
	var mapBound = [];
	Object.keys(opts.mapData).map(function(k) {
		var item = opts.mapData[k];
	
		var marker = L.marker([item.Latitude, item.Longitude]).addTo(map);
		marker.bindPopup(`<div id="${item.Name}mappopup">
		<p>
		<b>Name:</b>  ${item.Name}<br>
		<b>Altitude:</b> ${item.Altitude} m
		</p>
		</div>`);
	
		mapMarkers[item.ID] = marker
		mapBound.push(new L.latLng(item.Latitude, item.Longitude));
	});	
	map.fitBounds(mapBound);

	$(".js-range-slider").ionRangeSlider({
		skin: "round",
		type: "double",
		min: 900,
		max: 2500,
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
			ToggleOptionsForNumbers(opts.stationEl, stations);
			ToggleOptions(opts.landuseEl, landuse);
		}
	 });
}