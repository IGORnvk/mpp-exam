package infrastructure

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"dnd-char-generator/internal/domain"
)

func LoadData(equipmentPath string, spellsPath string) (
	map[int][]domain.Spell,
	map[string]domain.Weapon,
	map[string]domain.Armor,
	map[string]domain.Shield,
	error) {

	allSpells, err := LoadSpellData(spellsPath)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	allWeapons, allArmors, allShields, err := LoadEquipmentData(equipmentPath)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return allSpells, allWeapons, allArmors, allShields, nil
}

func LoadSpellData(filePath string) (map[int][]domain.Spell, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open spells file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("could not close file: %v", err)
		}
	}(file)

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading spells CSV: %w", err)
	}

	spellsByLevel := make(map[int][]domain.Spell)

	for i, record := range records {
		if i == 0 {
			continue
		}

		name := record[0]
		levelStr := record[1]
		classStr := record[2]

		levelStr = strings.TrimSpace(levelStr)
		level, err := strconv.Atoi(levelStr)
		if err != nil {
			fmt.Printf("Warning: Skipping spell '%s' with invalid level '%s'\n", name, levelStr)
			continue
		}

		// Split classes and clean up quotes/spaces
		classNames := strings.Split(strings.ReplaceAll(classStr, "\"", ""), ",")
		for i := range classNames {
			classNames[i] = strings.TrimSpace(classNames[i])
		}

		spell := domain.Spell{
			Name:  strings.ToLower(name),
			Level: level,
			Class: classNames,
		}

		spellsByLevel[level] = append(spellsByLevel[level], spell)
	}

	return spellsByLevel, nil
}

func LoadEquipmentData(filePath string) (map[string]domain.Weapon, map[string]domain.Armor, map[string]domain.Shield, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not open equipment file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("could not close file: %v", err)
		}
	}(file)

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error reading equipment CSV: %w", err)
	}

	allWeapons := make(map[string]domain.Weapon)
	allArmors := make(map[string]domain.Armor)
	allShields := make(map[string]domain.Shield)

	for i, record := range records {
		if i == 0 {
			continue
		}

		name := record[0]
		itemType := record[1]

		key := strings.ToLower(name)

		baseEq := domain.Equipment{
			Name: key,
			Type: itemType,
		}

		switch itemType {
		case "Weapon":
			allWeapons[key] = domain.Weapon{
				Equipment: baseEq,
			}

		case "Armor":
			if strings.EqualFold(name, "Shield") {
				allShields[key] = domain.Shield{Equipment: baseEq}
				continue
			}

			armor := domain.Armor{
				Equipment: baseEq,
			}

			switch name {
			// Light armor
			case "Padded Armor":
				armor.AC, armor.DexBonus = 11, "full"
			case "Leather Armor":
				armor.AC, armor.DexBonus = 11, "full"
			case "Studded Leather Armor":
				armor.AC, armor.DexBonus = 12, "full"

			// Medium armor
			case "Hide Armor":
				armor.AC, armor.DexBonus = 12, "limited"
			case "Chain Shirt":
				armor.AC, armor.DexBonus = 13, "limited"
			case "Scale Mail":
				armor.AC, armor.DexBonus = 14, "limited"
			case "Breastplate":
				armor.AC, armor.DexBonus = 14, "limited"
			case "Half Plate":
				armor.AC, armor.DexBonus = 15, "limited"

			// Heavy armor
			case "Ring Mail":
				armor.AC, armor.DexBonus = 14, "none"
			case "Chain Mail":
				armor.AC, armor.DexBonus = 16, "none"
			case "Splint Armor":
				armor.AC, armor.DexBonus = 17, "none"
			case "Plate Armor":
				armor.AC, armor.DexBonus = 18, "none"

			default:
				continue
			}

			allArmors[key] = armor

		case "Shield":
			allShields[key] = domain.Shield{
				Equipment: baseEq,
			}
		}
	}

	return allWeapons, allArmors, allShields, nil
}
