import { GAP_MS, MAX_POINTS, validatePoint } from "./model.mjs";

// A sample and its sequence counter commit together. No success before oncomplete.
export function openJournal(factory = indexedDB, name = "oentike-expeditions-v1") {
    return new Promise((resolve, reject) => {
        const request = factory.open(name, 1);
        request.onupgradeneeded = () => {
            request.result.createObjectStore("trips", { keyPath: "id" });
            request.result.createObjectStore("points", { keyPath: ["tripId", "seq"] });
        };
        request.onerror = () => reject(request.error);
        request.onblocked = () => reject(new Error("blocked"));
        request.onsuccess = () => {
            const db = request.result;
            db.onversionchange = () => db.close();
            resolve(db);
        };
    });
}

function transaction(db, stores, mode, work) {
    return new Promise((resolve, reject) => {
        let tx;
        try { tx = db.transaction(stores, mode, { durability: "strict" }); }
        catch (error) { reject(error); return; }
        let result, failure;
        tx.oncomplete = () => resolve(result);
        tx.onabort = () => reject(failure || tx.error || new Error("storage"));
        tx.onerror = () => {}; // onabort is the single failure path.
        const fail = (error) => { failure = error; tx.abort(); };
        try { work(tx, value => { result = value; }, fail); }
        catch (error) { fail(error); }
    });
}

export function listTrips(db) {
    return transaction(db, ["trips"], "readonly", (tx, done) => {
        tx.objectStore("trips").getAll().onsuccess = event =>
            done(event.target.result.sort((a, b) => b.createdAt - a.createdAt));
    });
}

export function readTrip(db, id) {
    return transaction(db, ["trips", "points"], "readonly", (tx, done, fail) => {
        const request = tx.objectStore("trips").get(id);
        request.onsuccess = () => {
            if (!request.result) { fail(new Error("missing")); return; }
            const trip = request.result;
            const points = tx.objectStore("points").getAll(IDBKeyRange.bound([id, 0], [id, MAX_POINTS]));
            points.onsuccess = () => done({ trip, points: points.result });
        };
    });
}

export function createTrip(db, name, parking) {
    const manual = parking?.source === "manual";
    parking = { ...validatePoint(parking), source: manual ? "manual" : "geolocation" };
    if (manual) parking.accuracy = null;
    name = name.trim();
    if (!name || name.length > 80) throw new Error("name");
    const trip = { id: crypto.randomUUID(), name, parking, createdAt: Date.now(), count: 0, last: null };
    return transaction(db, ["trips"], "readwrite", (tx, done) => {
        tx.objectStore("trips").add(trip);
        done(trip);
    });
}

export function appendPoint(db, id, input, session) {
    const point = validatePoint(input);
    if (typeof session !== "string" || !session || session.length > 80) throw new Error("session");
    return transaction(db, ["trips", "points"], "readwrite", (tx, done, fail) => {
        const trips = tx.objectStore("trips");
        const request = trips.get(id);
        request.onsuccess = () => {
            const trip = request.result;
            if (!trip) { fail(new Error("missing")); return; }
            if (trip.count >= MAX_POINTS) { fail(new Error("limit")); return; }
            if (trip.last && point.time <= trip.last.time) { fail(new Error("oldPoint")); return; }
            const sample = { ...point, tripId: id, seq: trip.count, session,
                breakBefore: !trip.last || trip.last.session !== session || point.time - trip.last.time > GAP_MS };
            tx.objectStore("points").add(sample);
            trips.put({ ...trip, count: trip.count + 1, last: sample });
            done(sample);
        };
    });
}
