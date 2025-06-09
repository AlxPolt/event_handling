import asyncio
import os
import json
import logging
from datetime import datetime, timedelta

import nats
from influxdb_client import InfluxDBClient
from influxdb_client.client.write_api import SYNCHRONOUS
import pandas as pd

# --- Configuration ---
NATS_URL = os.getenv("NATS_URL", "nats://nats:4222")
NATS_SUBJECT_REQUEST = os.getenv("NATS_SUBJECT_REQUEST", "reader.query")
INFLUXDB_HOST = os.getenv("INFLUXDB_HOST", "http://influxdb:8086")
INFLUXDB_TOKEN = os.getenv("INFLUXDB_TOKEN")
INFLUXDB_ORG = os.getenv("INFLUXDB_ORG")
INFLUXDB_BUCKET = os.getenv("INFLUXDB_BUCKET")

# --- Logging ---
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

influx_client = None
query_api = None

async def run():
    global influx_client, query_api

    if not all([INFLUXDB_TOKEN, INFLUXDB_ORG, INFLUXDB_BUCKET]):
        logging.error("Missing InfluxDB credentials.")
        exit(1)

    influx_client = InfluxDBClient(url=INFLUXDB_HOST, token=INFLUXDB_TOKEN, org=INFLUXDB_ORG)
    query_api = influx_client.query_api()
    if not influx_client.ping():
        logging.critical("InfluxDB ping failed.")
        exit(1)

    nc = await nats.connect(NATS_URL)
    await nc.subscribe(NATS_SUBJECT_REQUEST, cb=request_handler)
    logging.info(f"Subscribed to NATS subject '{NATS_SUBJECT_REQUEST}'")

    try:
        while True:
            await asyncio.sleep(1)
    finally:
        await nc.close()
        influx_client.close()

async def request_handler(msg):
    try:
        request = json.loads(msg.data.decode())
        query_type = request.get("query_type")
        params = request.get("params", {})

        if query_type == "alerts_critical":
            response = await handle_alerts_critical(params)
        elif query_type == "device_health":
            response = await handle_device_health(params)
        elif query_type == "anomaly_temperature":
            response = await handle_anomaly_temperature(params)
        else:
            response = {"status": "error", "message": f"Unknown query_type: {query_type}"}

    except Exception as e:
        logging.exception("Error handling request")
        response = {"status": "error", "message": str(e)}

    await msg.respond(json.dumps(response).encode())

async def handle_alerts_critical(params):
    since_minutes = int(params.get("since_minutes", 15))
    min_crit = int(params.get("min_criticality", 8))
    start_range = f"-{since_minutes}m"

    flux = f'''
    from(bucket: "{INFLUXDB_BUCKET}")
      |> range(start: {start_range})
      |> filter(fn: (r) => r._measurement == "events")
      |> filter(fn: (r) => r.criticality_level >= {min_crit})
      |> sort(columns: ["_time"], desc: true)
      |> keep(columns: ["_time", "event_id", "event_type", "source_device", "criticality_level"])
    '''

    result = []
    tables = query_api.query(flux, org=INFLUXDB_ORG)
    for table in tables:
        for record in table.records:
            result.append({
                "time": record.get_time().isoformat(),
                "event_id": record["event_id"],
                "source_device": record["source_device"],
                "event_type": record["event_type"],
                "criticality": int(record["criticality_level"])
            })

    if result:
        df = pd.DataFrame(result)
        summary = df.groupby("source_device").size().reset_index(name="critical_event_count")
        summary_data = summary.to_dict(orient="records")
    else:
        summary_data = []

    return {"status": "success", "data": result, "summary": summary_data}

async def handle_device_health(params):
    device = params.get("source_device")
    if not device:
        return {"status": "error", "message": "source_device is required"}

    flux = f'''
    from(bucket: "{INFLUXDB_BUCKET}")
      |> range(start: -5m)
      |> filter(fn: (r) => r._measurement == "device_metrics")
      |> filter(fn: (r) => r.source_device == "{device}")
      |> last()
    '''

    tables = query_api.query(flux, org=INFLUXDB_ORG)
    health_status = "unknown"
    for table in tables:
        for record in table.records:
            if record.get_field() == "value" and isinstance(record.get_value(), (int, float)):
                value = record.get_value()
                if value > 90:
                    health_status = "critical"
                elif value > 70:
                    health_status = "warning"
                else:
                    health_status = "ok"
    return {"status": "success", "data": {"device": device, "health": health_status}}

async def handle_anomaly_temperature(params):
    device = params.get("source_device")
    threshold = float(params.get("threshold", 1.3))
    window = int(params.get("window_minutes", 20))

    flux = f'''
    from(bucket: "{INFLUXDB_BUCKET}")
      |> range(start: -{window}m)
      |> filter(fn: (r) => r._measurement == "device_metrics")
      |> filter(fn: (r) => r.source_device == "{device}" and r.metric_type == "temperature")
      |> sort(columns: ["_time"], desc: false)
    '''

    tables = query_api.query(flux, org=INFLUXDB_ORG)
    values = [record.get_value() for table in tables for record in table.records if record.get_field() == "value"]

    if len(values) < 2:
        return {"status": "success", "data": "not enough data"}

    initial = values[0]
    latest = values[-1]
    ratio = latest / initial if initial != 0 else 0

    anomaly = ratio >= threshold
    return {"status": "success", "data": {"device": device, "initial_temp": initial, "latest_temp": latest, "ratio": round(ratio, 2), "anomaly": anomaly}}

if __name__ == '__main__':
    asyncio.run(run())
