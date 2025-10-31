package domain

type Equipment struct {
	Name     string
	Type     string
	Category string
	Range    string
}

type Weapon struct {
	Equipment
	Damage    string
	TwoHanded bool
}

type Armor struct {
	Equipment
	AC       int
	DexBonus string
}

type Shield struct {
	Equipment
}

var AllWeapons map[string]Weapon
var AllArmors map[string]Armor
var AllShields map[string]Shield
