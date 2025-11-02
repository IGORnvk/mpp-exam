package domain

import (
	"fmt"
	"math"
	"strings"
)

var AllAbilities = map[string]string{
	"STR": "strength",
	"DEX": "dexterity",
	"CON": "constitution",
	"INT": "intelligence",
	"WIS": "wisdom",
	"CHA": "charisma",
}

var AllSkills = map[string]string{
	"Acrobatics": "DEX", "Animal Handling": "WIS", "Arcana": "INT",
	"Athletics": "STR", "Deception": "CHA", "History": "INT",
	"Insight": "WIS", "Intimidation": "CHA", "Investigation": "INT",
	"Medicine": "WIS", "Nature": "INT", "Perception": "WIS",
	"Performance": "CHA", "Persuasion": "CHA", "Religion": "INT",
	"Sleight of Hand": "DEX", "Stealth": "DEX", "Survival": "WIS",
}

type Ability struct {
	Score    int
	Modifier int
}

func (a *Ability) CalculateModifier() {
	a.Modifier = int(math.Floor(float64(a.Score-10) / 2))
}

type Character struct {
	Name             string
	Race             string
	Class            string
	SpellcasterType  SpellcasterType
	Background       string
	Level            int
	ProficiencyBonus int

	MaxHitPoints      int
	CurrentHitPoints  int
	ArmorClass        int
	Initiative        int
	PassivePerception int

	AbilityScores          map[string]Ability
	SkillProficiencies     map[string]bool
	SkillExpertise         map[string]bool
	EquippedWeaponMainHand Weapon
	EquippedWeaponOffHand  Weapon
	EquippedArmor          Armor
	EquippedShield         Shield
	KnownSpells            map[string]Spell
	PreparedSpells         map[string]Spell

	MaxSpellSlots       map[int]int
	SpellCastingAbility string
	SpellSaveDC         int
	SpellAttackBonus    int
}

func NewCharacter(name, race, class, background string, scoreAssignments map[string]int) (*Character, error) {
	if len(scoreAssignments) != 6 {
		return nil, fmt.Errorf("must provide 6 ability scores using the Standard Array")
	}

	char := &Character{
		Name:               name,
		Race:               race,
		Class:              class,
		Background:         background,
		Level:              1,
		AbilityScores:      make(map[string]Ability),
		MaxSpellSlots:      make(map[int]int),
		SkillProficiencies: make(map[string]bool),
		SkillExpertise:     make(map[string]bool),
		KnownSpells:        make(map[string]Spell),
		PreparedSpells:     make(map[string]Spell),
	}

	if data, ok := AllClassesData[strings.ToLower(class)]; ok {
		char.SpellcasterType = data.SpellType
	} else {
		char.SpellcasterType = NoSpellcasting
	}

	for ab, score := range scoreAssignments {
		ability := Ability{Score: score}
		ability.CalculateModifier()
		char.AbilityScores[ab] = ability
	}

	normalizedRace := strings.ToLower(char.Race)

	if raceData, ok := AllRaces[normalizedRace]; ok {
		for ab, bonus := range raceData.AbilityScoreIncreases {
			if currentAbility, ok := char.AbilityScores[ab]; ok {
				currentAbility.Score += bonus

				currentAbility.CalculateModifier()

				char.AbilityScores[ab] = currentAbility
			}
		}
	}

	for skill := range AllSkills {
		char.SkillProficiencies[skill] = false
	}

	char.UpdateProficiencyBonus(1)
	return char, nil
}

func (c *Character) SetSkillProficiencies(skills []string) {
	for _, skill := range skills {
		if _, exists := AllSkills[skill]; exists {
			if c.SkillProficiencies[skill] {
				c.SkillExpertise[skill] = true
			}

			c.SkillProficiencies[skill] = true
		}
	}
}

func (c *Character) UpdateProficiencyBonus(newLevel int) {
	c.Level = newLevel
	c.ProficiencyBonus = 2 + int(math.Floor(float64(c.Level-1)/4))
}

func (c *Character) GetSkillModifier(skill string) int {
	abilityKey, ok := AllSkills[skill]
	if !ok {
		return 0
	}

	ability, ok := c.AbilityScores[abilityKey]
	if !ok {
		return 0
	}

	modifier := ability.Modifier
	isProficient := c.SkillProficiencies[skill]

	if isProficient {
		modifier += c.ProficiencyBonus
	}

	return modifier
}

func (c *Character) GetAbilityForSkill(skillName string) string {
	if ability, ok := AllSkills[skillName]; ok {
		return ability
	}
	return "?"
}

func (c *Character) CalculateMaxSpellSlots() {
	c.MaxSpellSlots = make(map[int]int)
	classLower := strings.ToLower(c.Class)

	var slots map[int]int
	var cantripProgression map[int]int

	switch classLower {
	case "wizard", "sorcerer", "bard", "cleric", "druid":
		slots = FullCasterSlots[c.Level]
		cantripProgression = CantripCount["default_full"]

	case "paladin", "ranger":
		slots = HalfCasterSlots[c.Level]
		cantripProgression = nil

	case "warlock":
		slots = PactCasterSlots[c.Level]
		cantripProgression = CantripCount["warlock"]

	default:
		// Non-casters
		return
	}

	for level, count := range slots {
		c.MaxSpellSlots[level] = count
	}

	if cantripProgression != nil {
		maxLvl := 0
		maxCount := 0

		for lvl, count := range cantripProgression {
			if c.Level >= lvl {
				if lvl > maxLvl {
					maxLvl = lvl
					maxCount = count
				}
			}
		}

		if maxCount > 0 {
			c.MaxSpellSlots[0] = maxCount
		}
	}
}

func (c *Character) CalculateMaxHitPoints() {
	hitDie := 6
	hitDieAvg := 4
	switch c.Class {
	case "Fighter", "Paladin":
		hitDie = 10
		hitDieAvg = 6
	case "Cleric", "Druid", "Ranger", "Rogue", "Bard", "Warlock":
		hitDie = 8
		hitDieAvg = 5
	case "Sorcerer":
		hitDie = 6
		hitDieAvg = 4
	}

	conMod := c.AbilityScores["CON"].Modifier
	totalMaxHP := 0

	lvl1HP := hitDie + conMod
	if lvl1HP < 1 {
		lvl1HP = 1
	}
	totalMaxHP += lvl1HP

	for lvl := 2; lvl <= c.Level; lvl++ {
		levelHP := hitDieAvg + conMod
		if levelHP < 1 {
			levelHP = 1
		}
		totalMaxHP += levelHP
	}

	c.MaxHitPoints = totalMaxHP

	if c.CurrentHitPoints == 0 || c.CurrentHitPoints == c.MaxHitPoints {
		c.CurrentHitPoints = totalMaxHP
	} else if c.CurrentHitPoints > totalMaxHP {
		c.CurrentHitPoints = totalMaxHP
	}
}

func (c *Character) CalculateCombatStats() {
	dexMod := c.AbilityScores["DEX"].Modifier

	c.Initiative = dexMod
	c.PassivePerception = 10 + c.GetSkillModifier("Perception")

	baseAC := 10 + dexMod

	classLower := strings.ToLower(c.Class)

	if c.EquippedArmor.Name == "" {
		if classLower == "barbarian" {
			conMod := c.AbilityScores["CON"].Modifier
			baseAC = 10 + dexMod + conMod
		} else if classLower == "monk" {
			wisMod := c.AbilityScores["WIS"].Modifier
			baseAC = 10 + dexMod + wisMod
		}
	}

	if c.EquippedArmor.Name != "" && c.EquippedArmor.AC > 0 {
		baseAC = c.EquippedArmor.AC

		if c.EquippedArmor.DexBonus != "" {
			currentDexMod := dexMod

			if c.EquippedArmor.DexBonus == "limited" {
				currentDexMod = int(math.Min(float64(dexMod), 2.0))
			} else if c.EquippedArmor.DexBonus == "none" {
				currentDexMod = 0
			}

			baseAC += currentDexMod
		}
	}

	c.ArmorClass = baseAC

	if c.EquippedShield.Name != "" {
		c.ArmorClass += 2
	}
}

func (c *Character) CalculateSpellStats() {
	classLower := strings.ToLower(c.Class)

	ability := ""
	switch classLower {
	case "bard", "sorcerer", "warlock", "paladin":
		ability = "CHA"
	case "cleric", "druid", "ranger":
		ability = "WIS"
	case "wizard":
		ability = "INT"
	default:
		c.SpellCastingAbility = ""
		c.SpellSaveDC = 0
		c.SpellAttackBonus = 0
		return
	}
	c.SpellCastingAbility = ability

	mod := c.AbilityScores[ability].Modifier
	pb := c.ProficiencyBonus

	c.SpellSaveDC = 8 + pb + mod
	c.SpellAttackBonus = pb + mod
}

func (c *Character) EquipWeaponSlot(weapon Weapon, slot string) error {
	switch slot {
	case "main hand":
		if c.EquippedWeaponMainHand.Name != "" {
			return fmt.Errorf("main hand already occupied")
		}

		c.EquippedWeaponMainHand = weapon
		if weapon.TwoHanded {
			c.EquippedWeaponOffHand = Weapon{}
		}
	case "off hand":
		if c.EquippedWeaponOffHand.Name != "" {
			return fmt.Errorf("off hand already occupied")
		}

		if c.EquippedWeaponMainHand.TwoHanded {
			return fmt.Errorf("cannot equip to off hand: main hand weapon '%s' is two-handed", c.EquippedWeaponMainHand.Name)
		}
		c.EquippedWeaponOffHand = weapon
	default:
		return fmt.Errorf("invalid equipment slot '%s'. Must be 'main hand' or 'off hand'", slot)
	}

	return nil
}
