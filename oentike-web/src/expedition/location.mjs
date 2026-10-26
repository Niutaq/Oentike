// macOS desktop uses Core Location directly; browsers retain their web API.
// Native watches sample serially, so a slow fix cannot overlap the next request.
// Purpose is required for the native trust gate (`parking` | `trail`).
export function createLocationProvider({ native, invoke, geolocation, schedule = setTimeout, unschedule = clearTimeout }) {
    if (!native) return geolocation;
    const watches = new Map();
    let nextWatch = 0;
    const mapError = (error) => ({
        code: [1, 2, 3, 4].includes(error?.code) ? error.code : 2,
    });
    const request = async (state, success, failure, purpose) => {
        state.requestId = crypto.randomUUID();
        try {
            const fix = await invoke("get_current_location", {
                requestId: state.requestId,
                purpose,
            });
            if (!state.cancelled) success({ timestamp: fix.timestamp, coords: {
                latitude: fix.latitude, longitude: fix.longitude, accuracy: fix.accuracy,
            } });
        } catch (error) {
            if (!state.cancelled) failure(mapError(error));
        } finally { state.requestId = null; }
    };
    return {
        getCurrentPosition(success, failure, options) {
            const purpose = options?.purpose === "trail" ? "trail" : "parking";
            void request({ cancelled: false }, success, failure, purpose);
        },
        watchPosition(success, failure, options) {
            const purpose = options?.purpose === "parking" ? "parking" : "trail";
            const id = ++nextWatch;
            const state = { cancelled: false, requestId: null, timer: null };
            watches.set(id, state);
            const poll = async () => {
                await request(state, success, failure, purpose);
                if (!state.cancelled) state.timer = schedule(poll, 5000);
            };
            void poll();
            return id;
        },
        clearWatch(id) {
            const state = watches.get(id);
            if (!state) return;
            state.cancelled = true;
            unschedule(state.timer);
            if (state.requestId) void invoke("cancel_location", { requestId: state.requestId }).catch(() => {});
            watches.delete(id);
        },
    };
}
