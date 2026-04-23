/**
 * Burst 1K Test — Can the broker handle 1,000+ msg/sec?
 *
 * Short and aggressive: 5 clients hammering the broker at maximum rate
 * for 60 seconds with no sleep between publishes. Pure throughput test.
 *
 * ┌──────────────────────────────────────────────────────────────────┐
 * │  VU │ Machine               │ Topic              │ Target       │
 * ├──────────────────────────────────────────────────────────────────┤
 * │  1  │ Solder Paste Printer  │ machine/status      │ ~200 msg/sec │
 * │  2  │ Pick-and-Place (PnP)  │ machine/production  │ ~200 msg/sec │
 * │  3  │ Reflow Oven           │ machine/alarm       │ ~200 msg/sec │
 * │  4  │ AOI Inspection        │ machine/oee         │ ~200 msg/sec │
 * │  5  │ Line Controller / MES │ machine/counter     │ ~200 msg/sec │
 * ├──────────────────────────────────────────────────────────────────┤
 * │     │ TOTAL TARGET          │                     │ ~1000 msg/sec│
 * └──────────────────────────────────────────────────────────────────┘
 *
 * Test phases (total ~90s):
 *   1. Connect     10s  — ramp 0 → 5 VUs
 *   2. Full blast  60s  — all 5 VUs publishing as fast as possible
 *   3. Cool-down   20s  — ramp 5 → 0 VUs
 *
 * Each VU publishes batches of 10 messages per iteration with zero
 * sleep — the only throttle is the QoS 1 PUBACK round-trip.
 */

import { check, sleep } from "k6";
import { Counter } from "k6/metrics";

const mqtt = require("k6/x/mqtt");

import {
  BROKER_ADDR, BROKER_USER, BROKER_PASS,
  CONNECT_TIMEOUT, PUBLISH_TIMEOUT, CLOSE_TIMEOUT,
  TOPICS,
} from "./config.js";

import {
  statusPayload,
  productionPayload,
  alarmPayload,
  oeePayload,
  counterPayload,
} from "./payloads.js";

// ── Metrics ─────────────────────────────────────────────────────────

const publishedTotal   = new Counter("published_total");
const publishedStatus  = new Counter("published_status");
const publishedProd    = new Counter("published_production");
const publishedAlarm   = new Counter("published_alarm");
const publishedOee     = new Counter("published_oee");
const publishedCounter = new Counter("published_counter");
const publishErrors    = new Counter("publish_errors");

// ── Options ─────────────────────────────────────────────────────────

export const options = {
  scenarios: {
    burst: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 5 },   // connect all
        { duration: "60s", target: 5 },   // full blast
        { duration: "20s", target: 0 },   // cool-down
      ],
      gracefulRampDown: "5s",
    },
  },
  thresholds: {
    checks:         ["rate>0.98"],
    publish_errors: ["count<100"],
  },
};

// ── Per-VU profile ──────────────────────────────────────────────────

const BATCH_SIZE = 10; // messages per iteration, zero sleep

function vuProfile(vu) {
  switch (vu) {
    case 1: return { topic: TOPICS.STATUS,     fn: statusPayload,     counter: publishedStatus  };
    case 2: return { topic: TOPICS.PRODUCTION, fn: productionPayload, counter: publishedProd    };
    case 3: return { topic: TOPICS.ALARM,      fn: alarmPayload,      counter: publishedAlarm   };
    case 4: return { topic: TOPICS.OEE,        fn: oeePayload,        counter: publishedOee     };
    case 5: return { topic: TOPICS.COUNTER,    fn: counterPayload,    counter: publishedCounter };
    default: return { topic: TOPICS.STATUS,    fn: statusPayload,     counter: publishedStatus  };
  }
}

// ── Lazy connect with retry ─────────────────────────────────────────

const MAX_RETRIES = 10;
const RETRY_BASE_MS = 500;

const publisher = new mqtt.Client(
  [BROKER_ADDR],
  BROKER_USER,
  BROKER_PASS,
  false,
  `k6-burst-${__VU}`,
  CONNECT_TIMEOUT,
);

function ensureConnected() {
  if (publisher.isConnected()) return true;

  for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
    if (attempt === 1) sleep(__VU * 0.2);

    try { publisher.connect(); } catch (_) { /* retry */ }

    if (publisher.isConnected()) {
      console.log(`VU ${__VU} connected on attempt ${attempt}`);
      return true;
    }

    const backoff = RETRY_BASE_MS * attempt / 1000;
    console.warn(`VU ${__VU} attempt ${attempt} failed, retry in ${backoff}s`);
    sleep(backoff);
  }

  console.error(`VU ${__VU} failed to connect after ${MAX_RETRIES} attempts`);
  return false;
}

// ── Main loop ───────────────────────────────────────────────────────

export default function () {
  if (!publisher.isConnected()) {
    const ok = ensureConnected();
    check(ok, { "connected": (v) => v === true });
    if (!ok) { sleep(1); return; }
  }

  const profile = vuProfile(__VU);

  // Fire BATCH_SIZE messages with no sleep — pure throughput
  for (let i = 0; i < BATCH_SIZE; i++) {
    const seq = __ITER * BATCH_SIZE + i;
    const payload = profile.fn(__VU, seq);

    let err;
    try {
      publisher.publish(profile.topic, 1, payload, false, PUBLISH_TIMEOUT);
    } catch (e) {
      err = e;
      publishErrors.add(1);
    }

    check(err, { "publish ok": (e) => e === undefined });

    if (err === undefined) {
      publishedTotal.add(1);
      profile.counter.add(1);
    }
  }

  // No sleep — let QoS 1 PUBACK be the only throttle
}

// ── Teardown ────────────────────────────────────────────────────────

export function teardown() {
  publisher.close(CLOSE_TIMEOUT);
}
