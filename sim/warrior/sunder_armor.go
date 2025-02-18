package warrior

import (
	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

// TODO: GlyphOfSunderArmor will require refactoring this a bit

var SunderArmorActionID = core.ActionID{SpellID: 47467}

func (warrior *Warrior) newSunderArmorSpell(isDevastateEffect bool) *core.Spell {
	cost := 15.0 - float64(warrior.Talents.FocusedRage) - float64(warrior.Talents.Puncture)
	refundAmount := cost * 0.8
	warrior.SunderArmorAura = core.SunderArmorAura(warrior.CurrentTarget, 0)
	warrior.ExposeArmorAura = core.ExposeArmorAura(warrior.CurrentTarget, false)
	warrior.AcidSpitAura = core.AcidSpitAura(warrior.CurrentTarget, 0)

	config := core.SpellConfig{
		ActionID:    SunderArmorActionID,
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics,

		ResourceType: stats.Rage,
		BaseCost:     cost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: cost,
				GCD:  core.GCDDefault,
			},
			IgnoreHaste: true,
		},

		ThreatMultiplier: 1,
		FlatThreatBonus:  360,
		DynamicThreatBonus: func(spellEffect *core.SpellEffect, spell *core.Spell) float64 {
			return 0.05 * spell.MeleeAttackPower()
		},
	}

	extraStack := isDevastateEffect && warrior.HasMajorGlyph(proto.WarriorMajorGlyph_GlyphOfDevastate)
	if isDevastateEffect {
		config.ResourceType = 0
		config.BaseCost = 0
		config.Cast.DefaultCast.Cost = 0
		config.Cast.DefaultCast.GCD = 0

		// In wrath sunder from devastate generates no threat
		config.ThreatMultiplier = 0
		config.FlatThreatBonus = 0
		config.DynamicThreatBonus = nil
	}

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		var result *core.SpellEffect
		if isDevastateEffect {
			result = spell.CalcOutcome(sim, target, spell.OutcomeAlwaysHit)
		} else {
			result = spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
		}

		if result.Landed() {
			warrior.SunderArmorAura.Activate(sim)
			if warrior.SunderArmorAura.IsActive() {
				warrior.SunderArmorAura.AddStack(sim)
				if extraStack {
					warrior.SunderArmorAura.AddStack(sim)
				}
			}
		} else {
			warrior.AddRage(sim, refundAmount, warrior.RageRefundMetrics)
		}

		spell.DealOutcome(sim, result)
	}
	return warrior.RegisterSpell(config)
}

func (warrior *Warrior) CanSunderArmor(sim *core.Simulation) bool {
	return warrior.CurrentRage() >= warrior.SunderArmor.DefaultCast.Cost &&
		!warrior.ExposeArmorAura.IsActive() &&
		!warrior.AcidSpitAura.IsActive()
}
