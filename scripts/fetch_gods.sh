#!/bin/bash

# Simple script to fetch Smite 2 gods data from the API

set -e

# Base API URL and configuration
BASE_URL="https://webcms.hirezstudios.com/smite2/api/gods/"
LANGUAGE="en-US"
PAGE="1"
PAGE_SIZE="100"

# Populate array - one entry per line for easy maintenance
POPULATE_FIELDS=(
    "Ability"
    "Ability.YouTubeLink"
    "Ability.Buffs"
    "Ability.Icon"
    "Ability.Buffs.Icon"
    "difficulty"
    "HeaderImage"
    "pantheon"
    "pantheon.Image"
    "pantheon.localizations"
    "Portrait"
    "roles"
    "roles.Image"
    "roles.localizations"
    "Skin"
    "Skin.Image"
    "type"
    "CommunityGuide"
    "CommunityGuide.previewThumbnail"
)

OUTPUT_FILE="data/gods.json"

# Build the populate parameters
POPULATE_PARAMS=""
for i in "${!POPULATE_FIELDS[@]}"; do
    POPULATE_PARAMS="${POPULATE_PARAMS}&populate%5B${i}%5D=${POPULATE_FIELDS[$i]}"
done

# Build the complete API URL
API_URL="${BASE_URL}?lng=${LANGUAGE}&pagination%5Bpage%5D=${PAGE}&pagination%5BpageSize%5D=${PAGE_SIZE}${POPULATE_PARAMS}"

echo "Fetching gods data from Smite 2 API..."

# Create data directory if it doesn't exist
mkdir -p data

# Fetch data with curl
echo "Downloading from: $API_URL"
curl -f --retry 3 --retry-delay 2 "$API_URL" -o "$OUTPUT_FILE"

# Check if file was created and has content
if [ -s "$OUTPUT_FILE" ]; then
    echo "Successfully downloaded gods data to $OUTPUT_FILE"
    echo "File size: $(du -h "$OUTPUT_FILE" | cut -f1)"
    
    # Validate URLs for the downloaded gods
    echo ""
    echo "Validating god URLs..."
    if command -v go >/dev/null 2>&1; then
        # Build and run the URL validator
        echo "Building URL validator..."
        if go build -o bin/validate_urls ./cmd/validate_urls/; then
            echo "Running URL validation..."
            ./bin/validate_urls "$OUTPUT_FILE"
        else
            echo "Warning: Failed to build URL validator"
        fi
    else
        echo "Warning: Go not found, skipping URL validation"
    fi
else
    echo "Error: Downloaded file is empty or doesn't exist"
    exit 1
fi