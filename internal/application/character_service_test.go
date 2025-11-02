package application_test

import (
	"context"
	"dnd-char-generator/internal/application"
	"dnd-char-generator/internal/domain"
	"sort"
	"strings"
	"testing"
)

type mockRepo struct{}

func (m *mockRepo) Save(ctx context.Context, char *domain.Character) error { return nil }
func (m *mockRepo) FindByID(ctx context.Context, name string) (*domain.Character, error) {
	return nil, nil
}
func (m *mockRepo) FindAll(ctx context.Context) ([]*domain.Character, error) { return nil, nil }
func (m *mockRepo) Delete(ctx context.Context, name string) error            { return nil }

type mockAPIClient struct{}

func (m *mockAPIClient) EnrichSpell(ctx context.Context, spells *domain.Spell)   {}
func (m *mockAPIClient) EnrichWeapon(ctx context.Context, weapon *domain.Weapon) {}
func (m *mockAPIClient) EnrichArmor(ctx context.Context, armor *domain.Armor)    {}

func setupService() *application.CharacterService {
	return application.NewCharacterService(
		&mockRepo{},
		&mockAPIClient{},
		nil,
		nil,
		nil,
		nil,
	)
}

func TestRacialSkillProficiencies(t *testing.T) {
	service := setupService()

	standardScores := map[string]int{
		"STR": 15, "DEX": 14, "CON": 13, "INT": 12, "WIS": 10, "CHA": 8,
	}

	tests := []struct {
		name              string
		request           application.CreateCharacterRequest
		expectedSkills    []string
		expectedExpertise map[string]bool
	}{
		{
			name: "Dwarf Acolyte Rogue - Includes History (Racial)",
			request: application.CreateCharacterRequest{
				Name:             "Dwarf Rogue",
				Race:             "hill dwarf",
				Class:            "rogue",
				Background:       "acolyte",
				Level:            1,
				ScoreAssignments: standardScores,
				InitialSkills:    []string{"Acrobatics", "Deception", "Athletics", "Insight"},
			},
			expectedSkills:    []string{"Acrobatics", "Athletics", "Deception", "History", "Insight", "Religion"},
			expectedExpertise: map[string]bool{"Insight": true},
		},
		{
			name: "Half-Orc Acolyte Barbarian - Includes Intimidation (Racial)",
			request: application.CreateCharacterRequest{
				Name:             "Half-Orc Barbarian",
				Race:             "half orc",
				Class:            "barbarian",
				Background:       "acolyte",
				Level:            1,
				ScoreAssignments: standardScores,
				InitialSkills:    []string{"Animal Handling", "Athletics"},
			},
			expectedSkills:    []string{"Animal Handling", "Athletics", "Insight", "Intimidation", "Religion"},
			expectedExpertise: map[string]bool{},
		},
		{
			name: "Edge Case 1: Max Expertise from All Sources",
			request: application.CreateCharacterRequest{
				Name:             "High Elf Rogue",
				Race:             "high elf",
				Class:            "rogue",
				Background:       "sage",
				Level:            1,
				InitialSkills:    []string{"Acrobatics", "Deception", "Insight", "Perception"},
				ScoreAssignments: standardScores,
			},
			expectedSkills:    []string{"Acrobatics", "Arcana", "Deception", "History", "Insight", "Perception"},
			expectedExpertise: map[string]bool{"Perception": true},
		},
		{
			name: "Edge Case 2: Zero Proficiencies from Race/Background",
			request: application.CreateCharacterRequest{
				Name:             "Mountain Dwarf Soldier",
				Race:             "mountain dwarf",
				Class:            "barbarian",
				Background:       "soldier",
				Level:            1,
				ScoreAssignments: standardScores,
				InitialSkills:    []string{"Athletics", "Survival"},
			},
			expectedSkills:    []string{"Athletics", "Intimidation", "Survival"},
			expectedExpertise: map[string]bool{"Athletics": true},
		},
		{
			name: "Edge Case 3: All Sources without Overlap",
			request: application.CreateCharacterRequest{
				Name:             "High Elf Outlander Bard",
				Race:             "high elf",
				Class:            "bard",
				Background:       "outlander",
				Level:            1,
				ScoreAssignments: standardScores,
				InitialSkills:    []string{"Acrobatics", "Stealth", "Performance"},
			},
			expectedSkills:    []string{"Acrobatics", "Athletics", "Performance", "Perception", "Stealth", "Survival"},
			expectedExpertise: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			char, err := service.CreateCharacter(ctx, tt.request)
			if err != nil {
				t.Fatalf("CreateCharacter failed: %v", err)
			}

			var actualExpertiseCount int
			var expectedExpertiseCount int

			var actualProficientSkills []string
			for skill, isProficient := range char.SkillProficiencies {
				if isProficient {
					actualProficientSkills = append(actualProficientSkills, skill)
				}
			}

			sort.Strings(actualProficientSkills)
			sort.Strings(tt.expectedSkills)

			actualStr := strings.Join(actualProficientSkills, ", ")
			expectedStr := strings.Join(tt.expectedSkills, ", ")

			if actualStr != expectedStr {
				t.Errorf("Proficiencies mismatch for %s\nactual:   %s\nexpected: %s",
					tt.name, actualStr, expectedStr)
			}

			if tt.expectedExpertise != nil {
				for skill, expected := range tt.expectedExpertise {
					actual := char.SkillExpertise[skill]
					if actual != expected {
						t.Errorf("Expertise mismatch for %s on skill '%s'. Expected: %t, actual: %t",
							tt.name, skill, expected, actual)
					}
				}

				for _, isExpert := range char.SkillExpertise {
					if isExpert {
						actualExpertiseCount++
					}
				}
				for _, expected := range tt.expectedExpertise {
					if expected {
						expectedExpertiseCount++
					}
				}

				if actualExpertiseCount != expectedExpertiseCount {
					t.Errorf("Total expertise count mismatch for %s. Expected: %d, actual: %d",
						tt.name, expectedExpertiseCount, actualExpertiseCount)
				}
			}

			if actualStr == expectedStr && (tt.expectedExpertise == nil || actualExpertiseCount == expectedExpertiseCount) {
				t.Logf("PASS: %s", tt.name)
			}
		})
	}
}
