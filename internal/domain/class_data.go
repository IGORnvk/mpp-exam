package domain

type SpellcasterType int

const (
	NoSpellcasting SpellcasterType = iota
	LearnedCasting
	PreparedCasting
)

type ClassData struct {
	SpellType           SpellcasterType
	SpellcastingAbility string
}

var AllClassesData = map[string]ClassData{
	"fighter":   {SpellType: NoSpellcasting},
	"rogue":     {SpellType: NoSpellcasting},
	"barbarian": {SpellType: NoSpellcasting},
	"monk":      {SpellType: NoSpellcasting},

	"wizard":  {SpellType: PreparedCasting, SpellcastingAbility: "INT"},
	"cleric":  {SpellType: PreparedCasting, SpellcastingAbility: "WIS"},
	"paladin": {SpellType: PreparedCasting, SpellcastingAbility: "CHA"},

	// Learned Casters
	"sorcerer": {SpellType: LearnedCasting, SpellcastingAbility: "CHA"},
	"warlock":  {SpellType: LearnedCasting, SpellcastingAbility: "CHA"},
}
