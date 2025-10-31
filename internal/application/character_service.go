package application

import (
	"context"
	"dnd-char-generator/internal/domain"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type CharacterRepository interface {
	Save(ctx context.Context, char *domain.Character) error
	FindByID(ctx context.Context, name string) (*domain.Character, error)
	FindAll(ctx context.Context) ([]*domain.Character, error)
	Delete(ctx context.Context, name string) error
}

type DndAPIClient interface {
	EnrichSpell(ctx context.Context, spells *domain.Spell)
	EnrichWeapon(ctx context.Context, weapon *domain.Weapon)
	EnrichArmor(ctx context.Context, armor *domain.Armor)
}

type CreateCharacterRequest struct {
	Name             string
	Race             string
	Class            string
	Background       string
	Level            int
	ScoreAssignments map[string]int
	InitialSkills    []string
}

type CharacterService struct {
	Repo       CharacterRepository
	ApiClient  DndAPIClient
	AllSpells  map[int][]domain.Spell
	AllWeapons map[string]domain.Weapon
	AllArmors  map[string]domain.Armor
	AllShields map[string]domain.Shield
}

func NewCharacterService(
	repo CharacterRepository,
	apiClient DndAPIClient,
	spells map[int][]domain.Spell,
	weapons map[string]domain.Weapon,
	armors map[string]domain.Armor,
	shields map[string]domain.Shield,
) *CharacterService {
	return &CharacterService{
		Repo:       repo,
		ApiClient:  apiClient,
		AllSpells:  spells,
		AllWeapons: weapons,
		AllArmors:  armors,
		AllShields: shields,
	}
}

func (s *CharacterService) CreateCharacter(ctx context.Context, req CreateCharacterRequest) (*domain.Character, error) {
	newChar, err := domain.NewCharacter(
		req.Name,
		req.Race,
		req.Class,
		req.Background,
		req.ScoreAssignments,
	)
	if err != nil {
		return nil, fmt.Errorf("domain creation failed: %w", err)
	}

	newChar.Level = req.Level

	if len(req.InitialSkills) == 0 {
		bgName := strings.ToLower(req.Background)
		className := strings.ToLower(req.Class)

		bgSkills := BackgroundSkills[bgName]
		classSkills := DefaultClassSkills[className]

		bgSkillMap := make(map[string]bool)
		for _, skill := range bgSkills {
			bgSkillMap[skill] = true
			req.InitialSkills = append(req.InitialSkills, skill)
		}

		for _, skill := range classSkills {
			req.InitialSkills = append(req.InitialSkills, skill)

			if bgSkillMap[skill] {
				newChar.SkillExpertise[skill] = true
			}
		}
	}

	newChar.SetSkillProficiencies(req.InitialSkills)

	newChar.UpdateProficiencyBonus(newChar.Level)
	newChar.CalculateMaxHitPoints()
	newChar.CalculateCombatStats()
	newChar.CalculateSpellStats()
	newChar.CalculateMaxSpellSlots()

	if err := s.Repo.Save(ctx, newChar); err != nil {
		return nil, fmt.Errorf("failed to save new character: %w", err)
	}

	return newChar, nil
}

func (s *CharacterService) GetCharacter(ctx context.Context, name string) (*domain.Character, error) {
	char, err := s.Repo.FindByID(ctx, name)
	if err != nil {
		return nil, err
	}

	char.CalculateMaxHitPoints()
	char.UpdateProficiencyBonus(char.Level)
	char.CalculateCombatStats()
	char.CalculateSpellStats()

	return char, nil
}

func (s *CharacterService) DeleteCharacter(ctx context.Context, name string) error {
	return s.Repo.Delete(ctx, name)
}

func (s *CharacterService) enrichWeapon(ctx context.Context, w *domain.Weapon, wg *sync.WaitGroup) {
	defer wg.Done()
	s.ApiClient.EnrichWeapon(ctx, w)
}

func (s *CharacterService) enrichArmor(ctx context.Context, a *domain.Armor, wg *sync.WaitGroup) {
	defer wg.Done()
	s.ApiClient.EnrichArmor(ctx, a)
}

func (s *CharacterService) enrichSpell(ctx context.Context, sp *domain.Spell, wg *sync.WaitGroup) {
	defer wg.Done()
	s.ApiClient.EnrichSpell(ctx, sp)
}

func (s *CharacterService) ListCharacters(ctx context.Context) ([]*domain.Character, error) {
	chars, err := s.Repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	sort.Slice(chars, func(i, j int) bool {
		return chars[i].Name < chars[j].Name
	})

	return chars, nil
}

func (s *CharacterService) UpdateCharacterLevel(ctx context.Context, name string, level int) error {
	char, err := s.Repo.FindByID(ctx, name)
	if err != nil {
		return err
	}

	char.Level = level

	char.UpdateProficiencyBonus(char.Level)

	char.CalculateMaxHitPoints()
	char.CalculateMaxSpellSlots()
	char.CalculateSpellStats()
	char.CalculateCombatStats()

	return s.Repo.Save(ctx, char)
}

func (s *CharacterService) filterSpellsByLevel(level int) []domain.Spell {
	if spells, ok := s.AllSpells[level]; ok {
		return spells
	}
	return nil
}

func (s *CharacterService) EquipItem(ctx context.Context, name, itemName, itemType, slot string) error {
	char, err := s.Repo.FindByID(ctx, name)
	if err != nil {
		return err
	}

	rateLimiter := time.NewTicker(time.Millisecond * 100)
	defer rateLimiter.Stop()

	var wg sync.WaitGroup

	switch strings.ToLower(itemType) {
	case "weapon":
		if w, ok := s.AllWeapons[itemName]; ok {
			var equippedWeapon *domain.Weapon
			if slot == "main hand" {
				equippedWeapon = &char.EquippedWeaponMainHand
			} else if slot == "off hand" {
				equippedWeapon = &char.EquippedWeaponOffHand
			}

			if err := char.EquipWeaponSlot(w, slot); err != nil {
				return err
			}

			if equippedWeapon != nil {
				wg.Add(1)
				<-rateLimiter.C
				go s.enrichWeapon(ctx, equippedWeapon, &wg)
				wg.Wait()

				if slot == "off hand" && char.EquippedWeaponMainHand.TwoHanded {
					char.EquippedWeaponOffHand = domain.Weapon{}
					return fmt.Errorf("cannot equip to off hand: main hand weapon '%s' is two-handed", char.EquippedWeaponMainHand.Name)
				}

				if slot == "main hand" && equippedWeapon.TwoHanded {
					char.EquippedWeaponOffHand = domain.Weapon{}
				}
			} else {
				return fmt.Errorf("invalid equipment slot specified")
			}
		} else {
			return fmt.Errorf("weapon '%s' not found in SRD data", itemName)
		}
	case "armor":
		if a, ok := s.AllArmors[itemName]; ok {
			char.EquippedArmor = a
			wg.Add(1)
			<-rateLimiter.C
			go s.enrichArmor(ctx, &char.EquippedArmor, &wg)
		} else {
			return fmt.Errorf("armor '%s' not found in SRD data", itemName)
		}
	case "shield":
		if sh, ok := s.AllShields[itemName]; ok {
			char.EquippedShield = sh
		} else {
			return fmt.Errorf("shield '%s' not found in SRD data", itemName)
		}
	default:
		return fmt.Errorf("invalid item type: %s. Must be 'weapon', 'armor', or 'shield'", itemType)
	}

	wg.Wait()

	char.CalculateCombatStats()

	if err := s.Repo.Save(ctx, char); err != nil {
		return fmt.Errorf("failed to save character after equipping item: %w", err)
	}

	return nil
}

func (s *CharacterService) LearnSpell(ctx context.Context, charName, spellName string) error {
	char, err := s.Repo.FindByID(ctx, charName)
	if err != nil {
		return err
	}

	if char.SpellcasterType == domain.NoSpellcasting {
		return fmt.Errorf("this class can't cast spells")
	}

	if char.SpellcasterType == domain.PreparedCasting {
		return fmt.Errorf("this class prepares spells and can't learn them")
	}

	var spellToLearn domain.Spell
	found := false

	normalizedSpellName := strings.ToLower(strings.TrimSpace(spellName))

	for _, spellsAtLevel := range s.AllSpells {
		for _, spell := range spellsAtLevel {
			if strings.EqualFold(spell.Name, normalizedSpellName) {
				spellToLearn = spell
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("spell '%s' not found in SRD spell list", spellName)
	}

	isCasterForSpell := false
	for _, class := range spellToLearn.Class {
		if strings.EqualFold(class, char.Class) {
			isCasterForSpell = true
			break
		}
	}

	if !isCasterForSpell {
		return fmt.Errorf("character class '%s' is not listed as a caster for spell '%s'", char.Class, spellName)
	}

	if _, ok := char.KnownSpells[normalizedSpellName]; ok {
		return fmt.Errorf("character '%s' already knows the spell '%s'", charName, spellName)
	}

	char.KnownSpells[normalizedSpellName] = spellToLearn

	spellCopy := char.KnownSpells[normalizedSpellName]
	rateLimiter := time.NewTicker(time.Millisecond * 100)
	defer rateLimiter.Stop()

	var wg sync.WaitGroup

	wg.Add(1)
	<-rateLimiter.C
	go s.enrichSpell(ctx, &spellCopy, &wg)

	wg.Wait()

	char.KnownSpells[normalizedSpellName] = spellCopy

	char.CalculateSpellStats()
	char.CalculateMaxSpellSlots()

	if err := s.Repo.Save(ctx, char); err != nil {
		return fmt.Errorf("failed to save character after learning spell: %w", err)
	}

	return nil
}

func (s *CharacterService) PrepareSpell(ctx context.Context, charName, spellName string) error {
	char, err := s.Repo.FindByID(ctx, charName)
	if err != nil {
		return err
	}

	if char.SpellcasterType == domain.NoSpellcasting {
		return fmt.Errorf("this class can't cast spells")
	}

	if char.SpellcasterType == domain.LearnedCasting {
		return fmt.Errorf("this class learns spells and can't prepare them")
	}

	normalizedSpellName := strings.ToLower(strings.TrimSpace(spellName))
	var spellToPrepare domain.Spell
	var found bool

	for _, spellsAtLevel := range s.AllSpells {
		for _, spell := range spellsAtLevel {
			if strings.EqualFold(spell.Name, normalizedSpellName) {
				spellToPrepare = spell
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("spell '%s' not found in SRD data", spellName)
	}

	spellLevel := spellToPrepare.Level

	maxSlotLevel := 0
	for level, count := range char.MaxSpellSlots {
		if level > maxSlotLevel && count > 0 {
			maxSlotLevel = level
		}
	}

	if spellLevel > maxSlotLevel && maxSlotLevel > 0 {
		return fmt.Errorf("the spell has higher level than the available spell slots")
	}

	if count, ok := char.MaxSpellSlots[spellLevel]; !ok || count == 0 {
		return fmt.Errorf("the spell has higher level than the available spell slots")
	}

	classData := domain.AllClassesData[strings.ToLower(char.Class)]
	castingAbility := classData.SpellcastingAbility

	ability, abilityOk := char.AbilityScores[castingAbility]
	if !abilityOk {
		return fmt.Errorf("class requires spellcasting ability %s, but score is missing", castingAbility)
	}

	preparationLimit := (char.Level / 2) + ability.Modifier

	if preparationLimit < 1 {
		preparationLimit = 1
	}

	if len(char.PreparedSpells) >= preparationLimit {
		return fmt.Errorf("character '%s' has reached the limit of %d prepared spells (Lvl %d + %s Mod %+d)",
			charName, preparationLimit, char.Level, castingAbility, ability.Modifier)
	}

	if _, ok := char.PreparedSpells[normalizedSpellName]; ok {
		return fmt.Errorf("spell '%s' is already prepared", spellName)
	}

	char.PreparedSpells[normalizedSpellName] = spellToPrepare

	spellCopy := char.PreparedSpells[normalizedSpellName]
	rateLimiter := time.NewTicker(time.Millisecond * 100)
	defer rateLimiter.Stop()

	var wg sync.WaitGroup

	wg.Add(1)
	<-rateLimiter.C
	go s.enrichSpell(ctx, &spellCopy, &wg)

	wg.Wait()

	char.PreparedSpells[normalizedSpellName] = spellCopy

	if err := s.Repo.Save(ctx, char); err != nil {
		return fmt.Errorf("failed to save prepared spell: %w", err)
	}

	return nil
}
