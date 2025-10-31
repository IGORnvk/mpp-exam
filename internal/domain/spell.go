package domain

type Spell struct {
	Name   string
	Level  int
	Class  []string
	School string
	Range  string
}

var AllSpells map[int][]Spell
