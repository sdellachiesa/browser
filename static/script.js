// Copyright 2019 Eurac Research. All rights reserved.

// errorHandler returns an XHR error callback that invokes the given
// browser error callback with the human-readable error string.
function errorHandler(callback) {
	return function(jqXHR, textStatus, errorThrown) {
		console.log(textStatus, errorThrown);
		if (errorThrown) {
			callback(errorThrown);
			return;
		}
		callback(textStatus);
	}
}

// TODO(pam): display errors in a more friendly way.
function error(err) {
	alert(err)
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

// Landuse maps the landuse key to its full description.
function Landuse(key) {
	switch(key) {
		case "pa":
			return "Pasture"
		case "me":
			return "Meadow"
		case "fo":
			return "Climate station in the forest"
		case "sf":
			return "SapFlow"
		case "de":
			return "Dendrometer"
		case "ro":
			return "Rock"
		case "bs":
			return "Bare soil"
	}
}

// Download enables the download botton if at least one
// station and one measurement was selected.
function Download(stationEl, fieldEl, submitEl) {
	if ($(stationEl).val() == null) {
		$(submitEl).attr("disabled", "disabled");
		return
	}

	if ($(fieldEl).val() == null) {
		$(submitEl).attr("disabled", "disabled");
		return
	}

	$(submitEl).removeAttr("disabled");
}

// Option checks adds an option html item to the given element.
function Option(el, data) {
	$(el).children('option').map(function(){
		if (data.includes(this.value)) {
			//console.log("included: " + this.value)
			$(this).prop('disabled', false);
		} else {
			//console.log("not included: " + this.value)
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
//	metaEL - metadata element
//	submitEl - submit button element
//	mapEl - map element
function browser(opts) {
	$(opts.fieldEl).multiselect({
		maxHeight: 400,
		buttonWidth: "100%",
		enableFiltering: true,
		onChange: function() {
			$.ajax("/api/v1/update", {
				method: "POST",
				data: JSON.stringify({
					//stations: $(opts.stationEl).val(),
					//landuse: $(opts.landuseEl).val(),
					fields: $(opts.fieldEl).val(),
				}),	
				dataType: "json",
				success: function(data) {
					Option(opts.stationEl, data.Stations);
					Option(opts.landuseEl, data.Landuse);
					Download(opts.stationEl, opts.fieldEl, opts.submitEl);
				}
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
					stations: $(opts.stationEl).val(),
				}),	
				dataType: "json",
				success: function(data) {
					Option(opts.fieldEl, data.Fields);
					Option(opts.landuseEl, data.Landuse);
					Download(opts.stationEl, opts.fieldEl, opts.submitEl);
				}
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
				//	stations: $(opts.stationEl).val(),
					landuse: $(opts.landuseEl).val(),
				//	fields: $(opts.fieldEl).val(),
				}),	
				dataType: "json",
				success: function(data) {
					Option(opts.fieldEl, data.Fields);
					Option(opts.stationEl, data.Stations);
					Download(opts.stationEl, opts.fieldEl, opts.submitEl);
				}
			});
		}
	});

	$(opts.dateEl).datepicker({
		todayHighlight: true,
		endDate: new Date(),
		format: 'yyyy-mm-dd',
	});

	var endDate = new Date()
	var startDate = new Date(new Date().setMonth(new Date().getMonth()-6))
	SetDefaultDate(opts.sDateEl, startDate)
	SetDefaultDate(opts.eDateEl, endDate)

	$(".js-range-slider").ionRangeSlider({
		skin: "round",
		type: "double",
		min: 900,
		max: 2500,
		grid: true
	 });

	// Initalize map.
	var map = L.map(opts.mapEl, {zoomControl: false}).setView([46.69765764825818, 10.638368502259254], 13);
	L.control.scale({position: "bottomright"}).addTo(map);
	L.control.zoom({position: "bottomright"}).addTo(map);
	L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png?', {
		attribution: 'Map data &copy; <a href="https://www.openstreetmap.org/">OpenStreetMap</a> contributors, <a href="https://creativecommons.org/licenses/by-sa/2.0/">CC-BY-SA</a>'
	}).addTo(map);
	
	var mapBound = [];
	opts.mapData.forEach(function(item) {
		var marker = L.marker([item.Latitude, item.Longitude]).addTo(map);
		marker.bindPopup(`<div id="${item.Name}mappopup">
		<p>
		<b>Name:</b>  ${item.Name}<br>
		<b>Altitude:</b> ${item.Altitude} m
		</p>
		</div>`);
		mapBound.push(new L.latLng(item.Latitude, item.Longitude));
	});	
	map.fitBounds(mapBound);
	//$.ajax("/api/v1/stations", {
	//	method: "GET",
	//	dataType: "json",
	//	success: function(data) {
	//		
	//	},
	//	error: errorHandler(error)
	//});
}