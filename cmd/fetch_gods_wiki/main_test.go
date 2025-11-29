package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/oliver/lan_smite_teams_bot/internal/smite"
)

// TestScraperOutput tests that the scraper output can be read by gods.go
func TestScraperOutput(t *testing.T) {
	// Create a sample output that mimics what the scraper would produce
	sampleGod := smite.APIGod{
		Attributes: smite.APIGodAttributes{
			Name: "Test God",
			Roles: smite.APIRoles{
				Data: []smite.APIRole{
					{
						Attributes: smite.APIRoleAttributes{
							Name: "Mid",
							Image: smite.APIRoleImage{
								Data: smite.APIRoleImageData{
									Attributes: smite.APIImageAttributes{
										URL: "https://wiki.smite2.com/images/roles/mid.png",
									},
								},
							},
						},
					},
				},
			},
			Portrait: smite.APIPortrait{
				Data: smite.APIPortraitData{
					Attributes: smite.APIImageAttributes{
						URL: "https://wiki.smite2.com/images/test_god.png",
					},
				},
			},
			Ability: []smite.APIAbility{
				{
					Name:        "Test Passive",
					Slot:        "Passive",
					Description: "A test passive ability",
					Icon: smite.APIIcon{
						Data: smite.APIIconData{
							Attributes: smite.APIImageAttributes{
								URL: "https://wiki.smite2.com/images/abilities/Test_God_0.png",
							},
						},
					},
				},
				{
					Name:        "Test Ability 1",
					Slot:        "Position 1",
					Description: "A test ability in position 1",
					Icon: smite.APIIcon{
						Data: smite.APIIconData{
							Attributes: smite.APIImageAttributes{
								URL: "https://wiki.smite2.com/images/abilities/Test_God_1.png",
							},
						},
					},
				},
			},
			Pantheon: smite.APIPantheon{
				Data: smite.APIPantheonData{
					Attributes: smite.APIPantheonAttributes{
						Name: "Greek",
					},
				},
			},
		},
	}

	response := smite.APIResponse{
		Data: []smite.APIGod{sampleGod},
	}

	// Create temp directory and file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_output.json")

	// Marshal and write
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Now try to load it using the gods package
	if err := smite.LoadGodsFromFile(tempFile); err != nil {
		t.Fatalf("Failed to load gods from scraper output format: %v", err)
	}

	// Verify the data was loaded correctly
	gods := smite.GetAllGods()
	if len(gods) != 1 {
		t.Fatalf("Expected 1 god, got %d", len(gods))
	}

	god := gods[0]
	if god.Name != "Test God" {
		t.Errorf("Expected god name 'Test God', got '%s'", god.Name)
	}

	if len(god.Roles) != 1 || god.Roles[0] != smite.RoleMid {
		t.Errorf("Expected role 'Mid', got %v", god.Roles)
	}

	if god.Pantheon != "Greek" {
		t.Errorf("Expected pantheon 'Greek', got '%s'", god.Pantheon)
	}

	if len(god.Abilities) < 2 {
		t.Errorf("Expected at least 2 abilities, got %d", len(god.Abilities))
	}
}

// TestHelperFunctions tests the helper functions used in the scraper
func TestStripHTMLHelper(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple HTML tags",
			input:    "<p>Hello World</p>",
			expected: "Hello World",
		},
		{
			name:     "HTML entities",
			input:    "Test&nbsp;with&amp;entities",
			expected: "Test with&entities",
		},
		{
			name:     "Complex HTML",
			input:    "<div class='test'><p>Paragraph</p><br/>Text</div>",
			expected: "ParagraphText",
		},
		{
			name:     "No HTML",
			input:    "Plain text",
			expected: "Plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHTML(tt.input)
			if result != tt.expected {
				t.Errorf("stripHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestAppendUnique tests the appendUnique helper function
func TestAppendUnique(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected []string
	}{
		{
			name:     "Add new item",
			slice:    []string{"a", "b"},
			item:     "c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Don't add duplicate",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Add to empty slice",
			slice:    []string{},
			item:     "a",
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendUnique(tt.slice, tt.item)
			if len(result) != len(tt.expected) {
				t.Errorf("appendUnique() length = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("appendUnique()[%d] = %s, want %s", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// TestBuildAPIRoles tests the buildAPIRoles helper
func TestBuildAPIRoles(t *testing.T) {
	roles := []string{"Mid", "Jungle"}
	apiRoles := buildAPIRoles(roles)

	if len(apiRoles.Data) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(apiRoles.Data))
	}

	if apiRoles.Data[0].Attributes.Name != "Mid" {
		t.Errorf("Expected first role 'Mid', got '%s'", apiRoles.Data[0].Attributes.Name)
	}

	if apiRoles.Data[1].Attributes.Name != "Jungle" {
		t.Errorf("Expected second role 'Jungle', got '%s'", apiRoles.Data[1].Attributes.Name)
	}

	// Verify image URLs are generated
	if apiRoles.Data[0].Attributes.Image.Data.Attributes.URL == "" {
		t.Error("Expected non-empty image URL for Mid role")
	}
}

// TestBuildAPIPantheon tests the buildAPIPantheon helper
func TestBuildAPIPantheon(t *testing.T) {
	pantheon := buildAPIPantheon("Greek")

	if pantheon.Data.Attributes.Name != "Greek" {
		t.Errorf("Expected pantheon 'Greek', got '%s'", pantheon.Data.Attributes.Name)
	}
}

// TestBuildAPIPortrait tests the buildAPIPortrait helper
func TestBuildAPIPortrait(t *testing.T) {
	url := "https://example.com/portrait.png"
	portrait := buildAPIPortrait(url)

	if portrait.Data.Attributes.URL != url {
		t.Errorf("Expected URL '%s', got '%s'", url, portrait.Data.Attributes.URL)
	}
}

// TestIsLikelyAbility tests the isLikelyAbility helper
func TestIsLikelyAbility(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		expected    bool
	}{
		{
			name:        "Valid ability",
			title:       "Fire Strike",
			description: "Deals massive fire damage to enemies",
			expected:    true,
		},
		{
			name:        "Lore section",
			title:       "Lore",
			description: "Once upon a time...",
			expected:    false,
		},
		{
			name:        "Tips section",
			title:       "Tips and Tricks",
			description: "Use this ability wisely",
			expected:    false,
		},
		{
			name:        "Short description",
			title:       "Ability",
			description: "Short",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLikelyAbility(tt.title, tt.description)
			if result != tt.expected {
				t.Errorf("isLikelyAbility(%q, %q) = %v, want %v",
					tt.title, tt.description, result, tt.expected)
			}
		})
	}
}
