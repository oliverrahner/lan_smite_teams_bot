package smite

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

// Role represents a Smite 2 conquest role
type Role string

const (
	RoleMid     Role = "Mid"
	RoleSupport Role = "Support"
	RoleJungle  Role = "Jungle"
	RoleCarry   Role = "Carry"
	RoleSolo    Role = "Solo"
)

// God represents a Smite 2 god with their preferred roles and abilities
type God struct {
	Name        string
	Roles       []Role
	PortraitURL string
	DamageType  string // "Physical", "Magical", or "Mixed"
	AttackType  string // "Melee", "Ranged", or "Hybrid"
	Abilities   []Ability
	Pantheon    string
}

// Ability represents a god's ability
type Ability struct {
	Name        string
	Slot        string // "Position 1", "Position 2", "Position 3", "Position 4", "Passive"
	Description string
	IconURL     string // URL to the ability icon
}

// APIResponse represents the response structure from the Smite 2 API
type APIResponse struct {
	Data []APIGod `json:"data"`
}

// APIGod represents a god from the Smite 2 API
type APIGod struct {
	Attributes APIGodAttributes `json:"attributes"`
}

// APIGodAttributes represents the attributes of a god from the API
type APIGodAttributes struct {
	Name     string       `json:"Name"`
	Roles    APIRoles     `json:"roles"`
	Portrait APIPortrait  `json:"Portrait"`
	Ability  []APIAbility `json:"Ability"`
	Pantheon APIPantheon  `json:"pantheon"`
}

// APIAbility represents an ability from the API
type APIAbility struct {
	Name        string   `json:"Name"`
	Slot        string   `json:"Slot"`
	Description string   `json:"Description"`
	Icon        APIIcon  `json:"Icon"`
}

// APIIcon represents the icon data from the API
type APIIcon struct {
	Data APIIconData `json:"data"`
}

// APIIconData represents the icon data container
type APIIconData struct {
	Attributes APIImageAttributes `json:"attributes"`
}

// APIPantheon represents the pantheon data from the API
type APIPantheon struct {
	Data APIPantheonData `json:"data"`
}

// APIPantheonData represents the pantheon data container
type APIPantheonData struct {
	Attributes APIPantheonAttributes `json:"attributes"`
}

// APIPantheonAttributes represents the attributes of a pantheon
type APIPantheonAttributes struct {
	Name string `json:"Name"`
}

// APIPortrait represents the portrait data from the API
type APIPortrait struct {
	Data APIPortraitData `json:"data"`
}

// APIPortraitData represents the portrait data container
type APIPortraitData struct {
	Attributes APIImageAttributes `json:"attributes"`
}

// APIImageAttributes represents image attributes
type APIImageAttributes struct {
	URL string `json:"url"`
}

// APIRoles represents the roles data from the API
type APIRoles struct {
	Data []APIRole `json:"data"`
}

// APIRole represents a single role from the API
type APIRole struct {
	Attributes APIRoleAttributes `json:"attributes"`
}

// APIRoleAttributes represents the attributes of a role from the API
type APIRoleAttributes struct {
	Name  string       `json:"Name"`
	Image APIRoleImage `json:"Image"`
}

// APIRoleImage represents the role image data from the API
type APIRoleImage struct {
	Data APIRoleImageData `json:"data"`
}

// APIRoleImageData represents the role image data container
type APIRoleImageData struct {
	Attributes APIImageAttributes `json:"attributes"`
}

// Team represents a team with gods assigned to each role
type Team struct {
	Mid     God
	Support God
	Jungle  God
	Carry   God
	Solo    God
}

// GameSetup represents the complete randomized game setup
type GameSetup struct {
	Team1 Team
	Team2 Team
}

// GetAllRoles returns all conquest roles in order
func GetAllRoles() []Role {
	return []Role{RoleMid, RoleSupport, RoleJungle, RoleCarry, RoleSolo}
}

// GetRoleEmojis returns the emoji mappings for roles (team 1 and team 2)
func GetRoleEmojis() (map[Role]string, map[Role]string) {
	team1Emojis := map[Role]string{
		RoleMid:     "1️⃣",
		RoleSupport: "2️⃣",
		RoleJungle:  "3️⃣",
		RoleCarry:   "4️⃣",
		RoleSolo:    "5️⃣",
	}

	team2Emojis := map[Role]string{
		RoleMid:     "🥇",
		RoleSupport: "🥈",
		RoleJungle:  "🥉",
		RoleCarry:   "🏅",
		RoleSolo:    "🎖️",
	}

	return team1Emojis, team2Emojis
}

// GetRoleImageURL returns the image URL for a given role
func GetRoleImageURL(role Role) string {
	// Ensure gods are loaded
	if err := EnsureGodsLoaded(); err != nil {
		return ""
	}

	// Return the role image URL from our stored map
	if url, exists := roleImageURLs[role]; exists {
		return url
	}
	return ""
}

// godsList contains all Smite 2 gods loaded from the API data
var godsList []God

// roleImageURLs contains the image URLs for each role
var roleImageURLs map[Role]string

// LoadGodsFromFile loads gods from the JSON file
func LoadGodsFromFile(filePath string) error {
	// If no file path provided, try default location
	if filePath == "" {
		filePath = filepath.Join("data", "gods.json")
	}

	// Read the JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read gods file: %w", err)
	}

	// Parse the JSON response
	var apiResponse APIResponse
	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse gods JSON: %w", err)
	}

	// Convert API gods to internal format
	gods := make([]God, 0, len(apiResponse.Data))
	roleImages := make(map[Role]string)

	for _, apiGod := range apiResponse.Data {
		// Convert API abilities to internal format
		abilities := make([]Ability, 0, len(apiGod.Attributes.Ability))
		for _, apiAbility := range apiGod.Attributes.Ability {
			abilities = append(abilities, Ability{
				Name:        apiAbility.Name,
				Slot:        apiAbility.Slot,
				Description: apiAbility.Description,
				IconURL:     apiAbility.Icon.Data.Attributes.URL,
			})
		}

		// Convert API roles to internal Role type
		roles := make([]Role, 0, len(apiGod.Attributes.Roles.Data))
		for _, apiRole := range apiGod.Attributes.Roles.Data {
			role := Role(apiRole.Attributes.Name)
			roles = append(roles, role)

			// Store role image URL if we haven't seen this role yet
			if _, exists := roleImages[role]; !exists {
				roleImages[role] = apiRole.Attributes.Image.Data.Attributes.URL
			}
		}

		god := God{
			Name:        apiGod.Attributes.Name,
			Roles:       roles,
			PortraitURL: apiGod.Attributes.Portrait.Data.Attributes.URL,
			Abilities:   abilities,
			Pantheon:    apiGod.Attributes.Pantheon.Data.Attributes.Name,
			DamageType:  determineDamageType(abilities),
			AttackType:  determineAttackType(apiGod.Attributes.Name, roles),
		}

		gods = append(gods, god)
	}

	// Store the loaded gods and role images
	godsList = gods
	roleImageURLs = roleImages
	return nil
}

// determineDamageType analyzes abilities to determine if a god is Physical, Magical, or Mixed
func determineDamageType(abilities []Ability) string {
	hasPhysical := false
	hasMagical := false

	for _, ability := range abilities {
		desc := strings.ToLower(ability.Description)
		if strings.Contains(desc, "physical damage") {
			hasPhysical = true
		}
		if strings.Contains(desc, "magical damage") {
			hasMagical = true
		}
	}

	if hasPhysical && hasMagical {
		return "Mixed"
	} else if hasPhysical {
		return "Physical"
	} else if hasMagical {
		return "Magical"
	}

	// Default fallback based on role if no explicit damage type found
	return "Unknown"
}

// determineAttackType determines if a god is primarily Melee, Ranged, or Hybrid based on role and name
func determineAttackType(godName string, roles []Role) string {
	// For now, use role-based heuristics since explicit range info isn't in the JSON
	// This could be enhanced with a lookup table for specific gods if needed

	hasCarry := false
	hasSupport := false
	hasSolo := false

	for _, role := range roles {
		switch role {
		case RoleCarry:
			hasCarry = true
		case RoleSupport:
			hasSupport = true
		case RoleSolo:
			hasSolo = true
		}
	}

	// Carry gods are typically ranged
	if hasCarry && !hasSolo {
		return "Ranged"
	}

	// Support gods are typically melee
	if hasSupport && !hasCarry {
		return "Melee"
	}

	// Gods that can do both carry and solo are often hybrid
	if hasCarry && hasSolo {
		return "Hybrid"
	}

	// Default to checking for specific god names or patterns
	godNameLower := strings.ToLower(godName)

	// Some known ranged gods
	rangedGods := []string{"anhur", "ullr", "apollo", "artemis", "cupid", "neith"}
	for _, ranged := range rangedGods {
		if strings.Contains(godNameLower, ranged) {
			return "Ranged"
		}
	}

	// Default to melee if unsure
	return "Melee"
}

// GetGodsCount returns the number of loaded gods
func GetGodsCount() int {
	return len(godsList)
}

// GodInfoForDiscord represents formatted god information for Discord embeds
type GodInfoForDiscord struct {
	Title        string
	Description  string
	ThumbnailURL string
	Fields       []GodInfoField
}

// GodInfoField represents a field in the Discord embed
type GodInfoField struct {
	Name   string
	Value  string
	Inline bool
}

// FormatGodInfoForDiscord returns structured god info for Discord embeds
func FormatGodInfoForDiscord(god *God) *GodInfoForDiscord {
	if god == nil {
		return &GodInfoForDiscord{
			Title:       "God Information",
			Description: "God information not available.",
		}
	}

	// Create the main description with basic info
	description := fmt.Sprintf("**%s Pantheon**\n🔸 **Damage Type:** %s\n🔸 **Attack Type:** %s", 
		god.Pantheon, god.DamageType, god.AttackType)

	// Add roles
	rolesList := make([]string, len(god.Roles))
	for i, role := range god.Roles {
		rolesList[i] = string(role)
	}
	description += fmt.Sprintf("\n🔸 **Roles:** %s", strings.Join(rolesList, ", "))

	result := &GodInfoForDiscord{
		Title:        god.Name,
		Description:  description,
		ThumbnailURL: god.PortraitURL,
		Fields:       make([]GodInfoField, 0),
	}

	// Find and add passive first
	for _, ability := range god.Abilities {
		if strings.ToLower(ability.Slot) == "passive" {
			desc := stripHTMLTags(ability.Description)
			result.Fields = append(result.Fields, GodInfoField{
				Name:   fmt.Sprintf("%s (Passive)", ability.Name),
				Value:  desc,
				Inline: false,
			})
			break
		}
	}

	// Then add active abilities
	abilityCount := 0
	for _, ability := range god.Abilities {
		if strings.ToLower(ability.Slot) != "passive" && abilityCount < 4 {
			desc := stripHTMLTags(ability.Description)
			result.Fields = append(result.Fields, GodInfoField{
				Name:   fmt.Sprintf("%s (%s)", ability.Name, ability.Slot),
				Value:  desc,
				Inline: false,
			})
			abilityCount++
		}
	}

	return result
}

// FormatGodInfo returns a formatted string with god mechanics information (legacy function)
func FormatGodInfo(god *God) string {
	info := FormatGodInfoForDiscord(god)
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("**%s**\n", info.Title))
	result.WriteString(fmt.Sprintf("%s\n\n", info.Description))
	
	result.WriteString("**Abilities:**\n")
	for _, field := range info.Fields {
		result.WriteString(fmt.Sprintf("**%s**\n└ %s\n\n", field.Name, field.Value))
	}
	
	return result.String()
}

// GetGodAbilityIcons returns the ability icons for a god (passive first, then active abilities)
func GetGodAbilityIcons(god *God) []string {
	if god == nil {
		return []string{}
	}

	var icons []string
	
	// Find passive first
	for _, ability := range god.Abilities {
		if strings.ToLower(ability.Slot) == "passive" && ability.IconURL != "" {
			icons = append(icons, ability.IconURL)
			break
		}
	}

	// Then add active ability icons
	abilityCount := 0
	for _, ability := range god.Abilities {
		if strings.ToLower(ability.Slot) != "passive" && abilityCount < 4 && ability.IconURL != "" {
			icons = append(icons, ability.IconURL)
			abilityCount++
		}
	}

	return icons
}

// stripHTMLTags removes HTML tags from ability descriptions
func stripHTMLTags(text string) string {
	// Simple HTML tag removal - replace common tags
	text = strings.ReplaceAll(text, "<p>", "")
	text = strings.ReplaceAll(text, "</p>", "")
	text = strings.ReplaceAll(text, "<br>", " ")
	text = strings.ReplaceAll(text, "<br/>", " ")
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	// Remove any remaining HTML tags using a simple approach
	for strings.Contains(text, "<") && strings.Contains(text, ">") {
		start := strings.Index(text, "<")
		end := strings.Index(text[start:], ">")
		if end != -1 {
			text = text[:start] + text[start+end+1:]
		} else {
			break
		}
	}

	return strings.TrimSpace(text)
}

// GetGodByName returns a god by name if found
func GetGodByName(name string) *God {
	// Ensure gods are loaded
	if err := EnsureGodsLoaded(); err != nil {
		return nil
	}

	for _, god := range godsList {
		if strings.EqualFold(god.Name, name) {
			return &god
		}
	}
	return nil
}

// EnsureGodsLoaded ensures gods are loaded, loads from default path if not
func EnsureGodsLoaded() error {
	if len(godsList) == 0 {
		return LoadGodsFromFile("")
	}
	return nil
}

// GetRandomGodForRole returns a random god that can play the specified role
func GetRandomGodForRole(role Role, usedGods map[string]bool) *God {
	// Ensure gods are loaded
	if err := EnsureGodsLoaded(); err != nil {
		fmt.Printf("Failed to load gods: %v\n", err)
		return nil
	}

	var viableGods []God

	for _, god := range godsList {
		// Skip if god is already used
		if usedGods[god.Name] {
			continue
		}

		// Check if god can play this role
		for _, godRole := range god.Roles {
			if godRole == role {
				viableGods = append(viableGods, god)
				break
			}
		}
	}

	if len(viableGods) == 0 {
		return nil
	}

	selectedGod := viableGods[rand.Intn(len(viableGods))]
	return &selectedGod
}

// GenerateRandomTeams creates two random teams with unique gods
func GenerateRandomTeams() (*GameSetup, error) {
	// Ensure gods are loaded
	if err := EnsureGodsLoaded(); err != nil {
		return nil, fmt.Errorf("failed to load gods: %w", err)
	}

	usedGods := make(map[string]bool)
	roles := GetAllRoles()

	var team1, team2 Team

	// Generate team 1
	for _, role := range roles {
		god := GetRandomGodForRole(role, usedGods)
		if god == nil {
			return nil, fmt.Errorf("could not find available god for role %s in team 1", role)
		}
		usedGods[god.Name] = true

		switch role {
		case RoleMid:
			team1.Mid = *god
		case RoleSupport:
			team1.Support = *god
		case RoleJungle:
			team1.Jungle = *god
		case RoleCarry:
			team1.Carry = *god
		case RoleSolo:
			team1.Solo = *god
		}
	}

	// reset gods so we can reuse them in team 2
	usedGods = make(map[string]bool)
	// Generate team 2
	for _, role := range roles {
		god := GetRandomGodForRole(role, usedGods)
		if god == nil {
			return nil, fmt.Errorf("could not find available god for role %s in team 2", role)
		}
		usedGods[god.Name] = true

		switch role {
		case RoleMid:
			team2.Mid = *god
		case RoleSupport:
			team2.Support = *god
		case RoleJungle:
			team2.Jungle = *god
		case RoleCarry:
			team2.Carry = *god
		case RoleSolo:
			team2.Solo = *god
		}
	}

	return &GameSetup{
		Team1: team1,
		Team2: team2,
	}, nil
}
