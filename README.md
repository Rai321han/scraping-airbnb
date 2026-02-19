# Airbnb Web Scraper

A high-performance, resilient Airbnb property web scraper built with Go, featuring advanced stealth mode, retry mechanisms, and PostgreSQL persistence.


---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Installation & Run](#installation)
- [Configuration](#configuration)
- [Schema Inspection](#schema-inspection)
- [Architecture](#architecture)

---

## Overview

This project is a production-ready web scraper for Airbnb listings that:
- Extracts property data with complete details (title, price, location, rating, description)
- Stores data in PostgreSQL with batch persistence
- Implements sophisticated retry logic with exponential backoff
- Includes stealth mode to avoid detection (random delays, user agents, rate limiting)
- Provides comprehensive terminal insights and analytics after scraping

---

## Features

### Core Scraping
- Multi-threaded concurrent scraping with worker pools
- Property data extraction: ID, platform, title, price, location, rating, description
- Pagination support for multi-page results
- Per-property logging with numbered sequential tracking

### Resilience & Error Handling
- Exponential backoff retry mechanism (configurable: 1-10+ attempts)
- Context-aware timeout handling
- Detailed retry attempt logging (start, success, failure, all attempts failed)
- Graceful error recovery

### Stealth Mode
- Random request delays (configurable 500ms-2s default)
- Random user agent rotation (8+ realistic agents)
- Request rate limiting (configurable: 2 req/sec default)
- All tuning parameters in config, not hardcoded

### Data Persistence
- PostgreSQL batch insert with transactions
- ON CONFLICT handling for duplicate URLs
- Location-based indexing for fast queries
- Auto-schema creation on startup

### Insights & Analytics
- Total listings, platform breakdown, price analytics
- Most expensive property details
- Listings per location (parsed city extraction)
- Top 5 highest-rated properties
- Clean formatted terminal output
![Analytics Screenshot](screenshot.png)

---

## Project Structure

```
scraping-airbnb/
├── cmd/
│   └── main.go                    # Application entry point
├── config/
│   └── settings.go                # Configuration structs & defaults
├── db/
│   └── init.sql                   # Database schema initialization
├── internal/
│   └── domain/
│       ├── repository.go          # Repository interface
│       ├── postgres_repository.go # PostgreSQL implementation
│       ├── csv_repository.go      # CSV implementation (optional)
│       └── scraper.go             # Scraper interface
├── models/
│   └── property.go                # Property data model
├── scraper/
│   ├── browser.go                 # Browser lifecycle management
│   └── airbnb/
│       ├── chromedp_scraper.go    # Main scraper with stealth mode
│       └── script.go              # JavaScript extract scripts
├── service/
│   └── scraper_service.go         # Service layer with retry & insights
├── utils/
│   └── utils.go                   # Utility functions (parsing, etc.)
├── .env                           # Environment variables (not in git)
├── docker-compose.yml             # PostgreSQL container setup
├── go.mod                         # Go module dependencies
├── go.sum                         # Dependency checksums
├── .gitignore                     # Git ignore rules
└── README.md                      # This file
```

---

## Prerequisites

- **Go 1.25+**
- **Git** - For cloning the repository

---

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/Rai321han/scraping-airbnb.git
cd scraping-airbnb
```

### 2. Install Go Dependencies

```bash
go mod download
```

### 3. Set Up Environment

Create a `.env` file in the project root:

```bash
PG_DSN="postgres://db_user:db_password@localhost:5432/db_name?sslmode=disable"
```

### 4. Database Setup Using Docker Compose

```bash
docker-compose up -d
```

This creates a PostgreSQL container with:
- User: `dbuser`
- Password: `db_password`
- Database: `db_name`
- Port: `5432`

The schema is automatically created on first run via `db/init.sql`.

### 5. Running the Scraper

### Build the Project

```bash
go build -o scraper_executable ./cmd
```

### Run the Scraper

```bash
# Or using compiled binary
./scraper_executable
```
---

## Schema Inspection

```bash
# Connect to the database
docker exec -it db-postgres psql -U db_user -d db_name


# Inside psql:
\dt                                    # List tables
SELECT * FROM properties LIMIT 10;      # View data
SELECT COUNT(*) FROM properties;        # Count rows
SELECT * FROM properties LIMIT 10;    # View via view
```


## Configuration

Edit `config/settings.go` to customize scraper behavior:

### Retry Configuration
```go
Retry: RetryConfig{
    MaxRetries:     3,                  // Max retry attempts
    InitialBackoff: 2 * time.Second,   // First backoff duration
    MaxBackoff:     10 * time.Second,  // Maximum backoff cap
}
```

### Stealth Mode Configuration
```go
Stealth: StealthConfig{
    RandomDelayEnabled:      true,           // Random delays between requests
    RandomDelayMin:          500 * time.Millisecond,  // Min delay
    RandomDelayMax:          2 * time.Second,         // Max delay
    RandomUserAgentEnabled:  true,           // Random user agent rotation
    MaxRequestsPerSecond:    2.0,            // Rate limiting
}
```

### Timing Configuration
```go
Timing: TimingConfig{
    PageLoadWait:     3 * time.Second,
    ScrollStepDelay:  300 * time.Millisecond,
    ScrollBottomWait: 3 * time.Second,
    AfterScrollWait:  3 * time.Second,
    ProductPageWait:  4 * time.Second,
    ProductTimeout:   50 * time.Second,
}
```

---

After successful scraping, the tool displays:

1. **Summary Statistics** - Total count, platform breakdown, price range
2. **Most Expensive Property** - Title, price, location details
3. **Listings per Location** - Sorted by frequency (city extracted from address)
4. **Top 5 Highest Rated** - Property name and rating

Database records are automatically persisted with:
- Auto-increment ID
- Platform (Airbnb)
- Property details (title, price, location, rating)
- Description
- URL as unique key (prevents duplicates)
- Location index for fast queries

---

## Architecture

### Component Overview

```
┌─────────────────────────────────────────┐
│         cmd/main.go                     │ Entry point, .env loading
└─────────────┬───────────────────────────┘
              │
              ├──────────────────────┐
              ↓                      ↓
    ┌──────────────────┐  ┌──────────────────┐
    │ Config Package   │  │ ScraperService   │ Retry orchestration
    │ (settings.go)    │  │ (with retries)   │ & insights
    └──────────────────┘  └──────┬───────────┘
                                  │
              ┌───────────────────┴───────────────────┐
              ↓                                       ↓
    ┌──────────────────────┐            ┌──────────────────────┐
    │ ChromedpScraper      │            │ PostgresRepository   │
    │ (stealth mode)       │            │ (batch persistence)  │
    │ + rate limiting      │            │ + transactions       │
    │ + random delays      │            └──────────────────────┘
    │ + user agent pool    │                      ↑
    │ + retry logic        │                      │
    └──────────┬───────────┘                      │
               │                                   │
               └───────────────────┬───────────────┘
                                   ↓
                        ┌──────────────────────┐
                        │ PostgreSQL Database  │
                        │ (via lib/pq)         │
                        └──────────────────────┘
```

### Data Flow

1. **Config Loaded** → User agent pool, retry settings, stealth params
2. **Scraper Initialized** → Rate limiter ticker started, user agents cached
3. **Location Pages Scraped** → With random delays & user agents
4. **Property URLs Collected** → From pagination
5. **Worker Pool Extracts** → Each property with per-request delays
6. **Batch Insert** → All properties in single transaction
7. **Insights Generated** → Analytics printed to terminal
8. **Database Queried** → For final summary stats
