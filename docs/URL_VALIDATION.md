# URL Validation Tool

This tool validates that the generated URLs for gods in the Smite 2 Discord bot are accessible.

## Usage

### Automatic Validation
The URL validator runs automatically after fetching gods data:
```bash
./scripts/fetch_gods.sh
```

### Manual Validation
```bash
# Build the validator
make validate-urls

# Or build and run manually
go build -o bin/validate_urls ./cmd/validate_urls/
./bin/validate_urls data/gods.json
```

## What it checks

1. **Smite2.com URLs**: Official god pages (format: `https://www.smite2.com/gods/god-name/`)
2. **Wiki URLs**: Community wiki pages (format: `https://wiki.smite2.com/w/God_Name`)

## Output

The tool provides a detailed table showing:
- God name
- Generated Smite2.com URL
- Generated Wiki URL  
- Status for each URL (✅ valid, ❌ invalid)
- Summary statistics

## URL Generation Rules

### Smite2.com URLs
- Convert to lowercase
- Replace spaces with hyphens
- Remove apostrophes, periods, commas
- Example: "Hun Batz" → "hun-batz"

### Wiki URLs  
- Replace spaces with underscores
- Preserve original capitalization
- Example: "Hun Batz" → "Hun_Batz"

## Troubleshooting

- **TLS Certificate Errors**: May occur in some environments but URLs typically work in production
- **404 Errors**: God name might not match the expected URL format
- **Connection Timeouts**: Check internet connectivity

The tool includes a 100ms delay between requests to be respectful to the target servers.