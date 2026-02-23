#!/bin/bash
set -e  # Exit on any error

# -----------------------------
# Colors for output
# -----------------------------
GREEN="\033[0;32m"
YELLOW="\033[1;33m"
NC="\033[0m" # No Color

# -----------------------------
# 1. Install Go dependencies
# -----------------------------
echo -e "${GREEN}Step 1: Installing Go dependencies...${NC}"
go mod download

# -----------------------------
# 2. Set up environment
# -----------------------------
echo -e "${GREEN}Step 2: Setting up environment variables...${NC}"
ENV_FILE=".env"
if [ ! -f "$ENV_FILE" ]; then
    cat <<EOL > $ENV_FILE
PG_DSN="postgres://db_user:db_password@localhost:5432/db_name?sslmode=disable"
SCRAPER_URL="https://airbnb.com/"
EOL
    echo -e "${GREEN}.env file created.${NC}"
else
    echo -e "${YELLOW}.env file already exists. Skipping creation.${NC}"
fi

# -----------------------------
# 3. Start Database using Docker Compose
# -----------------------------
echo -e "${GREEN}Step 3: Starting PostgreSQL container...${NC}"
docker-compose up -d

# Wait for DB to be ready
echo -e "${GREEN}Waiting for PostgreSQL to be ready...${NC}"
sleep 10

# -----------------------------
# 4. Build the scraper
# -----------------------------
echo -e "${GREEN}Step 4: Building the scraper...${NC}"
go build -o scraper_executable ./cmd

# -----------------------------
# 5. Run the scraper
# -----------------------------
echo -e "${GREEN}Step 5: Running the scraper...${NC}"
./scraper_executable

echo -e "${GREEN}✅ Scraper ran successfully!${NC}"