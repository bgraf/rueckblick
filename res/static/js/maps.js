


(function() {
    L.Control.FocusControl = L.Control.extend({
        onAdd: function (map) {
            let paths = this._paths;

            let el = L.DomUtil.create('div', 'leaflet-bar leaflet-control gpx-control-container');
            let button = L.DomUtil.create('a', 'gpx-map-control', el);
            button.innerHTML = '<strong>F</strong>';
            button.href = '#';

            button.addEventListener('click', function (e) {
                e.preventDefault();
                var bounds = paths[0].getBounds();
                for (var i = 1; i < paths.length; i++) {
                    bounds.extend(paths[i].getBounds());
                }
                map.fitBounds(bounds);
            });

            return el;
        },

        onRemove: function (map) {
        }
    });

    L.control.focusControl = function (paths, opts) {
        const obj = new L.Control.FocusControl(opts);
        obj._paths = paths;
        return obj;
    }
})();

function mountMap(container, data) {
    var map = L.map(container, {
        scrollWheelZoom: false,
    });

    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
    }).addTo(map);

    map.on('focus', function() { map.scrollWheelZoom.enable(); });
    map.on('blur', function() { map.scrollWheelZoom.disable(); });

    let latlngs = data.track;

    const polyline = L.polyline(latlngs, { color: 'blue' }).addTo(map);

    function mapFitBounds() {
        map.fitBounds(polyline.getBounds());
    }

    L.control.focusControl(
        [polyline], 
        {
            position: 'topleft',
        }
    ).addTo(map);

    mapFitBounds();
}

function loadAndMountMap(container, opts) {
    let http = new XMLHttpRequest();
    http.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            const data = JSON.parse(this.responseText);
            mountMap(container, data);
        }
    }
    http.open('GET', opts.dataURL, true);
    http.send();
}