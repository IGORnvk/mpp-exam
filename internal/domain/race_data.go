package domain

type RaceData struct {
	AbilityScoreIncreases map[string]int
	SkillProficiency      string
}

var AllRaces = map[string]RaceData{
	"human": {
		AbilityScoreIncreases: map[string]int{
			"STR": 1, "DEX": 1, "CON": 1, "INT": 1, "WIS": 1, "CHA": 1,
		},
		SkillProficiency: "",
	},
	"lightfoot halfling": {
		AbilityScoreIncreases: map[string]int{
			"DEX": 2, "CHA": 1,
		},
		SkillProficiency: "",
	},
	"high elf": {
		AbilityScoreIncreases: map[string]int{
			"DEX": 2, "INT": 1,
		},
		SkillProficiency: "Perception",
	},
	"half orc": {
		AbilityScoreIncreases: map[string]int{
			"STR": 2, "CON": 1,
		},
		SkillProficiency: "Intimidation",
	},
	"hill dwarf": {
		AbilityScoreIncreases: map[string]int{
			"CON": 2, "WIS": 1,
		},
		SkillProficiency: "History",
	},
	"dwarf": {
		AbilityScoreIncreases: map[string]int{
			"CON": 2, "WIS": 0,
		},
		SkillProficiency: "History",
	},
	"gnome": {
		AbilityScoreIncreases: map[string]int{
			"INT": 2,
		},
		SkillProficiency: "",
	},
}
