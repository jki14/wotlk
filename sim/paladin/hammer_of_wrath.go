package paladin

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (paladin *Paladin) registerHammerOfWrathSpell() {
	// From the perspective of max rank.
	baseCost := paladin.BaseMana * 0.12

	paladin.HammerOfWrath = paladin.RegisterSpell(core.SpellConfig{
		ActionID:     core.ActionID{SpellID: 48806},
		SpellSchool:  core.SpellSchoolHoly,
		ProcMask:     core.ProcMaskSpellDamage,
		Flags:        core.SpellFlagMeleeMetrics,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost *
					(1 - 0.02*float64(paladin.Talents.Benediction)) *
					core.TernaryFloat64(paladin.HasMajorGlyph(proto.PaladinMajorGlyph_GlyphOfHammerOfWrath), 0, 1),

				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    paladin.NewTimer(),
				Duration: time.Second * 6,
			},
		},

		BonusCritRating: 25 * float64(paladin.Talents.SanctifiedWrath) * core.CritRatingPerCritChance,
		DamageMultiplierAdditive: 1 +
			paladin.getItemSetLightbringerBattlegearBonus4() +
			paladin.getItemSetAegisBattlegearBonus2(),
		DamageMultiplier: 1,
		CritMultiplier:   paladin.MeleeCritMultiplier(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(1139, 1257) +
				.15*spell.SpellPower() +
				.15*spell.MeleeAttackPower()

			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialNoBlockDodgeParry)
		},
	})
}
