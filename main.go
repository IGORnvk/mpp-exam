package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"dnd-char-generator/internal/application"
	"dnd-char-generator/internal/domain"
	"dnd-char-generator/internal/infrastructure"
	"dnd-char-generator/internal/infrastructure/dndapi"
	"dnd-char-generator/internal/infrastructure/persistence"
)

func usage() {
	fmt.Printf(`Usage:
  %s create -name NAME -race RACE -class CLASS -str N -dex N -con N -int N -wis N -cha N
  %s view -name CHARACTER_NAME
  %s list
  %s delete -name CHARACTER_NAME
  %s equip -name CHARACTER_NAME -weapon WEAPON_NAME -slot SLOT
  %s equip -name CHARACTER_NAME -armor ARMOR_NAME
  %s equip -name CHARACTER_NAME -shield SHIELD_NAME
  %s learn-spell -name CHARACTER_NAME -spell SPELL_NAME
  %s prepare-spell -name CHARACTER_NAME -spell SPELL_NAME 
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func initApp() (*application.CharacterService, error) {
	allSpells, allWeapons, allArmors, allShields, err := infrastructure.LoadData("5e-SRD-Equipment.csv", "5e-SRD-Spells.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to load static SRD data: %w", err)
	}

	repo := persistence.NewFileRepository("characters.json")
	apiClient := dndapi.NewClient()

	service := application.NewCharacterService(repo, apiClient, allSpells, allWeapons, allArmors, allShields)

	return service, nil
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	service, err := initApp()
	if err != nil {
		fmt.Printf("Initialization Error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	cmd := os.Args[1]

	switch cmd {
	case "create":
		handleCreate(ctx, service)
	case "view":
		handleView(ctx, service)
	case "list":
		handleList(ctx, service)
	case "update-level":
		handleUpdateLevel(ctx, service)
	case "equip":
		handleEquip(ctx, service)
	case "learn-spell":
		handleLearnSpell(ctx, service)
	case "prepare-spell":
		handlePrepareSpell(ctx, service)
	case "delete":
		handleDelete(ctx, service)
	default:
		usage()
		os.Exit(1)
	}
}

func handleCreate(ctx context.Context, service *application.CharacterService) {
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	name := createCmd.String("name", "", "Character Name")
	race := createCmd.String("race", "Human", "Race")
	class := createCmd.String("class", "Wizard", "Class")
	background := createCmd.String("background", "Acolyte", "Background")
	level := createCmd.Int("level", 1, "Initial Level (defaults to 1)")
	str := createCmd.Int("str", 10, "Strength Score")
	dex := createCmd.Int("dex", 10, "Dexterity Score")
	con := createCmd.Int("con", 10, "Constitution Score")
	intl := createCmd.Int("int", 10, "Intelligence Score")
	wis := createCmd.Int("wis", 10, "Wisdom Score")
	cha := createCmd.Int("cha", 10, "Charisma Score")
	skills := createCmd.String("skills", "", "Comma-separated list of initial skill proficiencies (e.g., Arcana,History)")

	createCmd.Parse(os.Args[2:])

	if *name == "" {
		fmt.Printf("name is required")
		os.Exit(2)
	}

	scores := map[string]int{"STR": *str, "DEX": *dex, "CON": *con, "INT": *intl, "WIS": *wis, "CHA": *cha}

	var initialSkills []string
	if *skills != "" {
		for _, s := range strings.Split(*skills, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				initialSkills = append(initialSkills, s)
			}
		}
	}

	req := application.CreateCharacterRequest{
		Name: *name, Race: *race, Class: *class, Background: *background, Level: *level,
		ScoreAssignments: scores, InitialSkills: initialSkills,
	}

	char, err := service.CreateCharacter(ctx, req)
	if err != nil {
		fmt.Printf("Error creating character: %v\n", err)
		return
	}

	fmt.Printf("saved character %s", char.Name)
}

func handleView(ctx context.Context, service *application.CharacterService) {
	viewCmd := flag.NewFlagSet("view", flag.ExitOnError)
	name := viewCmd.String("name", "", "Character Name (required)")
	viewCmd.Parse(os.Args[2:])

	if *name == "" {
		fmt.Println("Error: Character name is required.")
		viewCmd.PrintDefaults()
		return
	}

	char, err := service.GetCharacter(ctx, *name)
	if err != nil {
		fmt.Printf(`character "%s" not found`, *name)
		return
	}

	displayCharacterSheet(char)
}

func handleList(ctx context.Context, service *application.CharacterService) {
	flag.NewFlagSet("list", flag.ExitOnError).Parse(os.Args[2:])

	chars, err := service.ListCharacters(ctx)
	if err != nil {
		fmt.Printf("Error listing characters: %v\n", err)
		return
	}

	fmt.Println("--- Character List ---")
	sort.Slice(chars, func(i, j int) bool {
		return chars[i].Name < chars[j].Name
	})

	for _, char := range chars {
		fmt.Printf("%s: Lvl %d %s %s\n", char.Name, char.Level, char.Race, char.Class)
	}
}

func handleUpdateLevel(ctx context.Context, service *application.CharacterService) {
	updateCmd := flag.NewFlagSet("update-level", flag.ExitOnError)
	name := updateCmd.String("name", "", "Character Name")
	level := updateCmd.Int("level", 0, "New Level (1-20)")
	updateCmd.Parse(os.Args[2:])

	if *name == "" || *level < 1 || *level > 20 {
		fmt.Println("Error: Valid character name and level (1-20) are required.")
		updateCmd.PrintDefaults()
		return
	}

	err := service.UpdateCharacterLevel(ctx, *name, *level)
	if err != nil {
		fmt.Printf("Error updating character '%s': %v\n", *name, err)
		return
	}
	fmt.Printf("Success! Character '%s' updated to Level %d.\n", *name, *level)
}

func handleEquip(ctx context.Context, service *application.CharacterService) {
	equipCmd := flag.NewFlagSet("equip", flag.ExitOnError)
	name := equipCmd.String("name", "", "Character Name")

	// Use three specific flags for item type, allowing only one to be set
	weapon := equipCmd.String("weapon", "", "Weapon Name")
	armor := equipCmd.String("armor", "", "Armor Name")
	shield := equipCmd.String("shield", "", "Shield Name")

	slot := equipCmd.String("slot", "", "Equipment slot")

	equipCmd.Parse(os.Args[2:])

	if *name == "" {
		fmt.Println("Error: Character name is required.")
		equipCmd.PrintDefaults()
		return
	}

	itemName, itemType, itemSlot := "", "", ""

	if *weapon != "" {
		itemName = *weapon
		itemType = "weapon"
		itemSlot = strings.ToLower(*slot)

		if itemSlot == "" {
			fmt.Println("Error: When equipping a weapon, the -slot (e.g., 'main hand') is required.")
			equipCmd.PrintDefaults()
			return
		}

	} else if *armor != "" {
		itemName = *armor
		itemType = "armor"
	} else if *shield != "" {
		itemName = *shield
		itemType = "shield"
	} else {
		fmt.Println("Error: Must specify one item type (e.g., -weapon Longsword or -armor Plate Armor).")
		equipCmd.PrintDefaults()
		return
	}

	err := service.EquipItem(ctx, *name, itemName, itemType, itemSlot)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}

	if itemType == "weapon" {
		fmt.Printf("Equipped weapon %s to %s", itemName, itemSlot)
	} else {
		fmt.Printf("Equipped %s %s", itemType, itemName)
	}
}

func displayCharacterSheet(char *domain.Character) {
	fmt.Printf("Name: %s\n", char.Name)
	fmt.Printf("Class: %s\n", strings.ToLower(char.Class))
	fmt.Printf("Race: %s\n", strings.ToLower(char.Race))
	fmt.Printf("Background: %s\n", strings.ToLower(char.Background))
	fmt.Printf("Level: %d\n", char.Level)

	fmt.Println("Ability scores:")
	for _, ab := range []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"} {
		score := char.AbilityScores[ab]
		fmt.Printf("  %s: %d (%+d)\n", ab, score.Score, score.Modifier)
	}

	fmt.Printf("Proficiency bonus: +%d\n", char.ProficiencyBonus)

	var proficiencies []string

	for skill, proficient := range char.SkillProficiencies {
		if proficient {
			lowerSkill := strings.ToLower(skill)
			proficiencies = append(proficiencies, lowerSkill)

			if char.SkillExpertise[skill] {
				proficiencies = append(proficiencies, lowerSkill)
			}
		}
	}
	sort.Strings(proficiencies)

	fmt.Printf("Skill proficiencies: %s\n", strings.Join(proficiencies, ", "))

	hasSpellSlots := false
	for _, count := range char.MaxSpellSlots {
		if count > 0 {
			hasSpellSlots = true
			break
		}
	}

	if hasSpellSlots {
		fmt.Println("Spell slots:")
		var levels []int
		for level := range char.MaxSpellSlots {
			levels = append(levels, level)
		}
		sort.Ints(levels)

		for _, level := range levels {
			if char.MaxSpellSlots[level] > 0 {
				if level == 0 {
					fmt.Printf("  Level 0: %d\n", char.MaxSpellSlots[level])
				} else {
					fmt.Printf("  Level %d: %d\n", level, char.MaxSpellSlots[level])
				}
			}
		}
	}

	if char.SpellCastingAbility != "" && hasSpellSlots && strings.ToLower(char.Class) != "warlock" {
		displayAbility := char.SpellCastingAbility
		if fullName, ok := domain.AllAbilities[char.SpellCastingAbility]; ok {
			displayAbility = fullName
		}

		fmt.Printf("Spellcasting ability: %s\n", displayAbility)
		fmt.Printf("Spell save DC: %d\n", char.SpellSaveDC)
		fmt.Printf("Spell attack bonus: %+d\n", char.SpellAttackBonus)
	}

	if char.EquippedWeaponMainHand.Name != "" {
		fmt.Printf("Main hand: %s\n", char.EquippedWeaponMainHand.Name)
	}

	if char.EquippedWeaponOffHand.Name != "" && !char.EquippedWeaponMainHand.TwoHanded {
		fmt.Printf("Off hand: %s\n", char.EquippedWeaponOffHand.Name)
	}

	if char.EquippedArmor.Name != "" {
		fmt.Printf("Armor: %s\n", char.EquippedArmor.Name)
	}

	if char.EquippedShield.Name != "" {
		fmt.Printf("Shield: %s\n", char.EquippedShield.Name)
	}

	fmt.Printf("Armor class: %d\n", char.ArmorClass)
	fmt.Printf("Initiative bonus: %d\n", char.Initiative)
	fmt.Printf("Passive perception: %d\n", char.PassivePerception)
}

func handleLearnSpell(ctx context.Context, service *application.CharacterService) {
	learnCmd := flag.NewFlagSet("learn-spell", flag.ExitOnError)
	name := learnCmd.String("name", "", "Character Name")
	spell := learnCmd.String("spell", "", "Spell Name to learn (e.g., Fireball)")
	learnCmd.Parse(os.Args[2:])

	if *name == "" || *spell == "" {
		fmt.Println("Error: Character name and spell name are required.")
		learnCmd.PrintDefaults()
		return
	}

	err := service.LearnSpell(ctx, *name, *spell)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}

	fmt.Printf("Learned spell %s", *spell)
}

func handlePrepareSpell(ctx context.Context, service *application.CharacterService) {
	prepareCmd := flag.NewFlagSet("prepare-spell", flag.ExitOnError)
	name := prepareCmd.String("name", "", "Character Name")
	spell := prepareCmd.String("spell", "", "Spell Name to prepare (e.g., Magic Missile)")
	prepareCmd.Parse(os.Args[2:])

	if *name == "" || *spell == "" {
		fmt.Println("Error: Character name and spell name are required.")
		prepareCmd.PrintDefaults()
		return
	}

	err := service.PrepareSpell(ctx, *name, *spell)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}

	fmt.Printf("Prepared spell %s", *spell)
}

func handleDelete(ctx context.Context, service *application.CharacterService) {
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	name := deleteCmd.String("name", "", "Character Name (required)")
	deleteCmd.Parse(os.Args[2:])

	if *name == "" {
		fmt.Println("Error: Character name is required.")
		deleteCmd.PrintDefaults()
		return
	}

	err := service.DeleteCharacter(ctx, *name)
	if err != nil {
		fmt.Printf("Error deleting character '%s': %v\n", *name, err)
		return
	}

	fmt.Printf("deleted %s", *name)
}
