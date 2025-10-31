package domain

import "context"

type CharacterRepository interface {
	Save(ctx context.Context, char *Character) error

	FindByID(ctx context.Context, name string) (*Character, error)

	FindAll(ctx context.Context) ([]*Character, error)
}
