# AFS Engine (Active Flight Schedule Generator)

## Overview

The AFS Engine is a production-ready **Go-based** system that automatically generates Active Flight Schedules (AFS) from Master Flight Schedules (MFS) and delivers them to downstream systems via XML API.

### Key Features

- ✅ **Automated Daily Processing** - Cron-based scheduler runs at midnight
- ✅ **MongoDB Replica Set** - High availability with automatic failover
- ✅ **Idempotent Operations** - Safe to re-run without duplicates
- ✅ **Retry Logic** - Exponential backoff for failed deliveries
- ✅ **XML Transformation** - Converts AFS to SSIM-compliant XML
- ✅ **Audit Trail** - Complete delivery tracking and archival
- ✅ **RESTful API** - Manual triggers and monitoring endpoints
- ✅ **Docker-based** - Complete containerized deployment
- ✅ **No Internet Required** - All services run offline

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  SCHEDULER (Cron)                                       │
│  ├─ Daily Job: 00:00 - Generate & Deliver              │
│  └─ Retry Job: Every 15 min - Retry failed             │
└─────────────────┬───────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────┐
│  PHASE 1-2: AFS GENERATION                              │
│  ├─ Query MFS (date & frequency match)                  │
│  ├─ Expand to AFS records (one per leg)                 │
│  └─ UPSERT to active_flights collection                 │
└─────────────────┬───────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────┐
│  PHASE 3-4: API DELIVERY                                │
│  ├─ Query pending AFS records                           │
│  ├─ Transform to XML (batches of 100)                   │
│  ├─ POST to API with retry logic                        │
│  ├─ Update delivery status (SENT/FAILED)                │
│  └─ Archive XML (optional)                              │
└─────────────────────────────────────────────────────────┘
```

### Technology Stack

| Component | Technology |
|-----------|-----------|
| **Language** | Go 1.21+ |
| **Database** | MongoDB 7.0 (Replica Set) |
| **Scheduler** | robfig/cron |
| **HTTP Framework** | Gin |
| **Logging** | Logrus (structured JSON) |
| **Containerization** | Docker + Docker Compose |

---

## Quick Start

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 4GB RAM minimum
- **No internet connection required**

### 1. Clone/Extract Project

```bash
cd afs-engine-go
```

### 2. Build & Start Services

```bash
# Build and start all services
docker-compose up -d --build

# View logs
docker-compose logs -f afs-engine

# Check service status
docker-compose ps
```

### 3. Verify Services

```bash
# Check AFS Engine health
curl http://localhost:3000/health

# Check MongoDB health
docker exec afs-mongodb-primary mongosh --eval "rs.status()"

# Check API Receiver
curl http://localhost:3001/health

# Access MongoDB UI
# http://localhost:8081 (admin/admin)
```

### 4. Seed Sample Data

```bash
# Option 1: Using your uploaded JSON
docker exec -i afs-mongodb-primary mongosh \
  -u admin -p afs_secure_pass_2026 \
  --authenticationDatabase admin afs_db \
  --eval "db.master_flights.insertMany($(cat /path/to/your/mfs.json))"

# Option 2: Use seed script
cd scripts
go run seed.go
```

### 5. Manual Test Run

```bash
# Trigger manual AFS generation for today
curl -X POST http://localhost:3000/api/generate \
  -H "Content-Type: application/json" \
  -d '{"date":"2026-01-30"}'

# Check generated AFS records
curl http://localhost:3000/api/afs?date=2026-01-30

# View statistics
curl http://localhost:3000/api/stats
```

---

## Configuration

### Environment Variables

Edit `.env` or `docker-compose.yml`:

```bash
# Scheduler - When to run daily job
CRON_SCHEDULE=0 0 * * *    # Midnight (default)
# CRON_SCHEDULE=0 2 * * *  # 2 AM

# Processing
BATCH_SIZE=100             # Flights per XML batch
MAX_WORKERS=4              # Parallel workers

# Retry
RETRY_ATTEMPTS=3           # Max retry attempts
RETRY_DELAY_MS=60s         # Initial retry delay

# Storage
AFS_TTL_DAYS=7             # Auto-delete after N days
ENABLE_XML_ARCHIVE=true    # Save XML files
```

---

## API Endpoints

### Health Check
```bash
GET /health
```
Response:
```json
{
  "status": "healthy",
  "service": "afs-engine",
  "time": "2026-01-30T00:00:00Z"
}
```

### Manual Generation
```bash
POST /api/generate
Content-Type: application/json

{
  "date": "2026-01-30"
}
```

### Get AFS Records
```bash
# All records for date
GET /api/afs?date=2026-01-30

# By status
GET /api/afs?date=2026-01-30&status=PENDING
GET /api/afs?date=2026-01-30&status=SENT
GET /api/afs?date=2026-01-30&status=FAILED
```

### Get Statistics
```bash
GET /api/stats
```
Response:
```json
{
  "date": "2026-01-30",
  "total": 150,
  "sent": 145,
  "pending": 0,
  "failed": 5,
  "successRate": 96.67
}
```

### Manual Retry
```bash
POST /api/retry
```

---

## Data Flow

### 1. Master Flight Schedule (MFS) Input

```json
{
  "flightNo": "MH 0085",
  "startDate": "2026-01-01",
  "endDate": "2026-04-30",
  "frequency": "1234567",  // Daily
  "scheduleStatus": "ACTIVE",
  "stations": [
    {
      "DepartureStation": "KUL",
      "std": "0855",
      "ArrivalStation": "SIN",
      "sta": "0955"
    },
    {
      "DepartureStation": "SIN",
      "std": "1055",
      "ArrivalStation": "LHR",
      "sta": "1735"
    }
  ]
}
```

### 2. Active Flight Schedule (AFS) Output

```json
{
  "_id": "MH 0085_2026-01-30_LEG1_KUL-SIN",
  "flightNo": "MH 0085",
  "flightDate": "2026-01-30",
  "departureStation": "KUL",
  "arrivalStation": "SIN",
  "std": "0855",
  "sta": "0955",
  "deliveryStatus": "SENT",
  "deliveredAt": "2026-01-30T00:45:23Z",
  "sentXMLBatchId": "batch_2026-01-30T00-45-23_001"
}
```

### 3. XML API Payload

```xml
<?xml version="1.0" encoding="UTF-8"?>
<FlightSchedule version="1.0" batchId="batch_001" recordCount="100">
  <Flights>
    <Flight id="MH 0085_2026-01-30_LEG1_KUL-SIN">
      <FlightIdentification>
        <FlightNumber>MH 0085</FlightNumber>
        <FlightDate>2026-01-30</FlightDate>
      </FlightIdentification>
      <Route>
        <DepartureStation>
          <Airport>KUL</Airport>
          <ScheduledTime>0855</ScheduledTime>
        </DepartureStation>
        <ArrivalStation>
          <Airport>SIN</Airport>
          <ScheduledTime>0955</ScheduledTime>
        </ArrivalStation>
      </Route>
    </Flight>
  </Flights>
</FlightSchedule>
```

---

## Monitoring

### View Logs

```bash
# AFS Engine logs
docker-compose logs -f afs-engine

# MongoDB logs
docker-compose logs -f mongodb-primary

# API Receiver logs
docker-compose logs -f api-receiver

# All services
docker-compose logs -f
```

### Log Files

Logs are persisted in `./logs/`:

```bash
ls -lh logs/
# combined.log - All logs
# error.log - Errors only
```

### Archived XML Files

```bash
ls -lh archive/2026/01/30/
# batch_2026-01-30T00-45-23_001.xml
# batch_2026-01-30T00-45-23_001_manifest.json
```

### MongoDB Web UI

Access at `http://localhost:8081`
- Username: `admin`
- Password: `admin`

Collections to monitor:
- `master_flights` - Input MFS data
- `active_flights` - Generated AFS records
- `jobs` - Scheduled job status

---

## Failure Scenarios & Recovery

### Scenario 1: MongoDB Primary Fails

**What Happens:**
- Replica set auto-elects secondary as new primary
- Downtime: 10-30 seconds
- AFS Engine reconnects automatically

**Recovery:**
```bash
# Check replica set status
docker exec afs-mongodb-primary mongosh -u admin -p afs_secure_pass_2026 \
  --authenticationDatabase admin --eval "rs.status()"

# Restart failed node
docker-compose restart mongodb-primary
```

### Scenario 2: API is Down

**What Happens:**
- Delivery fails after 3 retry attempts
- Records marked as `FAILED` in database
- Retry job (every 15 min) automatically retries

**Recovery:**
```bash
# Check failed records
curl http://localhost:3000/api/afs?status=FAILED

# Manual retry when API is back
curl -X POST http://localhost:3000/api/retry
```

### Scenario 3: Job Crashes Mid-Processing

**What Happens:**
- Database records remain in `PENDING` state
- Next cron run or manual trigger will reprocess
- Idempotent UPSERT prevents duplicates

**Recovery:**
```bash
# Check what's pending
curl http://localhost:3000/api/stats

# Trigger manual run
curl -X POST http://localhost:3000/api/generate \
  -d '{"date":"2026-01-30"}'
```

### Scenario 4: Need to Regenerate Specific Date

**Solution:**
```bash
# Regenerate AFS for any date
curl -X POST http://localhost:3000/api/generate \
  -H "Content-Type: application/json" \
  -d '{"date":"2026-01-25"}'
```

---

## Production Deployment

### Security Hardening

1. **Change MongoDB Password**
```yaml
# docker-compose.yml
MONGO_INITDB_ROOT_PASSWORD: YOUR_STRONG_PASSWORD_HERE
```

2. **Remove Mongo Express** (production)
```bash
# Comment out mongo-express service in docker-compose.yml
```

3. **Enable TLS** (if needed)
```bash
# Add to MongoDB configuration
--tlsMode requireTLS
--tlsCertificateKeyFile /path/to/cert.pem
```

### Resource Limits

```yaml
# docker-compose.yml - Add to afs-engine service
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 2G
    reservations:
      cpus: '1'
      memory: 1G
```

### Backup Strategy

```bash
# Daily MongoDB backup (add to cron)
0 1 * * * docker exec afs-mongodb-primary mongodump \
  -u admin -p PASSWORD --authenticationDatabase admin \
  -o /backup/$(date +\%Y\%m\%d)
```

---

## Troubleshooting

### AFS Engine Won't Start

```bash
# Check logs
docker-compose logs afs-engine

# Common issues:
# 1. MongoDB not ready - wait 30s after starting
# 2. Connection string wrong - check MONGO_URI
# 3. Replica set not initialized - check mongo-setup logs
```

### No AFS Records Generated

```bash
# 1. Check if MFS data exists
docker exec afs-mongodb-primary mongosh -u admin -p afs_secure_pass_2026 \
  --authenticationDatabase admin afs_db \
  --eval "db.master_flights.countDocuments()"

# 2. Check date range
# MFS startDate <= today <= endDate

# 3. Check frequency pattern
# Today's day-of-week must match frequency
```

### High Memory Usage

```bash
# Reduce batch size
BATCH_SIZE=50

# Reduce max workers
MAX_WORKERS=2
```

---

## Development

### Local Development Setup

```bash
# Install Go 1.21+
# Install dependencies
go mod download

# Run locally (without Docker)
export MONGO_URI="mongodb://admin:afs_secure_pass_2026@localhost:27017/afs_db?authSource=admin"
export API_ENDPOINT="http://localhost:3001/api/schedules"
go run cmd/afs-engine/main.go
```

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test ./internal/services/... -v
```

### Building Binary

```bash
# Build for Linux
CGO_ENABLED=0 GOOS=linux go build -o afs-engine cmd/afs-engine/main.go

# Build for current OS
go build -o afs-engine cmd/afs-engine/main.go
```

---

## Maintenance

### Cleanup Old Data

```bash
# TTL index auto-deletes after 7 days
# Manual cleanup if needed:
docker exec afs-mongodb-primary mongosh -u admin -p afs_secure_pass_2026 \
  --authenticationDatabase admin afs_db \
  --eval "db.active_flights.deleteMany({flightDate: {\$lt: new Date('2026-01-01')}})"
```

### Update MFS Data

```bash
# Import new MFS records
docker exec -i afs-mongodb-primary mongoimport \
  -u admin -p afs_secure_pass_2026 \
  --authenticationDatabase admin \
  -d afs_db -c master_flights \
  --jsonArray < new_mfs_data.json
```

---

## Support

### Logs Location

- Application logs: `./logs/`
- Docker logs: `docker-compose logs`
- MongoDB logs: Inside container `/var/log/mongodb/`

### Metrics to Monitor

- AFS generation success rate (should be >99%)
- API delivery success rate (should be >95%)
- Database replica set health
- Disk space (archive directory grows daily)

---

## License

MIT License - Free for commercial use

---

## Contact

For issues or questions, contact MH Airlines IT Operations Team.
