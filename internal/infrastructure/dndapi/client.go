package dndapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dnd-char-generator/internal/domain"
)

const baseURL = "https://www.dnd5eapi.co/api/"
const maxRequestsPerSecond = 8

type Client struct {
	httpClient  *http.Client
	rateLimiter chan time.Time
}

func NewClient() *Client {
	limiter := make(chan time.Time, maxRequestsPerSecond)

	go func() {
		// Interval to add a token (1 second / max requests)
		interval := time.Second / time.Duration(maxRequestsPerSecond)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for t := range ticker.C {
			select {
			case limiter <- t:
				// Token added
			default:
				// If the channel is full, skip
			}
		}
	}()

	return &Client{
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		rateLimiter: limiter,
	}
}

func (c *Client) getResource(ctx context.Context, endpoint string, target interface{}) error {
	// Wait
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.rateLimiter:
		// Token consumed, proceed with the request
	}

	// Build and send request
	url := baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", endpoint, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed for %s with status: %d", endpoint, resp.StatusCode)
	}

	// Decode response
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode response for %s: %w", endpoint, err)
	}
	return nil
}

func (c *Client) EnrichSpell(ctx context.Context, spell *domain.Spell) {
	if spell.Name == "" {
		return
	}

	spellNameSlug := strings.ToLower(strings.ReplaceAll(spell.Name, " ", "-"))

	var apiSpell struct {
		Range string `json:"range"`

		School struct {
			Name string `json:"name"`
		} `json:"school"`
	}

	endpoint := "spells/" + spellNameSlug
	if err := c.getResource(ctx, endpoint, &apiSpell); err == nil {
		spell.Range = apiSpell.Range
		spell.School = apiSpell.School.Name
	}
}

func (c *Client) EnrichWeapon(ctx context.Context, weapon *domain.Weapon) {
	weaponNameSlug := strings.ToLower(strings.ReplaceAll(weapon.Name, " ", "-"))

	var apiWeapon struct {
		CategoryRange string `json:"category_range"`

		Damage struct {
			DamageDice string `json:"damage_dice"`
			DamageType struct {
				Name string `json:"name"`
			} `json:"damage_type"`
		} `json:"damage"`

		Properties []struct {
			Name string `json:"name"`
		} `json:"properties"`

		Range struct {
			Normal int `json:"normal"`
		} `json:"range"`
	}

	endpoint := "equipment/" + weaponNameSlug
	if err := c.getResource(ctx, endpoint, &apiWeapon); err == nil {
		if apiWeapon.Damage.DamageDice != "" {
			primaryDamage := apiWeapon.Damage

			if primaryDamage.DamageType.Name != "" {
				weapon.Damage = fmt.Sprintf("%s %s",
					primaryDamage.DamageDice,
					primaryDamage.DamageType.Name,
				)
			} else {
				weapon.Damage = primaryDamage.DamageDice
			}
		}

		weapon.Equipment.Category = apiWeapon.CategoryRange

		if apiWeapon.Range.Normal > 5 {
			weapon.Equipment.Range = fmt.Sprintf("%d ft. (Ranged)", apiWeapon.Range.Normal)
		} else {
			weapon.Equipment.Range = "5 ft. (Melee)"
		}

		isTwoHanded := false
		for _, prop := range apiWeapon.Properties {
			if prop.Name == "Two-Handed" {
				isTwoHanded = true
				break
			}
		}
		weapon.TwoHanded = isTwoHanded
	}
}

func (c *Client) EnrichArmor(ctx context.Context, armor *domain.Armor) {
	armorNameSlug := strings.ToLower(strings.ReplaceAll(armor.Name, " ", "-"))

	var apiArmor struct {
		ArmorCategory string `json:"armor_category"`
	}

	endpoint := "equipment/" + armorNameSlug
	if err := c.getResource(ctx, endpoint, &apiArmor); err == nil {
		armor.Equipment.Category = apiArmor.ArmorCategory
	}
}
