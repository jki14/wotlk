package warlock

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (warlock *Warlock) registerIncinerateSpell() {
	baseCost := 0.14 * warlock.BaseMana
	actionID := core.ActionID{SpellID: 47838}
	spellSchool := core.SpellSchoolFire
	spellCoeff := 0.713 * (1 + 0.04*float64(warlock.Talents.ShadowAndFlame))

	warlock.Incinerate = warlock.RegisterSpell(core.SpellConfig{
		ActionID:     actionID,
		SpellSchool:  spellSchool,
		ProcMask:     core.ProcMaskSpellDamage,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost:     baseCost * (1 - []float64{0, .04, .07, .10}[warlock.Talents.Cataclysm]),
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * time.Duration(2500-50*warlock.Talents.Emberstorm),
			},
			ModifyCast: func(_ *core.Simulation, _ *core.Spell, cast *core.Cast) {
				cast.GCD = time.Duration(float64(cast.GCD) * warlock.backdraftModifier())
				cast.CastTime = time.Duration(float64(cast.CastTime) * warlock.moltenCoreIncinerateModifier() * warlock.backdraftModifier())
			},
		},

		BonusCritRating: 0 +
			warlock.masterDemonologistFireCrit() +
			core.TernaryFloat64(warlock.Talents.Devastation, 5*core.CritRatingPerCritChance, 0) +
			core.TernaryFloat64(warlock.HasSetBonus(ItemSetDeathbringerGarb, 4), 5*core.CritRatingPerCritChance, 0) +
			core.TernaryFloat64(warlock.HasSetBonus(ItemSetDarkCovensRegalia, 2), 5*core.CritRatingPerCritChance, 0),
		DamageMultiplierAdditive: warlock.staticAdditiveDamageMultiplier(actionID, spellSchool, false),
		CritMultiplier:           warlock.SpellCritMultiplier(1, float64(warlock.Talents.Ruin)/5),
		ThreatMultiplier:         1 - 0.1*float64(warlock.Talents.DestructiveReach),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(582, 676) + spellCoeff*spell.SpellPower()
			if warlock.ImmolateDot.IsActive() {
				baseDamage += 157 //  145 to 169 averages to 157
			}

			spell.CalcAndDealDamageMagicHitAndCrit(sim, target, baseDamage)
		},
	})
}

func (warlock *Warlock) moltenCoreIncinerateModifier() float64 {
	castTimeModifier := 1.0
	if warlock.MoltenCoreAura.IsActive() {
		castTimeModifier *= (1.0 - 0.1*float64(warlock.Talents.MoltenCore))
	}
	return castTimeModifier
}
