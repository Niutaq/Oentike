// Coordinates stay in WGS84. Distances are spherical surface estimates, not routing.
export const MAX_POINTS = 100000;
export const GAP_MS = 30000;
export const MAX_ACCURACY = 100;

export function validatePoint(point) {
    if (!point || !Number.isFinite(point.lat) || Math.abs(point.lat) > 90 ||
        !Number.isFinite(point.lon) || Math.abs(point.lon) > 180 ||
        !Number.isSafeInteger(point.time) || point.time <= 0 ||
        !Number.isFinite(point.accuracy) || point.accuracy < 0 || point.accuracy > MAX_ACCURACY) {
        throw new Error("invalidPoint");
    }
    return { lat: point.lat, lon: point.lon, time: point.time, accuracy: point.accuracy };
}

const radians = (degrees) => degrees * Math.PI / 180;
export function distance(a, b) {
    const h = Math.sin(radians(b.lat - a.lat) / 2) ** 2 +
        Math.cos(radians(a.lat)) * Math.cos(radians(b.lat)) *
        Math.sin(radians(b.lon - a.lon) / 2) ** 2;
    return 6371008.8 * 2 * Math.asin(Math.sqrt(Math.min(1, Math.max(0, h))));
}

export function bearing(a, b) {
    if (distance(a, b) < 1) return null;
    const d = radians(b.lon - a.lon);
    return (Math.atan2(Math.sin(d) * Math.cos(radians(b.lat)),
        Math.cos(radians(a.lat)) * Math.sin(radians(b.lat)) -
        Math.sin(radians(a.lat)) * Math.cos(radians(b.lat)) * Math.cos(d)) * 180 / Math.PI + 360) % 360;
}

export function segments(points) {
    const result = [];
    for (const point of points) {
        if (!result.length || point.breakBefore) result.push([]);
        result[result.length - 1].push(point);
    }
    return result;
}

export function trackLength(points) {
    return points.reduce((sum, point, i) => sum +
        (i && !point.breakBefore ? distance(points[i - 1], point) : 0), 0);
}

export function geoJSON(trip, points) {
    return {
        type: "FeatureCollection",
        features: [
            { type: "Feature", properties: { kind: "parking", name: trip.name,
                time: new Date(trip.parking.time).toISOString(), accuracy: trip.parking.accuracy,
                source: trip.parking.source },
              geometry: { type: "Point", coordinates: [trip.parking.lon, trip.parking.lat] } },
            ...segments(points).map((part) => ({
                type: "Feature", properties: { kind: "track", name: trip.name,
                    times: part.map(p => new Date(p.time).toISOString()),
                    accuracy: part.map(p => p.accuracy) },
                geometry: part.length === 1
                    ? { type: "Point", coordinates: [part[0].lon, part[0].lat] }
                    : { type: "LineString", coordinates: part.map(p => [p.lon, p.lat]) },
            })),
        ],
    };
}
