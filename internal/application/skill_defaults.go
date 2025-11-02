package application

var BackgroundSkills = map[string][]string{
	"acolyte":   {"Insight", "Religion"},
	"sage":      {"Arcana", "History"},
	"soldier":   {"Athletics", "Intimidation"},
	"outlander": {"Athletics", "Survival"},
}

var DefaultClassSkills = map[string][]string{
	"rogue":     {"Acrobatics", "Athletics", "Deception", "Insight"},
	"fighter":   {"Acrobatics", "Animal Handling"},
	"wizard":    {"Arcana", "History"},
	"sorcerer":  {"Intimidation", "Persuasion"},
	"warlock":   {"Arcana", "Deception"},
	"paladin":   {"Athletics", "Insight"},
	"cleric":    {"History", "Insight"},
	"barbarian": {"Animal Handling", "Athletics"},
	"monk":      {"Acrobatics", "Athletics"},
}
