/**
 * k6 + xk6-mqtt — Smoke test
 *
 * Publishes MESSAGE_COUNT messages to a single topic and checks
 * that every publish succeeds. After the run, the verify.sh wrapper
 * hits the audit API to confirm all messages were persisted.
 *
 * ENV (set via docker-compose):
 *   BROKER_ADDR   – host:port of the MQTT broker  (default: broker:1883)
 *   BROKER_USER   – MQTT username                  (default: machine01)
 *   BROKER_PASS   – MQTT password                  (default: secret123)
 *   MQTT_TOPIC    – topic to publish to            (default: machine/status)
 *   MESSAGE_COUNT – messages per VU iteration      (default: 50)
 */

import { check } from "k6";

const mqtt = require("k6/x/mqtt");

const addr = __ENV.BROKER_ADDR || "host.docker.internal:1883";
const user = __ENV.BROKER_USER || "machine01";
const pass = __ENV.BROKER_PASS || "secret123";
const topic = __ENV.MQTT_TOPIC || "machine/status";
const messageCount = parseInt(__ENV.MESSAGE_COUNT || "50", 10);

const connectTimeout = 5000;
const publishTimeout = 5000;

const clientId = `k6-pub-${__VU}`;

const publisher = new mqtt.Client(
  [addr],
  user,
  pass,
  false,
  clientId,
  connectTimeout
);

let connectErr;
try {
  publisher.connect();
} catch (e) {
  connectErr = e;
}

if (connectErr !== undefined) {
  console.error(`VU ${__VU} connect error: ${connectErr}`);
}

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: {
    checks: ["rate==1.0"],
  },
};

export default function () {
  check(publisher, {
    "publisher connected": (p) => p.isConnected(),
  });

  for (let i = 0; i < messageCount; i++) {
    const payload = JSON.stringify({
      machine_id: "CNC-01",
      vu: __VU,
      seq: i,
      ts: new Date().toISOString(),
    });

    let err;
    try {
      publisher.publish(topic, 1, payload, false, publishTimeout);
    } catch (e) {
      err = e;
    }

    check(err, {
      "publish ok": (e) => e === undefined,
    });
  }
}

export function teardown() {
  publisher.close(5000);
}
