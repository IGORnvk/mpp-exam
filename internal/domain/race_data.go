package domain

type RaceData struct {
	AbilityScoreIncreases map[string]int
}

var AllRaces = map[string]RaceData{
	"human": {
		AbilityScoreIncreases: map[string]int{
			"STR": 1, "DEX": 1, "CON": 1, "INT": 1, "WIS": 1, "CHA": 1,
		},
	},
	"lightfoot halfling": {
		AbilityScoreIncreases: map[string]int{
			"DEX": 2, "CHA": 1,
		},
	},
	"high elf": {
		AbilityScoreIncreases: map[string]int{
			"DEX": 2, "INT": 1,
		},
	},
	"half orc": {
		AbilityScoreIncreases: map[string]int{
			"STR": 2, "CON": 1,
		},
	},
	"hill dwarf": {
		AbilityScoreIncreases: map[string]int{
			"CON": 2, "WIS": 1,
		},
	},
	"dwarf": {
		AbilityScoreIncreases: map[string]int{
			"CON": 2, "WIS": 0,
		},
	},
	"gnome": {
		AbilityScoreIncreases: map[string]int{
			"INT": 2,
		},
	},
}
