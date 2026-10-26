import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createLocationProvider } from '../src/expedition/location.mjs';

test('browsers retain the browser geolocation provider', () => {
    const geolocation = {};
    assert.equal(createLocationProvider({ native: false, geolocation }), geolocation);
});

test('native fixes and structured errors map to the journal API', async () => {
    let allowed = false;
    const provider = createLocationProvider({ native: true, invoke: async (command, args) => {
        assert.equal(command, 'get_current_location');
        assert.ok(args.requestId);
        assert.equal(args.purpose, 'parking');
        if (!allowed) throw { code: 1 };
        return { latitude: 50.7, longitude: 22.4, accuracy: 8, timestamp: 1000 };
    } });
    const error = await new Promise(resolve => provider.getCurrentPosition(assert.fail, resolve));
    assert.equal(error.code, 1);
    allowed = true;
    const fix = await new Promise((resolve, reject) => provider.getCurrentPosition(resolve, reject));
    assert.deepEqual(fix, { timestamp: 1000, coords: { latitude: 50.7, longitude: 22.4, accuracy: 8 } });
});

test('native untrusted host is mapped as code 4 and purpose defaults by call type', async () => {
    const purposes = [];
    const provider = createLocationProvider({ native: true, invoke: async (command, args) => {
        if (command === 'get_current_location') {
            purposes.push(args.purpose);
            throw { code: 4 };
        }
    } });
    const parking = await new Promise(resolve => provider.getCurrentPosition(assert.fail, resolve));
    assert.equal(parking.code, 4);
    const trail = await new Promise(resolve => {
        const id = provider.watchPosition(assert.fail, error => {
            provider.clearWatch(id);
            resolve(error);
        }, { purpose: 'trail' });
    });
    assert.equal(trail.code, 4);
    assert.deepEqual(purposes, ['parking', 'trail']);
});

test('clearing a native watch cancels its in-flight fix and ignores late callbacks', async () => {
    let deliver;
    const calls = [];
    let scheduled = 0;
    const provider = createLocationProvider({ native: true,
        invoke: (command, args) => {
            calls.push({ command, args });
            return command === 'get_current_location' ? new Promise(resolve => deliver = resolve) : Promise.resolve();
        },
        schedule: () => { scheduled++; }, unschedule: () => {},
    });
    const id = provider.watchPosition(assert.fail, assert.fail);
    provider.clearWatch(id);
    assert.equal(calls[1].command, 'cancel_location');
    assert.equal(calls[1].args.requestId, calls[0].args.requestId);
    deliver({ latitude: 1, longitude: 2, accuracy: 5, timestamp: 1000 });
    await new Promise(resolve => setImmediate(resolve));
    assert.equal(scheduled, 0);
});

test('native watches wait for each fix before scheduling another and stop on denial', async () => {
    let deliver;
    let poll;
    let requests = 0;
    const provider = createLocationProvider({ native: true,
        invoke: async command => {
            if (command === 'cancel_location') return;
            requests++;
            if (requests === 2) throw { code: 1 };
            return new Promise(resolve => deliver = resolve);
        },
        schedule: callback => { poll = callback; }, unschedule: () => {},
    });
    const fixes = [];
    const id = provider.watchPosition(fix => fixes.push(fix), error => {
        assert.equal(error.code, 1);
        provider.clearWatch(id);
    });
    assert.equal(requests, 1);
    assert.equal(poll, undefined);
    deliver({ latitude: 1, longitude: 2, accuracy: 5, timestamp: 1000 });
    await new Promise(resolve => setImmediate(resolve));
    assert.equal(fixes.length, 1);
    const next = poll;
    poll = undefined;
    await next();
    assert.equal(requests, 2);
    assert.equal(poll, undefined);
});
