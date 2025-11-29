# Smite2 Wiki God Data Scraper

This tool scrapes god information from the Smite2 Wiki (https://wiki.smite2.com) to populate the bot's god database.

## Purpose

The bot previously relied on the Smite2 API, but that data source proved incomplete. This scraper fetches god data directly from the wiki using the MediaWiki API.

## What It Does

1. Fetches a list of all gods from the `Category:SMITE 2 gods` page
2. For each god, retrieves their wiki page content
3. Parses the HTML to extract:
   - God name
   - Roles (Mid, Support, Jungle, Carry, Solo)
   - Pantheon
   - Abilities (name, description, slot)
   - Portrait image URL
4. Outputs the data in the same JSON format used by the previous API

## Usage

### Via Script (Recommended)

```bash
cd /path/to/lan_smite_teams_bot
./scripts/fetch_gods.sh
```

This will:
- Build the scraper
- Fetch all god data from the wiki
- Save to `data/gods.json`
- Run URL validation

### Directly

```bash
# Build
go build -o bin/fetch_gods_wiki ./cmd/fetch_gods_wiki/

# Run (output to default location: data/gods.json)
./bin/fetch_gods_wiki

# Run with custom output file
./bin/fetch_gods_wiki /path/to/output.json
```

## How It Works

### MediaWiki API

The scraper uses the MediaWiki API endpoints:

1. **List gods**: `/api.php?action=query&list=categorymembers&cmtitle=Category:SMITE 2 gods`
   - Returns all pages in the SMITE 2 gods category

2. **Parse god page**: `/api.php?action=parse&page={god_name}&format=json`
   - Returns the rendered HTML and metadata for a god's page

### Data Extraction

The scraper parses the HTML content to extract:

- **Roles**: Looks for role information in infoboxes and HTML tables
- **Abilities**: Parses heading and paragraph tags to find ability descriptions
- **Pantheon**: Extracted from page categories and infobox data
- **Images**: Finds portrait URLs in img tags

### Rate Limiting

The scraper includes a 500ms delay between requests to avoid overloading the wiki server.

## Output Format

The scraper outputs data in the same JSON structure as the previous API:

```json
{
  "data": [
    {
      "attributes": {
        "Name": "God Name",
        "roles": {
          "data": [
            {
              "attributes": {
                "Name": "Mid",
                "Image": {
                  "data": {
                    "attributes": {
                      "url": "https://wiki.smite2.com/images/roles/mid.png"
                    }
                  }
                }
              }
            }
          ]
        },
        "Portrait": {
          "data": {
            "attributes": {
              "url": "https://wiki.smite2.com/..."
            }
          }
        },
        "Ability": [
          {
            "Name": "Ability Name",
            "Slot": "Passive",
            "Description": "...",
            "Icon": {
              "data": {
                "attributes": {
                  "url": "..."
                }
              }
            }
          }
        ],
        "pantheon": {
          "data": {
            "attributes": {
              "Name": "Greek"
            }
          }
        }
      }
    }
  ]
}
```

This ensures compatibility with the existing `gods.go` interface.

## Limitations

- **Incomplete data**: Some god pages may not have complete information, especially for newer gods
- **HTML parsing**: The scraper uses regex-based HTML parsing for simplicity. This works for most wiki pages but may be fragile if wiki formatting changes significantly. For more robust parsing, consider migrating to `golang.org/x/net/html`.
- **Image URLs**: Some image URLs may be placeholder URLs if not found in the HTML. The scraper assumes double-quoted src attributes in img tags.
- **Abilities**: The ability extraction pattern may not catch all cases, especially if there are nested HTML elements or non-standard formatting
- **Character encoding**: HTML entities are handled for common cases but may not cover all Unicode scenarios

## Troubleshooting

### "Failed to fetch gods list"

- Check your internet connection
- Verify that https://wiki.smite2.com is accessible
- Check if the wiki API is available: https://wiki.smite2.com/api.php

### "Failed to fetch data for [god]"

- Some god pages may not exist or may have different names
- The scraper will continue with other gods and log warnings

### Empty or incomplete ability data

- Some god pages may not have properly formatted ability sections
- The scraper will create placeholder entries to maintain structure

## Future Improvements

Potential enhancements:

1. **Use proper HTML parser**: Replace regex-based parsing with `golang.org/x/net/html` for more robust and reliable extraction
2. **Caching**: Cache results to avoid re-fetching unchanged pages
3. **Incremental updates**: Only fetch new/modified gods instead of re-scraping everything
4. **Improved ability extraction**: Use more sophisticated parsing logic to handle various wiki page formats
5. **Add tests**: Create integration tests for the scraper functionality (requires wiki access)
6. **Better error recovery**: Handle transient network errors with retries and exponential backoff
7. **Structured data extraction**: Look for JSON-LD or microdata in wiki pages for more reliable data extraction
