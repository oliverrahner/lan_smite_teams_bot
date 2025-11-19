package smite

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// createTestGodsFile creates a temporary test gods file
func createTestGodsFile(t *testing.T) string {
	// Create a temporary file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_gods.json")

	// Create test data - need enough gods for 2 full teams
	testData := APIResponse{
		Data: []APIGod{
			// Mid gods
			{
				Attributes: APIGodAttributes{
					Name: "Test Mid God 1",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Mid"}},
						},
					},
				},
			},
			{
				Attributes: APIGodAttributes{
					Name: "Test Mid God 2",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Mid"}},
						},
					},
				},
			},
			// Support gods
			{
				Attributes: APIGodAttributes{
					Name: "Test Support God 1",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Support"}},
						},
					},
				},
			},
			{
				Attributes: APIGodAttributes{
					Name: "Test Support God 2",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Support"}},
						},
					},
				},
			},
			// Jungle gods
			{
				Attributes: APIGodAttributes{
					Name: "Test Jungle God 1",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Jungle"}},
						},
					},
				},
			},
			{
				Attributes: APIGodAttributes{
					Name: "Test Jungle God 2",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Jungle"}},
						},
					},
				},
			},
			// Carry gods
			{
				Attributes: APIGodAttributes{
					Name: "Test Carry God 1",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Carry"}},
						},
					},
				},
			},
			{
				Attributes: APIGodAttributes{
					Name: "Test Carry God 2",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Carry"}},
						},
					},
				},
			},
			// Solo gods
			{
				Attributes: APIGodAttributes{
					Name: "Test Solo God 1",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Solo"}},
						},
					},
				},
			},
			{
				Attributes: APIGodAttributes{
					Name: "Test Solo God 2",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Solo"}},
						},
					},
				},
			},
			// Flex god
			{
				Attributes: APIGodAttributes{
					Name: "Test Flex God",
					Roles: APIRoles{
						Data: []APIRole{
							{Attributes: APIRoleAttributes{Name: "Mid"}},
							{Attributes: APIRoleAttributes{Name: "Solo"}},
						},
					},
				},
			},
		},
	}

	// Write test data to file
	data, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	err = os.WriteFile(tempFile, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	return tempFile
}

func TestLoadGodsFromFile(t *testing.T) {
	// Create test file
	testFile := createTestGodsFile(t)

	// Clear existing gods list
	godsList = nil

	// Test loading
	err := LoadGodsFromFile(testFile)
	if err != nil {
		t.Fatalf("LoadGodsFromFile() error = %v", err)
	}

	// Verify gods were loaded
	if len(godsList) != 11 {
		t.Errorf("Expected 11 gods loaded, got %d", len(godsList))
	}

	// Verify specific gods exist
	expectedGods := map[string][]Role{
		"Test Mid God 1":     {RoleMid},
		"Test Mid God 2":     {RoleMid},
		"Test Support God 1": {RoleSupport},
		"Test Support God 2": {RoleSupport},
		"Test Jungle God 1":  {RoleJungle},
		"Test Jungle God 2":  {RoleJungle},
		"Test Carry God 1":   {RoleCarry},
		"Test Carry God 2":   {RoleCarry},
		"Test Solo God 1":    {RoleSolo},
		"Test Solo God 2":    {RoleSolo},
		"Test Flex God":      {RoleMid, RoleSolo},
	}

	godsFound := make(map[string]bool)
	for _, god := range godsList {
		godsFound[god.Name] = true
		expectedRoles, exists := expectedGods[god.Name]
		if !exists {
			t.Errorf("Unexpected god loaded: %s", god.Name)
			continue
		}

		if len(god.Roles) != len(expectedRoles) {
			t.Errorf("God %s has %d roles, expected %d", god.Name, len(god.Roles), len(expectedRoles))
		}
	}

	// Check that all expected gods were found
	for expectedGod := range expectedGods {
		if !godsFound[expectedGod] {
			t.Errorf("Expected god not found: %s", expectedGod)
		}
	}
}

func TestLoadGodsFromFile_NonExistent(t *testing.T) {
	// Clear existing gods list
	godsList = nil

	// Test loading non-existent file
	err := LoadGodsFromFile("non_existent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file, got nil")
	}
}

func TestEnsureGodsLoaded(t *testing.T) {
	// Clear existing gods list
	godsList = nil

	// Create test file at default location
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	// Change to temp directory for this test
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	testFile := createTestGodsFile(t)
	err = os.Rename(testFile, filepath.Join(dataDir, "gods.json"))
	if err != nil {
		t.Fatalf("Failed to move test file: %v", err)
	}

	// Test ensure gods loaded
	err = EnsureGodsLoaded()
	if err != nil {
		t.Fatalf("EnsureGodsLoaded() error = %v", err)
	}

	// Verify gods were loaded
	if len(godsList) == 0 {
		t.Error("Expected gods to be loaded, but list is empty")
	}
}

func TestGetGodsCount(t *testing.T) {
	// Create test file and load
	testFile := createTestGodsFile(t)
	godsList = nil
	LoadGodsFromFile(testFile)

	count := GetGodsCount()
	if count != 11 {
		t.Errorf("Expected count 11, got %d", count)
	}
}

func TestGetRandomGodForRole(t *testing.T) {
	// Setup test data
	testFile := createTestGodsFile(t)
	godsList = nil
	err := LoadGodsFromFile(testFile)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	tests := []struct {
		name     string
		role     Role
		usedGods map[string]bool
		wantNil  bool
	}{
		{
			name:     "Get random mid god",
			role:     RoleMid,
			usedGods: make(map[string]bool),
			wantNil:  false,
		},
		{
			name:     "Get random support god",
			role:     RoleSupport,
			usedGods: make(map[string]bool),
			wantNil:  false,
		},
		{
			name:     "Get random jungle god",
			role:     RoleJungle,
			usedGods: make(map[string]bool),
			wantNil:  false,
		},
		{
			name:     "Get random carry god",
			role:     RoleCarry,
			usedGods: make(map[string]bool),
			wantNil:  false,
		},
		{
			name:     "Get random solo god",
			role:     RoleSolo,
			usedGods: make(map[string]bool),
			wantNil:  false,
		},
		{
			name:     "No available gods",
			role:     RoleMid,
			usedGods: map[string]bool{"Test Mid God 1": true, "Test Mid God 2": true, "Test Flex God": true},
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			god := GetRandomGodForRole(tt.role, tt.usedGods)

			if tt.wantNil {
				if god != nil {
					t.Errorf("GetRandomGodForRole() = %v, want nil", god)
				}
				return
			}

			if god == nil {
				t.Errorf("GetRandomGodForRole() = nil, want god")
				return
			}

			// Check if god can play the role
			canPlay := false
			for _, role := range god.Roles {
				if role == tt.role {
					canPlay = true
					break
				}
			}

			if !canPlay {
				t.Errorf("GetRandomGodForRole() returned god %s that cannot play role %s", god.Name, tt.role)
			}
		})
	}
}

func TestGenerateRandomTeams(t *testing.T) {
	// Setup test data
	testFile := createTestGodsFile(t)
	godsList = nil
	err := LoadGodsFromFile(testFile)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	gameSetup, err := GenerateRandomTeams()

	if err != nil {
		t.Fatalf("GenerateRandomTeams() error = %v", err)
	}

	if gameSetup == nil {
		t.Fatal("GenerateRandomTeams() returned nil game setup")
	}

	// Check that all roles are filled
	team1Gods := []God{gameSetup.Team1.Mid, gameSetup.Team1.Support, gameSetup.Team1.Jungle, gameSetup.Team1.Carry, gameSetup.Team1.Solo}
	team2Gods := []God{gameSetup.Team2.Mid, gameSetup.Team2.Support, gameSetup.Team2.Jungle, gameSetup.Team2.Carry, gameSetup.Team2.Solo}

	for i, god := range team1Gods {
		if god.Name == "" {
			t.Errorf("Team 1 role %d has empty god name", i)
		}
	}

	for i, god := range team2Gods {
		if god.Name == "" {
			t.Errorf("Team 2 role %d has empty god name", i)
		}
	}

	// Check that gods are unique within each team (but teams can overlap)
	// Check team 1 uniqueness
	team1Names := make(map[string]bool)
	for _, god := range team1Gods {
		if team1Names[god.Name] {
			t.Errorf("Duplicate god found in team 1: %s", god.Name)
		}
		team1Names[god.Name] = true
	}

	// Check team 2 uniqueness
	team2Names := make(map[string]bool)
	for _, god := range team2Gods {
		if team2Names[god.Name] {
			t.Errorf("Duplicate god found in team 2: %s", god.Name)
		}
		team2Names[god.Name] = true
	}

	// Each team should have exactly 5 unique gods
	if len(team1Names) != 5 {
		t.Errorf("Expected team 1 to have 5 unique gods, but found %d", len(team1Names))
	}
	if len(team2Names) != 5 {
		t.Errorf("Expected team 2 to have 5 unique gods, but found %d", len(team2Names))
	}
}

func TestGetAllRoles(t *testing.T) {
	roles := GetAllRoles()

	expectedRoles := []Role{RoleMid, RoleSupport, RoleJungle, RoleCarry, RoleSolo}

	if len(roles) != len(expectedRoles) {
		t.Errorf("Expected %d roles, got %d", len(expectedRoles), len(roles))
	}

	for i, role := range roles {
		if role != expectedRoles[i] {
			t.Errorf("Expected role %s at index %d, got %s", expectedRoles[i], i, role)
		}
	}
}

func TestGetRoleEmojis(t *testing.T) {
	team1Emojis, team2Emojis := GetRoleEmojis()

	// Check that we have emojis for all roles
	roles := GetAllRoles()

	for _, role := range roles {
		if team1Emojis[role] == "" {
			t.Errorf("Team 1 missing emoji for role %s", role)
		}
		if team2Emojis[role] == "" {
			t.Errorf("Team 2 missing emoji for role %s", role)
		}
	}

	// Check that emojis are different between teams
	for _, role := range roles {
		if team1Emojis[role] == team2Emojis[role] {
			t.Errorf("Team 1 and Team 2 have same emoji for role %s: %s", role, team1Emojis[role])
		}
	}
}

func BenchmarkGenerateRandomTeams(b *testing.B) {
	// Setup test data
	tempDir := b.TempDir()
	tempFile := filepath.Join(tempDir, "test_gods.json")

	testData := APIResponse{
		Data: []APIGod{
			// Mid gods
			{
				Attributes: APIGodAttributes{
					Name: "Test Mid God 1",
					Roles: APIRoles{
						Data: []APIRole{{Attributes: APIRoleAttributes{Name: "Mid"}}},
					},
				},
			},
			{
				Attributes: APIGodAttributes{
					Name: "Test Mid God 2",
					Roles: APIRoles{
						Data: []APIRole{{Attributes: APIRoleAttributes{Name: "Mid"}}},
					},
				},
			},
			// Support gods
			{
				Attributes: APIGodAttributes{
					Name: "Test Support God 1",
					Roles: APIRoles{
						Data: []APIRole{{Attributes: APIRoleAttributes{Name: "Support"}}},
					},
				},
			},
			{
				Attributes: APIGodAttributes{
					Name: "Test Support God 2",
					Roles: APIRoles{
						Data: []APIRole{{Attributes: APIRoleAttributes{Name: "Support"}}},
					},
				},
			},
			// Jungle gods
			{
				Attributes: APIGodAttributes{
					Name: "Test Jungle God 1",
					Roles: APIRoles{
						Data: []APIRole{{Attributes: APIRoleAttributes{Name: "Jungle"}}},
					},
				},
			},
			{
				Attributes: APIGodAttributes{
					Name: "Test Jungle God 2",
					Roles: APIRoles{
						Data: []APIRole{{Attributes: APIRoleAttributes{Name: "Jungle"}}},
					},
				},
			},
			// Carry gods
			{
				Attributes: APIGodAttributes{
					Name: "Test Carry God 1",
					Roles: APIRoles{
						Data: []APIRole{{Attributes: APIRoleAttributes{Name: "Carry"}}},
					},
				},
			},
			{
				Attributes: APIGodAttributes{
					Name: "Test Carry God 2",
					Roles: APIRoles{
						Data: []APIRole{{Attributes: APIRoleAttributes{Name: "Carry"}}},
					},
				},
			},
			// Solo gods
			{
				Attributes: APIGodAttributes{
					Name: "Test Solo God 1",
					Roles: APIRoles{
						Data: []APIRole{{Attributes: APIRoleAttributes{Name: "Solo"}}},
					},
				},
			},
			{
				Attributes: APIGodAttributes{
					Name: "Test Solo God 2",
					Roles: APIRoles{
						Data: []APIRole{{Attributes: APIRoleAttributes{Name: "Solo"}}},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(testData)
	os.WriteFile(tempFile, data, 0644)

	godsList = nil
	err := LoadGodsFromFile(tempFile)
	if err != nil {
		b.Fatalf("Failed to load test data: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateRandomTeams()
		if err != nil {
			b.Fatalf("GenerateRandomTeams() error = %v", err)
		}
	}
}

func TestGetRoleImageURL(t *testing.T) {
	// Setup test data with role images
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_gods.json")

	testData := APIResponse{
		Data: []APIGod{
			{
				Attributes: APIGodAttributes{
					Name: "Test Mid God",
					Portrait: APIPortrait{
						Data: APIPortraitData{
							Attributes: APIImageAttributes{
								URL: "https://example.com/mid_god.jpg",
							},
						},
					},
					Roles: APIRoles{
						Data: []APIRole{
							{
								Attributes: APIRoleAttributes{
									Name: "Mid",
									Image: APIRoleImage{
										Data: APIRoleImageData{
											Attributes: APIImageAttributes{
												URL: "https://example.com/mid_role.png",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(testData)
	os.WriteFile(tempFile, data, 0644)

	// Reset state and load test data
	godsList = nil
	roleImageURLs = nil
	err := LoadGodsFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	// Test getting role image URL
	url := GetRoleImageURL(RoleMid)
	expectedURL := "https://example.com/mid_role.png"
	if url != expectedURL {
		t.Errorf("Expected role image URL %s, got %s", expectedURL, url)
	}

	// Test non-existent role
	url = GetRoleImageURL(RoleCarry)
	if url != "" {
		t.Errorf("Expected empty URL for non-existent role, got %s", url)
	}
}
