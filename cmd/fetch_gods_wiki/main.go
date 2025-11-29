package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/oliver/lan_smite_teams_bot/internal/smite"
)

const (
	wikiBaseURL = "https://wiki.smite2.com"
	apiEndpoint = "/api.php"
	userAgent   = "Smite2DiscordBot/1.0 (Go; +https://github.com/oliverrahner/lan_smite_teams_bot)"
)

// MediaWikiResponse represents the response from MediaWiki API
type MediaWikiResponse struct {
	Query struct {
		CategoryMembers []struct {
			PageID int    `json:"pageid"`
			Ns     int    `json:"ns"`
			Title  string `json:"title"`
		} `json:"categorymembers"`
	} `json:"query"`
	Continue struct {
		CmContinue string `json:"cmcontinue"`
		Continue   string `json:"continue"`
	} `json:"continue,omitempty"`
}

// ParseResponse represents the parse API response
type ParseResponse struct {
	Parse struct {
		Title    string `json:"title"`
		PageID   int    `json:"pageid"`
		Text     struct {
			Content string `json:"*"`
		} `json:"text"`
		Categories []struct {
			Category string `json:"*"`
		} `json:"categories"`
	} `json:"parse"`
}

// httpClient is a shared HTTP client with timeout
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	outputFile := "data/gods.json"
	if len(os.Args) > 1 {
		outputFile = os.Args[1]
	}

	log.Println("Fetching gods list from wiki...")
	godNames, err := fetchGodsFromCategory()
	if err != nil {
		log.Fatalf("Failed to fetch gods list: %v", err)
	}

	log.Printf("Found %d gods, fetching details...", len(godNames))

	var apiGods []smite.APIGod
	for i, godName := range godNames {
		log.Printf("[%d/%d] Fetching data for %s...", i+1, len(godNames), godName)
		
		god, err := fetchGodData(godName)
		if err != nil {
			log.Printf("Warning: Failed to fetch data for %s: %v", godName, err)
			continue
		}
		
		apiGods = append(apiGods, god)
		
		// Rate limiting - be respectful to the wiki
		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("Successfully fetched data for %d gods", len(apiGods))

	// Create API response structure
	response := smite.APIResponse{
		Data: apiGods,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Write to file
	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	log.Printf("Successfully wrote gods data to %s", outputFile)
	log.Printf("Total gods: %d", len(apiGods))
}

// fetchGodsFromCategory fetches all god names from the SMITE 2 gods category
func fetchGodsFromCategory() ([]string, error) {
	var allGods []string
	cmcontinue := ""

	for {
		params := url.Values{}
		params.Set("action", "query")
		params.Set("list", "categorymembers")
		params.Set("cmtitle", "Category:SMITE 2 gods")
		params.Set("cmlimit", "500")
		params.Set("format", "json")
		
		if cmcontinue != "" {
			params.Set("cmcontinue", cmcontinue)
		}

		apiURL := fmt.Sprintf("%s%s?%s", wikiBaseURL, apiEndpoint, params.Encode())
		
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch category: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
		}

		var mwResp MediaWikiResponse
		if err := json.NewDecoder(resp.Body).Decode(&mwResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		for _, member := range mwResp.Query.CategoryMembers {
			// Only include main namespace pages (ns=0)
			if member.Ns == 0 {
				allGods = append(allGods, member.Title)
			}
		}

		// Check if there are more pages
		if mwResp.Continue.CmContinue == "" {
			break
		}
		cmcontinue = mwResp.Continue.CmContinue
	}

	return allGods, nil
}

// fetchGodData fetches and parses data for a specific god
func fetchGodData(godName string) (smite.APIGod, error) {
	params := url.Values{}
	params.Set("action", "parse")
	params.Set("page", godName)
	params.Set("format", "json")
	params.Set("prop", "text|categories")

	apiURL := fmt.Sprintf("%s%s?%s", wikiBaseURL, apiEndpoint, params.Encode())
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return smite.APIGod{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return smite.APIGod{}, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return smite.APIGod{}, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var parseResp ParseResponse
	if err := json.NewDecoder(resp.Body).Decode(&parseResp); err != nil {
		return smite.APIGod{}, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse the HTML content
	htmlContent := parseResp.Parse.Text.Content
	
	// Extract data from HTML
	roles := extractRoles(htmlContent)
	abilities := extractAbilities(htmlContent, godName)
	pantheon := extractPantheon(htmlContent, parseResp.Parse.Categories)
	portraitURL := extractPortraitURL(htmlContent, godName)

	// Build APIGod structure
	apiGod := smite.APIGod{
		Attributes: smite.APIGodAttributes{
			Name:     godName,
			Roles:    buildAPIRoles(roles),
			Portrait: buildAPIPortrait(portraitURL),
			Ability:  abilities,
			Pantheon: buildAPIPantheon(pantheon),
		},
	}

	return apiGod, nil
}

// extractRoles extracts role information from HTML content
func extractRoles(html string) []string {
	var roles []string
	
	// Look for role information in various formats
	// MediaWiki infoboxes typically have role data
	rolePatterns := []string{
		`(?i)role[s]?\s*[=:]\s*([^<\n]+)`,
		`(?i)<th[^>]*>Roles?</th>\s*<td[^>]*>([^<]+)</td>`,
		`(?i)class="role[^"]*">([^<]+)</`,
	}

	for _, pattern := range rolePatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(html); len(matches) > 1 {
			roleText := matches[1]
			// Split by common delimiters
			for _, role := range strings.Split(roleText, ",") {
				role = strings.TrimSpace(role)
				// Convert to title case manually (strings.Title is deprecated)
				if len(role) > 0 {
					role = strings.ToUpper(role[:1]) + strings.ToLower(role[1:])
				}
				
				// Map to our role names
				switch {
				case strings.Contains(strings.ToLower(role), "mid"):
					roles = appendUnique(roles, "Mid")
				case strings.Contains(strings.ToLower(role), "support"):
					roles = appendUnique(roles, "Support")
				case strings.Contains(strings.ToLower(role), "jungle"):
					roles = appendUnique(roles, "Jungle")
				case strings.Contains(strings.ToLower(role), "carry") || strings.Contains(strings.ToLower(role), "adc"):
					roles = appendUnique(roles, "Carry")
				case strings.Contains(strings.ToLower(role), "solo"):
					roles = appendUnique(roles, "Solo")
				}
			}
			
			if len(roles) > 0 {
				break
			}
		}
	}

	// Default to mid if no roles found
	if len(roles) == 0 {
		roles = []string{"Mid"}
	}

	return roles
}

// extractAbilities extracts ability information from HTML
// Note: Uses regex for simplicity. For production use with complex HTML,
// consider using golang.org/x/net/html for more robust parsing.
func extractAbilities(html string, godName string) []smite.APIAbility {
	var abilities []smite.APIAbility
	
	// Look for ability sections
	// This pattern may not catch all cases but works for most wiki pages
	abilityPattern := regexp.MustCompile(`(?is)<h[23][^>]*>(.*?)</h[23]>.*?<p>(.*?)</p>`)
	matches := abilityPattern.FindAllStringSubmatch(html, -1)
	
	slots := []string{"Passive", "Position 1", "Position 2", "Position 3", "Position 4"}
	slotIdx := 0
	
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		
		title := stripHTML(match[1])
		description := stripHTML(match[2])
		
		// Check if this looks like an ability
		if isLikelyAbility(title, description) && slotIdx < len(slots) {
			ability := smite.APIAbility{
				Name:        title,
				Slot:        slots[slotIdx],
				Description: description,
				Icon: smite.APIIcon{
					Data: smite.APIIconData{
						Attributes: smite.APIImageAttributes{
							URL: fmt.Sprintf("https://wiki.smite2.com/images/abilities/%s_%d.png", 
								strings.ReplaceAll(godName, " ", "_"), slotIdx),
						},
					},
				},
			}
			abilities = append(abilities, ability)
			slotIdx++
		}
		
		if slotIdx >= len(slots) {
			break
		}
	}

	// If we didn't find enough abilities, create placeholders
	for len(abilities) < 5 {
		slot := slots[len(abilities)]
		abilities = append(abilities, smite.APIAbility{
			Name:        fmt.Sprintf("%s %s", godName, slot),
			Slot:        slot,
			Description: "Ability information not available from wiki.",
			Icon: smite.APIIcon{
				Data: smite.APIIconData{
					Attributes: smite.APIImageAttributes{
						URL: "",
					},
				},
			},
		})
	}

	return abilities
}

// extractPantheon extracts pantheon from HTML and categories
func extractPantheon(html string, categories []struct {
	Category string `json:"*"`
}) string {
	// Try to find pantheon in categories first
	pantheonPrefixes := []string{
		"Greek", "Roman", "Norse", "Egyptian", "Chinese", 
		"Hindu", "Japanese", "Celtic", "Mayan", "Maya",
		"Arthurian", "Voodoo", "Polynesian", "Slavic",
		"Yoruba", "Babylonian", "Mesopotamian", "Great Old Ones",
	}

	for _, cat := range categories {
		catName := cat.Category
		for _, prefix := range pantheonPrefixes {
			if strings.Contains(catName, prefix) {
				return prefix
			}
		}
	}

	// Try to find in HTML content
	pantheonPattern := regexp.MustCompile(`(?i)pantheon[s]?\s*[=:]\s*([^<\n]+)`)
	if matches := pantheonPattern.FindStringSubmatch(html); len(matches) > 1 {
		return strings.TrimSpace(stripHTML(matches[1]))
	}

	return "Unknown"
}

// extractPortraitURL tries to extract portrait image URL
// Note: Assumes double-quoted src attributes. May not work with all HTML variants.
func extractPortraitURL(html string, godName string) string {
	// Look for infobox images or main god image
	imgPattern := regexp.MustCompile(`(?i)<img[^>]*src="([^"]*(?:` + regexp.QuoteMeta(godName) + `|portrait|card)[^"]*)"`)
	if matches := imgPattern.FindStringSubmatch(html); len(matches) > 1 {
		imgURL := matches[1]
		// Make absolute URL
		if strings.HasPrefix(imgURL, "//") {
			return "https:" + imgURL
		} else if strings.HasPrefix(imgURL, "/") {
			return wikiBaseURL + imgURL
		}
		return imgURL
	}

	// Default placeholder
	return fmt.Sprintf("https://wiki.smite2.com/images/%s.png", strings.ReplaceAll(godName, " ", "_"))
}

// Helper functions

// stripHTML removes HTML tags from a string
// Note: This is a simple implementation using regex. For complex HTML with
// script tags, CDATA, or comments, consider using golang.org/x/net/html
func stripHTML(s string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	
	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	
	return strings.TrimSpace(s)
}

func isLikelyAbility(title, description string) bool {
	// Filter out common non-ability sections
	nonAbilities := []string{
		"lore", "background", "appearance", "personality",
		"skins", "videos", "tips", "trivia", "changelog",
		"see also", "references", "external links",
	}
	
	titleLower := strings.ToLower(title)
	for _, na := range nonAbilities {
		if strings.Contains(titleLower, na) {
			return false
		}
	}
	
	// Must have some description
	return len(description) > 10
}

func appendUnique(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}

func buildAPIRoles(roles []string) smite.APIRoles {
	var apiRoles []smite.APIRole
	for _, role := range roles {
		apiRoles = append(apiRoles, smite.APIRole{
			Attributes: smite.APIRoleAttributes{
				Name: role,
				Image: smite.APIRoleImage{
					Data: smite.APIRoleImageData{
						Attributes: smite.APIImageAttributes{
							URL: fmt.Sprintf("https://wiki.smite2.com/images/roles/%s.png", strings.ToLower(role)),
						},
					},
				},
			},
		})
	}
	return smite.APIRoles{Data: apiRoles}
}

func buildAPIPortrait(url string) smite.APIPortrait {
	return smite.APIPortrait{
		Data: smite.APIPortraitData{
			Attributes: smite.APIImageAttributes{
				URL: url,
			},
		},
	}
}

func buildAPIPantheon(name string) smite.APIPantheon {
	return smite.APIPantheon{
		Data: smite.APIPantheonData{
			Attributes: smite.APIPantheonAttributes{
				Name: name,
			},
		},
	}
}
