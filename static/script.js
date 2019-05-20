
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
	var landuse = [
		{key: "pa", name: "Pasture"},
		{key: "me", name: "Meadow"},
		{key: "fo", name: "Climate station in the forest"},
		{key: "sf", name: "SapFlow"},
		{key: "de", name: "Dendrometer"},
		{key: "ro", name: "Rock"},
		{key: "bs", name: "Bare soil"}
	]

	$(opts.dateEl).datepicker({
		todayHighlight: true,
		format: 'yyyy-mm-dd',
	});

	$(".js-range-slider").ionRangeSlider({
		skin: "round",
      	  	type: "double",
       	 	min: 900,
        		max: 2500,
		grid: true
 	});

	var $elSelectStation = $(opts.stationEl).selectize({
		plugins: ['remove_button'],
    		delimiter: ',',
    		persist: false,
    		valueField: 'name',
    		labelField: 'name',
		searchField: 'name',
		create: false,
		onInitialize: function() {
			$.ajax("/_api", {
				method: "POST",
				data:  {
					method: "stations"
				},
				dataType: "json",
				success: function(data) {
					$elSelectStation[0].selectize.clearOptions();
					$elSelectStation[0].selectize.addOption(data);
				},
				error: errorHandler(error)
			});
		}
    	});

	var $elSelectLanduse = $(opts.landuseEl).selectize({
		plugins: ['remove_button'],
    		persist: false,
		create: false,
		valueField: 'key',
    		labelField: 'name',
		searchField: 'name',
		options: landuse
    	});

	var $elSelectFields = $(opts.fieldEl).selectize({
		plugins: ['remove_button'],
    		delimiter: ',',
    		persist: false,
		create: false,
		valueField: 'name',
    		labelField: 'name',
		searchField: 'name',
		onInitialize: function() {
			$.ajax("/_api", {
				method: "POST",
				data:  {
					method: "fields"
				},
				dataType: "json",
				success: function(data) {
					var fields = []
					data.forEach(function(v, i) {
						v.values.forEach(function(f, k){
							fields.push({name: f[0]})
						});
					});

					$elSelectFields[0].selectize.clearOptions();
					$elSelectFields[0].selectize.addOption(fields);
				},
				error: errorHandler(error)
			});
		}
    	});

	$(opts.stationEl).change(function() {
		console.log("stations change")
		$.ajax("/_api", {
				method: "POST",
				data:  {
					method: "fields",
					stations: $elSelectStation[0].selectize.getValue().join(",")
				},
				dataType: "json",
				success: function(data) {
					console.log(data)
					var fields = []
					data.forEach(function(v, i) {
						v.values.forEach(function(f, k){
							fields.push({name: f[0]})
						});
					});
				///	var control = $elSelectFields[0].selectize

					//control.getValue().forEach(function(item, index) {
				//		console.log($.grep(fields, function(obj) { return obj.name == item }))
				//	});

					$elSelectFields[0].selectize.addOption(fields);
				},
				error: errorHandler(error)
			});
	});

	$(opts.fieldEl).change(function() {
		console.log("field change")
		$.ajax("/_api", {
				method: "POST",
				data:  {
					method: "stations",
					fields: $elSelectFields[0].selectize.getValue().join(",")
				},
				dataType: "json",
				success: function(data) {
					//$elSelectStation[0].selectize.clearOptions();
					$elSelectStation[0].selectize.addOption(data);
				},
				error: errorHandler(error)
			});
	});


	$(opts.submitEl).click(function() {
		$.ajax("/_api", {
			method: "POST",
			data: {
				method: "series",
				stations: $elSelectStation[0].selectize.getValue().join(","),
				landuse: $elSelectLanduse[0].selectize.getValue().join(","),
				fields: $elSelectFields[0].selectize.getValue().join(","),
				start: $(opts.sDateEl).val(),
				end: $(opts.eDateEl).val(),
				altitude: $(opts.altitudeEl).val()
			},
			dataType: "json",
			success: function(data) {
				var dataSet = []
				var header = []
				data.forEach(function(item, index) {
					item.columns.forEach(function(col, i) {
						header.push({title: col})
					});
					item.values.forEach(function(val, i) {
						dataSet.push(val)
					});
				});

			//	$('#tbl").destroy();
				$("#tbl").DataTable({
					data: dataSet,
					columns: header
				})
			//	table(data)
			},
			error: errorHandler(error)
		});
	});

	// initalize map
	var map = L.map(opts.mapEl, {zoomControl: false}).setView([46.69765764825818, 10.638368502259254], 13);
	L.control.scale({position: "bottomright"}).addTo(map);
	L.control.zoom({position: "bottomright"}).addTo(map);
	L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png?', {
		attribution: 'Map data &copy; <a href="https://www.openstreetmap.org/">OpenStreetMap</a> contributors, <a href="https://creativecommons.org/licenses/by-sa/2.0/">CC-BY-SA</a>'
	}).addTo(map);
	

	map.on("click", function(e) {
		console.log(map.getZoom());
		console.log(map.getCenter());
	});

}
