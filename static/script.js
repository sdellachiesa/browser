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
	$.fn.selectpicker.Constructor.DEFAULTS.styleBase = "btn-sm";

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
	
	// initialize station,landuse and map points.
	//$.ajax("/api/v1/stations/", {
	//	method: "POST",
	//	dataType: "json",
	//	success: function(data) {
	//		var mapBound = [];
	//		console.log(data)
	//		Object.keys(data).map(function(objectKey, index) {
	//			var v = data[objectKey];
	//		
	//			// station dropdown
	//			AddOption(opts.stationEl, v.Name, v.Name);

	//			// landuse dropdown
	//			AddOption(opts.landuseEl, v.Landuse, Landuse(v.Landuse));

	//			var marker = L.marker([v.Latitude, v.Longitude]).addTo(map);
	//			marker.bindPopup(`<div id="${v.Name}mappopup">
	//			<p>
	//			<b>Name:</b>  ${v.Name}<br>
	//			<b>Altitude:</b> ${v.Altitude} m
	//			</p>
	//			</div>`);
	//			mapBound.push(new L.latLng(v.Latitude, v.Longitude));
	//		});
	//		
	//		map.fitBounds(mapBound);
	//	},
	//	error: errorHandler(error)
	//});

	// initialize fields
	//$.ajax("/api/v1/fields/", {
	//	method: "POST",
	//	dataType: "json",
	//	success: function(data) {
			
	//		data.forEach(function(item) {
	//			// field dropdown
	//			AddOption(opts.fieldEl, item, item);
	//		});
	///	},
	//	error: errorHandler(error)
	//});

	// Handle field update event: update stations and landuse.
	$(opts.fieldEl).on('changed.bs.select', function(){
		console.log("field changed.")
		$.ajax("/api/v1/stations/", {
			method: "POST",
			data: JSON.stringify({
				fields: $(opts.fieldEl).val(),
			}),
			dataType: "json",
			success: function(data) {
				var selStations = $(opts.stationEl).val();
				var selLanduse = $(opts.landuseEl).val();

				$(opts.stationEl).find('option').remove();
				$(opts.landuseEl).find('option').remove();
				
				Object.keys(data).map(function(objectKey, index) {
					var v = data[objectKey];
					
					AddOption(opts.stationEl, v.Name, v.Name, selStations);
					AddOption(opts.landuse, v.Landuse, Landuse(v.Landuse), selLanduse);
				});
			},
			error: errorHandler(error)
		});
		Download(opts.stationEl, opts.fieldEl, opts.submitEl);
	});

	// Handle station update event: update fields and landuse.
	$(opts.stationEl).on('changed.bs.select', function(){
		// update fields
		$.ajax("/api/v1/fields/", {
			method: "POST",
			data: JSON.stringify({
				stations: $(opts.stationEl).val(),
			}),
			dataType: "json",
			success: function(data) {
				var selFields = $(opts.fieldEl).val();
				$(opts.fieldEl).find('option').remove();
				data.forEach(function(item){
					AddOption(opts.fieldEl, item, item, selFields);
				});
				
			},
			error: errorHandler(error)
		});

		// update landuse
		$.ajax("/api/v1/stations/", {
			method: "POST",
			data: JSON.stringify({
				stations: $(opts.stationEl).val(),
			}),
			dataType: "json",
			success: function(data) {
				var selLanduse = $(opts.landuseEl).val();
				$(opts.landuseEl).find('option').remove();
				Object.keys(data).map(function(objectKey, index) {
					var v = data[objectKey];
					AddOption(opts.landuseEl, v.Landuse, Landuse(v.Landuse), selLanduse);
				});
			},
			error: errorHandler(error)
		});
		Download(opts.stationEl, opts.fieldEl, opts.submitEl);
	});

	// Handle landuse update event: update fields and stations.
	$(opts.landuseEl).on('changed.bs.select', function(){
		$.ajax("/api/v1/stations/", {
			method: "POST",
			data: JSON.stringify({
				landuse: $(opts.landuseEl).val(),
			}),
			dataType: "json",
			success: function(data) {
				var selStations = $(opts.stationEl).val();
				var selFields = $(opts.fieldEl).val();
				
				$(opts.fieldEl).find('option').remove();
				$(opts.stationEl).find('option').remove();

				Object.keys(data).map(function(objectKey, index) {
					var v = data[objectKey];

					v.Measurements.forEach(function(item) {
						AddOption(opts.fieldEl, item, item, selFields);
					});

					AddOption(opts.stationEl, v.Name, v.Name, selStations);
				});
			},
			error: errorHandler(error)
		});
		Download(opts.stationEl, opts.fieldEl, opts.submitEl);
	});

}

// AddOption adds an option html item to the given element.
function AddOption(el, value, text, currentSelected) {
	// If option already exists do not add it.
	if ($(el +" option[value='"+value+"']").length > 0) {
		return
	}

	var selected = "";
	if (currentSelected != null && currentSelected.includes(value)) {
		$(el).append('<option value="'+value+'" selected>'+text+'</option>');
		$(el).selectpicker('refresh');
		return
	}

	$(el).append('<option value="'+value+'">'+text+'</option>');
	$(el).selectpicker('refresh');
	return
}
