import i18next from "i18next";
import { fieldText } from "./copy.mjs";
import { bearing, distance, geoJSON, segments, trackLength, validatePoint } from "./model.mjs";
import { appendPoint, createTrip, listTrips, openJournal, readTrip } from "./store.mjs";

type Point = { lat: number; lon: number; time: number; accuracy: number; breakBefore?: boolean };
type Trip = { id: string; name: string; parking: Point; createdAt: number };
const element = <T extends HTMLElement>(id: string) => document.getElementById(id) as T;
const button = (id: string) => element<HTMLButtonElement>(id);
const input = (id: string) => element<HTMLInputElement>(id);
const select = element<HTMLSelectElement>("trip-select");
const fields = element<HTMLFieldSetElement>("trip-fields");
let db: IDBDatabase;
let trip: Trip | null = null;
let points: Point[] = [];
let watch: number | null = null;
let epoch = 0;
let queue = Promise.resolve();
let busy = false;
let statusKey = "loading";
let statusError = false;
let storageKey = "";
let parkingFix: Point | null = null;
const t = (key: string) => fieldText(key, document.documentElement.lang);
const message = (key: string, error = false) => {
    statusKey = key;
    statusError = error;
    element("field-status").textContent = t(key);
    element("field-status").dataset.error = String(error);
};

function controls() {
    fields.disabled = !db || busy || watch !== null;
    select.disabled = !db || busy || watch !== null || !select.options.length;
    button("record").disabled = !trip || busy || watch !== null;
    button("pause").disabled = watch === null;
    button("export-trip").disabled = !trip || busy;
}

function stop(key = "paused") {
    epoch++;
    if (watch !== null) navigator.geolocation.clearWatch(watch);
    watch = null;
    controls();
    message(key);
}

function failure(error: unknown) {
    stop();
    const key = error instanceof Error ? error.message : "storage";
    message(["invalidPoint", "limit", "oldPoint"].includes(key) ? key : key === "name" ? "nameError" : "storage", true);
}

function locationError(error: GeolocationPositionError) {
    stop();
    message(error.code === 1 ? "permission" : error.code === 3 ? "timeout" : "unavailable", true);
}

function sample(position: GeolocationPosition): Point {
    if (Math.abs(Date.now() - position.timestamp) > 30000) throw new Error("invalidPoint");
    return validatePoint({ lat: position.coords.latitude, lon: position.coords.longitude,
        accuracy: position.coords.accuracy, time: Math.round(position.timestamp) });
}

function sketch() {
    const lines = element("trail-lines");
    const markers = element("trail-markers");
    lines.replaceChildren();
    markers.replaceChildren();
    if (!trip) return;
    const parking = trip.parking;
    const xy = (p: Point) => [(((p.lon - parking.lon + 540) % 360) - 180) *
        Math.cos(parking.lat * Math.PI / 180) * 111195, (parking.lat - p.lat) * 111195];
    const all = [parking, ...points];
    let minX = -25, maxX = 25, minY = -25, maxY = 25;
    for (const p of all) {
        const [x, y] = xy(p);
        minX = Math.min(minX, x); maxX = Math.max(maxX, x);
        minY = Math.min(minY, y); maxY = Math.max(maxY, y);
    }
    const scale = Math.min(500 / (maxX - minX), 290 / (maxY - minY));
    const project = (p: Point) => {
        const [x, y] = xy(p);
        return [300 + (x - (minX + maxX) / 2) * scale, 200 + (y - (minY + maxY) / 2) * scale];
    };
    const svg = (name: string, attrs: Record<string, string>) => {
        const node = document.createElementNS("http://www.w3.org/2000/svg", name);
        Object.entries(attrs).forEach(([key, value]) => node.setAttribute(key, value));
        return node;
    };
    // Bound drawn vertices, preserving each segment's endpoints and every gap.
    const stride = Math.max(1, Math.ceil(points.length / 2000));
    for (const part of segments(points)) {
        const displayed = part.filter((_: Point, i: number) => i % stride === 0 || i === part.length - 1);
        if (part.length === 1) {
            const [x, y] = project(part[0]);
            lines.append(svg("circle", { cx: String(x), cy: String(y), r: "2", fill: "var(--orange)", stroke: "none" }));
        } else {
            lines.append(svg("polyline", { points: displayed.map((p: Point) => project(p).join(",")).join(" "),
                fill: "none", stroke: "var(--orange)", "stroke-width": "3" }));
        }
    }
    const marker = (p: Point, label: string, color: string, offset: number) => {
        const [x, y] = project(p);
        markers.append(svg("circle", { cx: String(x), cy: String(y), r: "6", fill: color, stroke: "var(--surface)", "stroke-width": "2" }));
        const text = svg("text", { x: String(x), y: String(y + offset), fill: "var(--ink)", stroke: "none", "text-anchor": "middle", "font-size": "12" });
        text.textContent = label;
        markers.append(text);
    };
    marker(parking, t("parking"), "var(--ink)", -14);
    if (points.length) marker(points[points.length - 1], t("lastPoint"), "var(--orange)", 23);
}

function render() {
    const last = points.at(-1);
    const length = (meters: number) => meters < 1000 ? `${Math.round(meters)} m` : `${(meters / 1000).toFixed(2)} km`;
    element("active-trip").textContent = trip?.name || t("empty");
    element("parking-coordinates").textContent = trip ? `${t("parking")}: ${trip.parking.lat.toFixed(6)}, ${trip.parking.lon.toFixed(6)}` : "";
    element("trail-distance").textContent = trip ? length(trackLength(points)) : "—";
    element("parking-distance").textContent = last && trip ? length(distance(last, trip.parking)) : "—";
    const azimuth = last && trip ? bearing(last, trip.parking) : null;
    element("parking-bearing").textContent = azimuth === null ? "—" : `${Math.round(azimuth) % 360}°`;
    element("sample-count").textContent = String(points.length);
    element("last-sample").textContent = last ? new Date(last.time).toLocaleString(document.documentElement.lang) : "—";
    element("sample-accuracy").textContent = last ? `${Math.round(last.accuracy)} m` : "—";
    sketch();
    controls();
}

async function load(id: string) {
    await queue;
    const result = await readTrip(db, id);
    trip = result.trip;
    points = result.points;
    render();
}

async function refreshList(id?: string) {
    const trips = await listTrips(db);
    select.replaceChildren(...trips.map((item: Trip) => {
        const option = document.createElement("option");
        option.value = item.id; option.textContent = item.name;
        return option;
    }));
    if (trips.length) {
        select.value = id || trips[0].id;
        await load(select.value);
    }
}

element("trip-form").addEventListener("submit", async event => {
    event.preventDefault();
    if (busy || watch !== null) return;
    busy = true; controls();
    try {
        const lat = input("parking-lat").valueAsNumber;
        const lon = input("parking-lon").valueAsNumber;
        const parking = parkingFix && lat === parkingFix.lat && lon === parkingFix.lon ? parkingFix :
            { lat, lon, time: Date.now(), accuracy: 0, source: "manual" };
        const created = await createTrip(db, input("trip-name").value, parking);
        await refreshList(created.id);
        message("saved");
    } catch (error) { failure(error); }
    finally { busy = false; controls(); }
});

button("locate-parking").addEventListener("click", () => {
    if (!navigator.geolocation) { message("unavailable", true); return; }
    busy = true; controls(); message("locating");
    navigator.geolocation.getCurrentPosition(position => {
        try {
            parkingFix = sample(position);
            input("parking-lat").value = String(parkingFix.lat);
            input("parking-lon").value = String(parkingFix.lon);
            message("located");
        } catch { message("poor", true); }
        finally { busy = false; controls(); }
    }, error => { busy = false; locationError(error); }, { enableHighAccuracy: true, maximumAge: 0, timeout: 15000 });
});

select.addEventListener("change", async () => {
    stop(); busy = true; controls();
    try { await load(select.value); message("ready"); }
    catch (error) { failure(error); }
    finally { busy = false; controls(); }
});

button("record").addEventListener("click", () => {
    if (!trip || watch !== null || busy) return;
    if (!navigator.geolocation) { message("unavailable", true); return; }
    const token = ++epoch;
    const session = crypto.randomUUID();
    const tripId = trip.id;
    message("recording");
    watch = navigator.geolocation.watchPosition(position => {
        if (token !== epoch) return;
        let point: Point;
        try { point = sample(position); }
        catch { message("poor", true); return; }
        queue = queue.then(async () => {
            if (token !== epoch) return;
            const last = points.at(-1);
            if (last && point.time <= last.time) { message("oldPoint", true); return; }
            if (last && point.time - last.time < 5000) return;
            const saved = await appendPoint(db, tripId, point, session);
            points.push(saved);
            render();
            if (token === epoch) message("recording");
        }).catch(failure);
    }, error => { if (token === epoch) locationError(error); },
    { enableHighAccuracy: true, maximumAge: 0, timeout: 20000 });
    controls();
});

button("pause").addEventListener("click", () => stop());
button("export-trip").addEventListener("click", async () => {
    if (!trip || busy) return;
    busy = true; controls();
    try {
        await queue;
        const snapshot = await readTrip(db, trip.id);
        const url = URL.createObjectURL(new Blob([JSON.stringify(geoJSON(snapshot.trip, snapshot.points), null, 2)], { type: "application/geo+json" }));
        const anchor = document.createElement("a");
        anchor.href = url; anchor.download = `oentike-${trip.id}.geojson`;
        document.body.append(anchor); anchor.click(); anchor.remove();
        setTimeout(() => URL.revokeObjectURL(url), 10000);
    } catch (error) { failure(error); }
    finally { busy = false; controls(); }
});

button("protect-storage").addEventListener("click", async () => {
    try {
        storageKey = await navigator.storage?.persist?.() ? "persistent" : "bestEffort";
    } catch { storageKey = "bestEffort"; }
    element("storage-status").textContent = t(storageKey);
});

document.addEventListener("visibilitychange", () => {
    if (document.hidden && watch !== null) stop("hidden");
});
window.addEventListener("pagehide", () => { stop(); db?.close(); });
window.addEventListener("pageshow", event => { if (event.persisted) location.reload(); });

function translate() {
    document.querySelectorAll<HTMLElement>("[data-field]").forEach(node => {
        node.textContent = t(node.dataset.field!);
    });
    document.title = t("title");
    element("sketch-title").textContent = t("diagram");
    message(statusKey, statusError);
    if (storageKey) element("storage-status").textContent = t(storageKey);
    render();
}
new MutationObserver(translate).observe(document.documentElement, { attributes: true, attributeFilter: ["lang"] });
i18next.on("languageChanged", () => queueMicrotask(translate));
translate();
try {
    db = await openJournal();
    await refreshList();
    button("protect-storage").disabled = false;
    message("ready");
    controls();
} catch (error) { failure(error); }
