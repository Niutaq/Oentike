import test from "node:test";
import assert from "node:assert/strict";
import "fake-indexeddb/auto";
import { appendPoint, createTrip, openJournal, readTrip, updateTrip } from "../src/expedition/store.mjs";
import { bearing, distance, geoJSON, trackLength, validatePoint } from "../src/expedition/model.mjs";

const point = (lat = 50, time = 100000) => ({ lat, lon: 22, time, accuracy: 8 });
const database = () => openJournal(indexedDB, crypto.randomUUID());

test("distance and bearing handle equator, antimeridian and coincident points", () => {
    assert.ok(Math.abs(distance({ lat: 0, lon: 0 }, { lat: 0, lon: 1 }) - 111195.08) < 1);
    assert.ok(distance({ lat: 0, lon: 179.999 }, { lat: 0, lon: -179.999 }) < 223);
    assert.equal(bearing({ lat: 0, lon: 0 }, { lat: 0, lon: 1 }), 90);
    assert.equal(bearing(point(), point()), null);
});

test("invalid and poor-quality samples never become valid coordinates", () => {
    for (const invalid of [{ lat: NaN }, { lon: Infinity }, { lat: 91 }, { lon: -181 },
        { accuracy: 101 }, { accuracy: -1 }, { time: 0 }, { time: 1.5 }]) {
        assert.throws(() => validatePoint({ ...point(), ...invalid }), /invalidPoint/);
    }
});

test("committed parking and samples survive closing and reopening the database", async () => {
    const name = crypto.randomUUID();
    let db = await openJournal(indexedDB, name);
    const trip = await createTrip(db, "Las", { ...point(), source: "manual" });
    assert.equal(trip.parking.accuracy, null);
    await appendPoint(db, trip.id, point(), "first");
    await appendPoint(db, trip.id, point(50.001, 106000), "first");
    db.close();
    db = await openJournal(indexedDB, name);
    try {
        const saved = await readTrip(db, trip.id);
        assert.equal(saved.trip.count, 2);
        assert.equal(saved.points[1].seq, 1);
        assert.equal(saved.trip.parking.lat, 50);
        assert.ok(trackLength(saved.points) > 111);
    } finally { db.close(); }
});

test("restart and loss of fixes do not invent connecting trail segments", async () => {
    const db = await database();
    try {
        const trip = await createTrip(db, "Gaps", point());
        await appendPoint(db, trip.id, point(), "first");
        await appendPoint(db, trip.id, point(50.001, 106000), "first");
        await appendPoint(db, trip.id, point(51, 112000), "resumed");
        await appendPoint(db, trip.id, point(52, 150000), "resumed");
        const saved = await readTrip(db, trip.id);
        assert.deepEqual(saved.points.map(p => p.breakBefore), [true, false, true, true]);
        assert.ok(trackLength(saved.points) < 112);
        const features = geoJSON(saved.trip, saved.points).features;
        assert.deepEqual(features.map(f => f.geometry.type), ["Point", "LineString", "Point", "Point"]);
        assert.deepEqual(features[1].geometry.coordinates[0], [22, 50]);
        assert.equal(features[1].properties.times.length, 2);
    } finally { db.close(); }
});

test("rejected stale samples leave both point count and data unchanged", async () => {
    const db = await database();
    try {
        const trip = await createTrip(db, "Order", point());
        await appendPoint(db, trip.id, point(), "a");
        await assert.rejects(appendPoint(db, trip.id, point(51, 99000), "a"), /oldPoint/);
        const saved = await readTrip(db, trip.id);
        assert.equal(saved.trip.count, 1);
        assert.equal(saved.points.length, 1);
    } finally { db.close(); }
});

test("concurrent writers serialize sequence allocation without losing samples", async () => {
    const db = await database();
    try {
        const trip = await createTrip(db, "Concurrent", point());
        await Promise.all([
            appendPoint(db, trip.id, point(50, 100000), "a"),
            appendPoint(db, trip.id, point(50.001, 106000), "b"),
        ]);
        const saved = await readTrip(db, trip.id);
        assert.equal(saved.trip.count, 2);
        assert.deepEqual(saved.points.map(p => p.seq), [0, 1]);
        assert.equal(trackLength(saved.points), 0);
    } finally { db.close(); }
});


test("a plan survives restart, can gain parking, and keeps its trail when edited", async () => {
    const name = crypto.randomUUID();
    let db = await openJournal(indexedDB, name);
    const plan = await createTrip(db, "Jesienny las", null, { area: "Janów", date: "2026-10-10", notes: "Termos" });
    await assert.rejects(appendPoint(db, plan.id, point(), "a"), /parkingRequired/);
    db.close();
    db = await openJournal(indexedDB, name);
    try {
        const saved = await readTrip(db, plan.id);
        assert.equal(saved.trip.parking, null);
        assert.equal(saved.trip.notes, "Termos");
        assert.deepEqual(geoJSON(saved.trip, []).features, []);
        assert.equal(geoJSON(saved.trip, []).properties.area, "Janów");
        await updateTrip(db, plan.id, "Jesienny las", { ...point(), source: "manual" }, saved.trip);
        await appendPoint(db, plan.id, point(), "a");
        await updateTrip(db, plan.id, "Nowa nazwa", point(51), { ...saved.trip, notes: "Zmienione" });
        const edited = await readTrip(db, plan.id);
        assert.equal(edited.trip.parking.lat, 50);
        assert.equal(edited.trip.parking.accuracy, null);
        assert.equal(edited.trip.notes, "Zmienione");
        assert.equal(edited.points.length, 1);
        assert.equal(edited.trip.count, 1);
    } finally { db.close(); }
});

test("invalid plan dates and oversized notes do not write a trip", async () => {
    const db = await database();
    try {
        for (const date of ["2026-02-30", "2026-13-01", "not-a-date"])
            assert.throws(() => createTrip(db, "Plan", null, { date }), /plan/);
        assert.throws(() => createTrip(db, "Plan", null, { notes: "x".repeat(4001) }), /plan/);
        assert.throws(() => createTrip(db, "Plan", { ...point(), lon: NaN }), /invalidPoint/);
    } finally { db.close(); }
});
