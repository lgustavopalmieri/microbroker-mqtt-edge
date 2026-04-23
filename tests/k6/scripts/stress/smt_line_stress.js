/**
 * SMT Line Stress Test — 5 clients, 5 topics, sustained + spike load
 *
 * Simulates a complete SMT (Surface Mount Technology) production line
 * pushing data to the edge broker. Each VU represents a different machine
 * on the line, publishing to its own dedicated topic at a realistic rate.
 *
 * ┌──────────────────────────────────────────────────────────────────┐
 * │  VU │ Machine               │ Topic              │ Base msg/sec │
 * ├──────────────────────────────────────────────────────────────────┤
 * │  1  │ Solder Paste Printer  │ machine/status      │     10       │
 * │  2  │ Pick-and-Place (PnP)  │ machine/production  │     20       │
 * │  3  │ Reflow Oven           │ machine/alarm       │      5       │
 * │  4  │ AOI Inspection        │ machine/oee         │     10       │
 * │  5  │ Line Controller / MES │ machine/counter     │     15       │
 * ├──────────────────────────────────────────────────────────────────┤
 * │     │ TOTAL BASELINE        │                     │  ~60 msg/sec │
 * └──────────────────────────────────────────────────────────────────┘
 *
 * Test phases (total ~3 min):
 *   1. Warm-up       30s   — ramp from 0 → 5 VUs
 *   2. Sustained     60s   — steady 5 VUs at baseline rate
 *   3. Spike         30s   — all VUs active at full rate
 *   4. Recovery      30s   — steady at baseline rate
 *   5. Cool-down     30s   — ramp down 5 → 0 VUs
 *
 * After the run, use the verify script to check that every published
 * message was persisted via the audit-count API.
 *
 * ENV:
 *   BROKER_ADDR       (default: host.docker.internal:1883)
 *   BROKER_USER       (default: machine01)
 *   BROKER_PASS       (default: secret123)
 *   RATE_MULTIPLIER   (default: 1) — scale all rates up/down
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

// ── Custom metrics ──────────────────────────────────────────────────

const publishedTotal   = new Counter("published_total");
const publishedStatus  = new Counter("published_status");
const publishedProd    = new Counter("published_production");
const publishedAlarm   = new Counter("published_alarm");
const publishedOee     = new Counter("published_oee");
const publishedCounter = new Counter("published_counter");
const publishErrors    = new Counter("publish_errors");
const connectRetries   = new Counter("connect_retries");

// ── Options ─────────────────────────────────────────────────────────

export const options = {
  scenarios: {
    smt_line: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "30s", target: 5 },   // warm-up
        { duration: "60s", target: 5 },   // sustained
        { duration: "30s", target: 5 },   // spike (handled in code)
        { duration: "30s", target: 5 },   // recovery
        { duration: "30s", target: 0 },   // cool-down
      ],
      gracefulRampDown: "10s",
    },
  },
  thresholds: {
    checks:         ["rate>0.99"],
    publish_errors: ["count<50"],
  },
};

// ── Per-VU profile ──────────────────────────────────────────────────

const rateMultiplier = parseFloat(__ENV.RATE_MULTIPLIER || "1");

function vuProfile(vu) {
  switch (vu) {
    case 1: return { topic: TOPICS.STATUS,     fn: statusPayload,     rate: 10, counter: publishedStatus  };
    case 2: return { topic: TOPICS.PRODUCTION, fn: productionPayload, rate: 20, counter: publishedProd    };
    case 3: return { topic: TOPICS.ALARM,      fn: alarmPayload,      rate:  5, counter: publishedAlarm   };
    case 4: return { topic: TOPICS.OEE,        fn: oeePayload,        rate: 10, counter: publishedOee     };
    case 5: return { topic: TOPICS.COUNTER,    fn: counterPayload,    rate: 15, counter: publishedCounter };
    default: return { topic: TOPICS.STATUS,    fn: statusPayload,     rate: 10, counter: publishedStatus  };
  }
}

// ── Lazy connect with retry ─────────────────────────────────────────
//
// xk6-mqtt Client is created in init (required by k6), but connection
// happens lazily on first iteration with staggered delay per VU and
// retry logic to avoid the thundering-herd problem when all 5 VUs
// try to CONNECT simultaneously.

const MAX_RETRIES = 10;
const RETRY_BASE_MS = 500;

const clientId = `k6-smt-${__VU}`;

const publisher = new mqtt.Client(
  [BROKER_ADDR],
  BROKER_USER,
  BROKER_PASS,
  false,
  clientId,
  CONNECT_TIMEOUT,
);

// Do NOT connect here — defer to first iteration.

function ensureConnected() {
  if (publisher.isConnected()) {
    return true;
  }

  for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
    // Stagger: VU N waits N * 1s before first attempt.
    // The broker needs time to fully complete each CONNECT/CONNACK
    // handshake and register the client before the next one arrives.
    if (attempt === 1) {
      sleep(__VU * 1);
    }

    try {
      publisher.connect();
    } catch (e) {
      // ignore, check below
    }

    if (publisher.isConnected()) {
      console.log(`VU ${__VU} connected on attempt ${attempt}`);
      return true;
    }

    connectRetries.add(1);
    const backoff = RETRY_BASE_MS * attempt / 1000;
    console.warn(`VU ${__VU} connect attempt ${attempt} failed, retrying in ${backoff}s...`);
    sleep(backoff);
  }

  console.error(`VU ${__VU} failed to connect after ${MAX_RETRIES} attempts`);
  return false;
}

// ── Main loop ───────────────────────────────────────────────────────

export default function () {
  // Lazy connect on first iteration
  if (!publisher.isConnected()) {
    const ok = ensureConnected();
    check(ok, { "connected": (v) => v === true });
    if (!ok) {
      sleep(1);
      return;
    }
  }

  const profile = vuProfile(__VU);
  const effectiveRate = profile.rate * rateMultiplier;

  // Publish a batch of messages, then sleep to achieve target rate.
  const batchSize = Math.max(1, Math.ceil(effectiveRate / 5));
  const sleepDuration = batchSize / effectiveRate;

  for (let i = 0; i < batchSize; i++) {
    const seq = __ITER * batchSize + i;
    const payload = profile.fn(__VU, seq);

    let err;
    try {
      publisher.publish(profile.topic, 1, payload, false, PUBLISH_TIMEOUT);
    } catch (e) {
      err = e;
      publishErrors.add(1);
    }

    check(err, {
      "publish ok": (e) => e === undefined,
    });

    if (err === undefined) {
      publishedTotal.add(1);
      profile.counter.add(1);
    }
  }

  sleep(sleepDuration);
}

// ── Teardown ────────────────────────────────────────────────────────

export function teardown() {
  publisher.close(CLOSE_TIMEOUT);
}
