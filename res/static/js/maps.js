


(function() {
    L.Control.FocusControl = L.Control.extend({
        onAdd: function (map) {
            let el = L.DomUtil.create('div', 'leaflet-bar leaflet-control gpx-control-container');
            let button = L.DomUtil.create('a', 'gpx-map-control', el);
            button.innerHTML = '<strong>F</strong>';
            button.href = '#';

            const boundsF = this._boundsF;

            button.addEventListener('click', function (e) {
                e.preventDefault();
                map.fitBounds(boundsF());
            });

            return el;
        },

        onRemove: function (map) {
        }
    });

    L.control.focusControl = function (boundsF, opts) {
        const obj = new L.Control.FocusControl(opts);
        obj._boundsF = boundsF;
        return obj;
    }
})();

function mountMap(container, data) {
    let map = L.map(container, {
        scrollWheelZoom: false,
    });

    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
    }).addTo(map);

    map.on('focus', function() { map.scrollWheelZoom.enable(); });
    map.on('blur', function() { map.scrollWheelZoom.disable(); });


    const overlayLayers = {}
    const focusControlLayers = [];

    if (data.track) {
        let latLngs = data.track;
        const polyline = L.polyline(latLngs, { color: 'blue' }).addTo(map);
        overlayLayers.Track = polyline;
        focusControlLayers.push(polyline);
    }

    if (data.images) {
        const markers = data.images.map(function (img) {
            console.log(img);
            let marker = L.marker(img.LatLng);

            let popupContainer = L.DomUtil.create('div', 'gpx-map-marker');
            let popupAnchor = L.DomUtil.create('a', '', popupContainer);
            popupAnchor.href = img.URI;
            let popupImage = L.DomUtil.create('img', '', popupAnchor);
            popupImage.src = img.ThumbURI;

            marker.bindPopup(popupContainer);

            return marker;
        });

        overlayLayers.Photos = L.layerGroup(markers).addTo(map);
    }

    L.control.layers({}, overlayLayers).addTo(map);

    L.control.focusControl(
        () => {
            let bounds = focusControlLayers[0].getBounds();
            for (let i = 1; i < focusControlLayers.length; i++) {
                bounds.extend(focusControlLayers[i].getBounds());
            }
            return bounds;
        },
        {
            position: 'topleft',
        }
    ).addTo(map);

    function mapFitBounds() {
        console.log(focusControlLayers[0].getBounds());
        map.fitBounds(focusControlLayers[0].getBounds());
    }
    mapFitBounds();
}

function mountGlobalMap(container, data) {
    let map = L.map(container, {
        scrollWheelZoom: false,
    });

    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
    }).addTo(map);

    map.on('focus', function() { map.scrollWheelZoom.enable(); });
    map.on('blur', function() { map.scrollWheelZoom.disable(); });

    let bounds = null;
    const overlayLayers = {};

    if (data) {
        const groups = new Map();

        const markers = data.map(function (img) {
            let marker = L.marker(img.LatLng);

            let popupContainer = L.DomUtil.create('div', 'gpx-map-marker');

            let popupAnchor = L.DomUtil.create('a', '', popupContainer);
            popupAnchor.title = img.Title;
            popupAnchor.href = img.URI;

            let popupImage = L.DomUtil.create('img', '', popupAnchor);
            popupImage.src = img.Preview;

            marker.bindPopup(popupContainer);

            if (groups[img.Year] === undefined) { 
                groups[img.Year] = [marker];
            } else {
                groups[img.Year].push(marker);
            }

            return marker;
        });

        for (let k in groups) {
            const v = groups[k];
            overlayLayers["" + k] = L.layerGroup(v).addTo(map);
        }

        bounds = L.latLngBounds(markers.map(marker => marker.getLatLng()));

    }

    L.control.layers({}, overlayLayers).addTo(map);

    L.control.focusControl(
        () => bounds,
        { position: 'topleft' }
    ).addTo(map);

    function mapFitBounds() {
        map.fitBounds(bounds);
    }
    mapFitBounds();
}

function loadAndMountMap(container, opts) {
    let http = new XMLHttpRequest();
    http.onreadystatechange = function() {
        if (this.readyState === 4 && this.status === 200) {
            const data = JSON.parse(this.responseText);
            mountMap(container, data);
        }
    }
    http.open('GET', opts.dataURL, true);
    http.send();
}