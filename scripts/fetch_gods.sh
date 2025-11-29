#!/bin/bash

# Script to fetch Smite 2 gods data from the Smite2 Wiki
# This replaces the previous API-based approach due to incomplete data

set -e

OUTPUT_FILE="data/gods.json"

echo "Fetching gods data from Smite2 Wiki..."
echo "Source: https://wiki.smite2.com/w/Category:SMITE_2_gods"

# Create data directory if it doesn't exist
mkdir -p data

# Build the wiki scraper
echo "Building wiki scraper..."
if ! go build -o bin/fetch_gods_wiki ./cmd/fetch_gods_wiki/; then
    echo "Error: Failed to build wiki scraper"
    exit 1
fi

# Run the wiki scraper
echo "Running wiki scraper..."
./bin/fetch_gods_wiki "$OUTPUT_FILE"

# Check if file was created and has content
if [ -s "$OUTPUT_FILE" ]; then
    echo ""
    echo "Successfully fetched gods data to $OUTPUT_FILE"
    echo "File size: $(du -h "$OUTPUT_FILE" | cut -f1)"
    
    # Validate URLs for the fetched gods
    echo ""
    echo "Validating god URLs..."
    echo "Building URL validator..."
    if go build -o bin/validate_urls ./cmd/validate_urls/; then
        echo "Running URL validation..."
        ./bin/validate_urls "$OUTPUT_FILE"
    else
        echo "Warning: Failed to build URL validator"
    fi
else
    echo "Error: Output file is empty or doesn't exist"
    exit 1
fi