# MCP Tools

The NASA MCP server provides 13 tools. Each tool accepts JSON parameters and returns formatted text.

All requests use JSON-RPC 2.0 over stdin. Every request must include `"jsonrpc": "2.0"` and an `"id"` field; responses echo the same `"id"` back.

## nasa_apod

Get NASA's Astronomy Picture of the Day.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date` | string | No | Date in YYYY-MM-DD format (default: today) |
| `count` | integer | No | Number of random APOD entries to return |

The `date` and `count` parameters cannot be used together. Providing both returns an error.

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_apod", "arguments": {}}}
```

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_apod", "arguments": {"date": "2024-01-01"}}}
```

## nasa_apod_range

Get Astronomy Picture of the Day for a date range.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start_date` | string | Yes | Start date in YYYY-MM-DD format |
| `end_date` | string | Yes | End date in YYYY-MM-DD format |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_apod_range", "arguments": {"start_date": "2024-01-01", "end_date": "2024-01-07"}}}
```

## nasa_neo_feed

Get near-Earth objects for a date range (max 7 days).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start_date` | string | Yes | Start date in YYYY-MM-DD format |
| `end_date` | string | No | End date in YYYY-MM-DD format (default: start + 6 days) |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_neo_feed", "arguments": {"start_date": "2024-06-01"}}}
```

## nasa_neo_get

Get details for a specific near-Earth asteroid by ID.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `asteroid_id` | string | Yes | The asteroid SPK-ID (e.g., "3542519") |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_neo_get", "arguments": {"asteroid_id": "3542519"}}}
```

## nasa_donki_cme

Get coronal mass ejection events from DONKI.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start_date` | string | No | Start date YYYY-MM-DD (default: 30 days ago) |
| `end_date` | string | No | End date YYYY-MM-DD (default: today) |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_donki_cme", "arguments": {}}}
```

## nasa_donki_gst

Get geomagnetic storm events from DONKI.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start_date` | string | No | Start date YYYY-MM-DD (default: 30 days ago) |
| `end_date` | string | No | End date YYYY-MM-DD (default: today) |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_donki_gst", "arguments": {}}}
```

## nasa_donki_flr

Get solar flare events from DONKI.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start_date` | string | No | Start date YYYY-MM-DD (default: 30 days ago) |
| `end_date` | string | No | End date YYYY-MM-DD (default: today) |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_donki_flr", "arguments": {}}}
```

## nasa_donki_sep

Get solar energetic particle events from DONKI.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start_date` | string | No | Start date YYYY-MM-DD (default: 30 days ago) |
| `end_date` | string | No | End date YYYY-MM-DD (default: today) |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_donki_sep", "arguments": {}}}
```

## nasa_donki_notifications

Get DONKI space weather notifications.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start_date` | string | No | Start date YYYY-MM-DD (default: 30 days ago) |
| `end_date` | string | No | End date YYYY-MM-DD (default: today) |
| `type` | string | No | Filter: all, FLR, SEP, CME, IPS, MPC, GST, RBE, report |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_donki_notifications", "arguments": {"type": "CME"}}}
```

## nasa_images_search

Search the NASA Image and Video Library.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | string | Yes | Search query (e.g., "mars rover", "apollo 11") |
| `media_type` | string | No | Filter: image, video, audio |
| `year_start` | integer | No | Results from this year or later |
| `year_end` | integer | No | Results up to this year |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_images_search", "arguments": {"q": "apollo 11", "media_type": "image"}}}
```

## nasa_ssd_cad

Get asteroid/comet close approach data from JPL SSD/CNEOS.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_min` | string | No | Minimum close-approach date YYYY-MM-DD |
| `date_max` | string | No | Maximum close-approach date YYYY-MM-DD |
| `dist_max` | string | No | Maximum approach distance in AU (e.g., "0.05") |
| `body` | string | No | Filter by body (e.g., "Earth", "Mars") |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_ssd_cad", "arguments": {"dist_max": "0.05", "body": "Earth"}}}
```

## nasa_ssd_fireball

Get fireball/bolide event data from JPL SSD/CNEOS.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_min` | string | No | Minimum event date YYYY-MM-DD |
| `date_max` | string | No | Maximum event date YYYY-MM-DD |
| `energy_min` | number | No | Minimum energy in kilotons (client-side filter) |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_ssd_fireball", "arguments": {"date_min": "2024-01-01"}}}
```

## nasa_eonet_events

Get natural events from the Earth Observatory Natural Event Tracker.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `status` | string | No | Filter: "open" or "closed" |
| `limit` | integer | No | Maximum number of events |
| `days` | integer | No | Events within last N days |
| `category` | string | No | Filter by category identifier |

```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "nasa_eonet_events", "arguments": {"status": "open", "limit": 5}}}
```
