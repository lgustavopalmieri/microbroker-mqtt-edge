/**
 * Realistic payload generators for each topic.
 *
 * Modeled after a high-speed SMT (Surface Mount Technology) production line
 * with the following machines:
 *
 *   VU 1 — Solder Paste Printer     → machine/status    (10 msg/sec)
 *          Publishes board alignment, squeegee pressure, paste volume.
 *
 *   VU 2 — Pick-and-Place (chipshooter) → machine/production (20 msg/sec)
 *          Publishes placement events. Real chip shooters reach ~15
 *          placements/sec; we push 20 msg/sec to stress the broker.
 *
 *   VU 3 — Reflow Oven              → machine/alarm      (5 msg/sec)
 *          Publishes zone temperatures and threshold alarms.
 *
 *   VU 4 — AOI (Automated Optical Inspection) → machine/oee (10 msg/sec)
 *          Publishes inspection results, defect counts, OEE metrics.
 *
 *   VU 5 — Line Controller / MES Gateway → machine/counter (15 msg/sec)
 *          Publishes cumulative counters: boards produced, cycle time,
 *          feeder pick counts.
 *
 * Total baseline: ~60 msg/sec across 5 clients.
 * Under spike phases the rate doubles to ~120 msg/sec.
 *
 * Reference data points (content rephrased for compliance):
 *   - High-end chip shooters place up to 53,000 components/hour (~15/sec)
 *   - Edge MQTT brokers on industrial PCs handle 500-2000 msg/sec
 *   - Vibration sensors sample at 100-1000 Hz but aggregate before MQTT publish
 */

/**
 * Solder Paste Printer — machine/status
 */
export function statusPayload(vu, seq) {
  return JSON.stringify({
    machine_id: "SPP-01",
    machine_type: "solder_paste_printer",
    vu: vu,
    seq: seq,
    status: seq % 50 === 0 ? "cleaning" : "printing",
    board_id: `PCB-${vu}-${seq}`,
    squeegee_pressure_kpa: 45 + Math.random() * 10,
    paste_volume_mm3: 0.8 + Math.random() * 0.4,
    alignment_offset_um: Math.random() * 25 - 12.5,
    cycle_time_ms: 3200 + Math.floor(Math.random() * 800),
    ts: new Date().toISOString(),
  });
}

/**
 * Pick-and-Place Chipshooter — machine/production
 * Highest frequency: simulates individual placement events.
 */
export function productionPayload(vu, seq) {
  const componentTypes = [
    "0201", "0402", "0603", "0805", "SOT-23", "QFP-48", "BGA-256", "SOP-8",
  ];
  return JSON.stringify({
    machine_id: "PNP-01",
    machine_type: "pick_and_place",
    vu: vu,
    seq: seq,
    board_id: `PCB-${vu}-${Math.floor(seq / 200)}`,
    component: componentTypes[seq % componentTypes.length],
    feeder_slot: 1 + (seq % 80),
    head: 1 + (seq % 12),
    placement_x_mm: 10 + Math.random() * 280,
    placement_y_mm: 10 + Math.random() * 180,
    rotation_deg: Math.floor(Math.random() * 4) * 90,
    vacuum_kpa: -65 + Math.random() * 10,
    nozzle_ok: Math.random() > 0.002,
    ts: new Date().toISOString(),
  });
}

/**
 * Reflow Oven — machine/alarm
 * Lower frequency but critical: zone temperatures and threshold violations.
 */
export function alarmPayload(vu, seq) {
  const zones = 10;
  const zone = (seq % zones) + 1;
  const baseTemp = zone <= 3 ? 150 : zone <= 7 ? 220 : 250;
  const temp = baseTemp + Math.random() * 15 - 7.5;
  const threshold = baseTemp + 10;
  const isAlarm = temp > threshold;

  return JSON.stringify({
    machine_id: "RFO-01",
    machine_type: "reflow_oven",
    vu: vu,
    seq: seq,
    zone: zone,
    temperature_c: Math.round(temp * 10) / 10,
    setpoint_c: baseTemp,
    threshold_c: threshold,
    alarm_active: isAlarm,
    alarm_code: isAlarm ? `TEMP-ZONE${zone}-HIGH` : null,
    severity: isAlarm ? "warning" : "normal",
    conveyor_speed_cm_min: 80 + Math.random() * 10,
    ts: new Date().toISOString(),
  });
}

/**
 * AOI (Automated Optical Inspection) — machine/oee
 * Publishes per-board inspection results and rolling OEE.
 */
export function oeePayload(vu, seq) {
  const defectRate = Math.random();
  const passed = defectRate > 0.03;

  return JSON.stringify({
    machine_id: "AOI-01",
    machine_type: "aoi_inspection",
    vu: vu,
    seq: seq,
    board_id: `PCB-${vu}-${seq}`,
    result: passed ? "pass" : "fail",
    defects_found: passed ? 0 : 1 + Math.floor(Math.random() * 3),
    inspection_time_ms: 1800 + Math.floor(Math.random() * 600),
    availability: 0.88 + Math.random() * 0.10,
    performance: 0.82 + Math.random() * 0.15,
    quality: 0.95 + Math.random() * 0.05,
    oee: 0,  // calculated below
    ts: new Date().toISOString(),
  });
}

/**
 * Line Controller / MES Gateway — machine/counter
 * Cumulative counters aggregated from the full SMT line.
 */
export function counterPayload(vu, seq) {
  return JSON.stringify({
    machine_id: "MES-GW-01",
    machine_type: "line_controller",
    vu: vu,
    seq: seq,
    boards_produced: seq,
    boards_failed: Math.floor(seq * 0.02),
    components_placed: seq * 200,
    feeder_picks_total: seq * 210,
    feeder_pick_errors: Math.floor(seq * 210 * 0.001),
    line_cycle_time_sec: 18.5 + Math.random() * 3,
    uptime_minutes: Math.floor(seq / 4),
    ts: new Date().toISOString(),
  });
}
