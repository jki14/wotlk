package shaman

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/items"
	"github.com/wowsims/wotlk/sim/core/stats"
)

// Totem Item IDs
const (
	StormfuryTotem           = 31031
	TotemOfAncestralGuidance = 32330
	TotemOfStorms            = 23199
	TotemOfThePulsingEarth   = 29389
	TotemOfTheVoid           = 28248
	TotemOfHex               = 40267
	VentureCoLightningRod    = 38361
	ThunderfallTotem         = 45255
)

const (
	// This could be value or bitflag if we ended up needing multiple flags at the same time.
	//1 to 5 are used by MaelstromWeapon Stacks
	CastTagLightningOverload int32 = 6
)

// Mana cost numbers based on in-game testing:
//
// With 5/5 convection:
// Normal: 270, w/ EF: 150
//
// With 5/5 convection and TotPE equipped:
// Normal: 246, w/ EF: 136

// Shared precomputation logic for LB and CL.
func (shaman *Shaman) newElectricSpellConfig(actionID core.ActionID, baseCost float64, baseCastTime time.Duration, isLightningOverload bool) core.SpellConfig {
	spell := core.SpellConfig{
		ActionID:     actionID,
		SpellSchool:  core.SpellSchoolNature,
		ProcMask:     core.ProcMaskSpellDamage,
		Flags:        SpellFlagElectric | SpellFlagFocusable,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost:     baseCost,
				CastTime: baseCastTime,
				GCD:      core.GCDDefault,
			},
		},

		BonusHitRating: float64(shaman.Talents.ElementalPrecision) * core.SpellHitRatingPerHitChance,
		BonusCritRating: 0 +
			(float64(shaman.Talents.TidalMastery) * 1 * core.CritRatingPerCritChance) +
			core.TernaryFloat64(shaman.Talents.CallOfThunder, 5*core.CritRatingPerCritChance, 0),
		DamageMultiplier: 1 * (1 + 0.01*float64(shaman.Talents.Concussion)),
		CritMultiplier:   shaman.ElementalCritMultiplier(0),
		ThreatMultiplier: 1 - (0.1/3)*float64(shaman.Talents.ElementalPrecision),
	}

	if isLightningOverload {
		spell.ActionID.Tag = CastTagLightningOverload
		spell.ResourceType = 0
		spell.Cast.DefaultCast.CastTime = 0
		spell.Cast.DefaultCast.GCD = 0
		spell.Cast.DefaultCast.Cost = 0
		spell.DamageMultiplier *= 0.5
		spell.ThreatMultiplier = 0
	} else if shaman.Talents.LightningMastery > 0 {
		// Convection applies against the base cost of the spell.
		spell.Cast.DefaultCast.Cost -= baseCost * float64(shaman.Talents.Convection) * 0.02
		spell.Cast.DefaultCast.CastTime -= time.Millisecond * 100 * time.Duration(shaman.Talents.LightningMastery)
	}

	return spell
}

func (shaman *Shaman) electricSpellBonusDamage(spellCoeff float64) float64 {
	bonusDamage := 0 +
		core.TernaryFloat64(shaman.Equip[items.ItemSlotRanged].ID == TotemOfStorms, 33, 0) +
		core.TernaryFloat64(shaman.Equip[items.ItemSlotRanged].ID == TotemOfTheVoid, 55, 0) +
		core.TernaryFloat64(shaman.Equip[items.ItemSlotRanged].ID == TotemOfAncestralGuidance, 85, 0) +
		core.TernaryFloat64(shaman.Equip[items.ItemSlotRanged].ID == TotemOfHex, 165, 0)

	return bonusDamage * spellCoeff // These items do not benefit from the bonus coeff from shamanism.
}

// Shared LB/CL logic that is dynamic, i.e. can't be precomputed.
func (shaman *Shaman) applyElectricSpellCastInitModifiers(spell *core.Spell, cast *core.Cast) {
	shaman.modifyCastClearcasting(spell, cast)
	if shaman.ElementalMasteryAura.IsActive() {
		cast.CastTime = 0
	}
}
