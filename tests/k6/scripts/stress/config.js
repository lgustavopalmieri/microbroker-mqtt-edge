/**
 * Shared configuration for stress tests.
 *
 * All environment variables are read here and exported as constants
 * so individual test scripts stay clean.
 */

export const BROKER_ADDR = __ENV.BROKER_ADDR || "host.docker.internal:1883";
export const BROKER_USER = __ENV.BROKER_USER || "machine01";
export const BROKER_PASS = __ENV.BROKER_PASS || "secret123";

export const CONNECT_TIMEOUT = 5000;
export const PUBLISH_TIMEOUT = 5000;
export const CLOSE_TIMEOUT   = 5000;

// Topics matching the broker's allowed list
export const TOPICS = {
  STATUS:     "machine/status",
  PRODUCTION: "machine/production",
  ALARM:      "machine/alarm",
  OEE:        "machine/oee",
  COUNTER:    "machine/counter",
};
