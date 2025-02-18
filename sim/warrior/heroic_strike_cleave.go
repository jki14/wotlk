package warrior

import (
	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (warrior *Warrior) registerHeroicStrikeSpell() {
	cost := 15.0 - float64(warrior.Talents.ImprovedHeroicStrike) - float64(warrior.Talents.FocusedRage)
	refundAmount := cost * 0.8

	hasGlyph := warrior.HasMajorGlyph(proto.WarriorMajorGlyph_GlyphOfHeroicStrike)
	var rageMetrics *core.ResourceMetrics
	if hasGlyph {
		rageMetrics = warrior.NewRageMetrics(core.ActionID{ItemID: 43418})
	}

	warrior.HeroicStrikeOrCleave = warrior.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 47450},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskMeleeMHAuto | core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagIncludeTargetBonusDamage | core.SpellFlagNoOnCastComplete,

		ResourceType: stats.Rage,
		BaseCost:     cost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: cost,
			},

			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				if warrior.glyphOfRevengeProcAura != nil {
					if warrior.glyphOfRevengeProcAura.IsActive() {
						cast.Cost = 0

						warrior.glyphOfRevengeProcAura.Deactivate(sim)
					}
				}
			},
		},

		BonusCritRating:  (float64(warrior.Talents.Incite)*5 + core.TernaryFloat64(warrior.HasSetBonus(ItemSetWrynnsBattlegear, 4), 5, 0)) * core.CritRatingPerCritChance,
		DamageMultiplier: 1,
		CritMultiplier:   warrior.critMultiplier(mh),
		ThreatMultiplier: 1,
		FlatThreatBonus:  259,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := 495 +
				spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower()) +
				spell.BonusWeaponDamage()

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if sim.CurrentTime < warrior.disableHsCleaveUntil {
				return
			}
			if !result.Landed() {
				warrior.AddRage(sim, refundAmount, warrior.RageRefundMetrics)
			}
			if result.DidCrit() && hasGlyph {
				warrior.AddRage(sim, 10, rageMetrics)
			}

			spell.DealDamage(sim, result)
		},
	})
}

func (warrior *Warrior) registerCleaveSpell() {
	cost := 20.0 - float64(warrior.Talents.FocusedRage)
	flatDamageBonus := 222 * (1 + 0.4*float64(warrior.Talents.ImprovedCleave))

	targets := core.TernaryInt32(warrior.HasMajorGlyph(proto.WarriorMajorGlyph_GlyphOfCleaving), 3, 2)
	numHits := core.MinInt32(targets, warrior.Env.GetNumTargets())
	results := make([]*core.SpellEffect, numHits)

	warrior.HeroicStrikeOrCleave = warrior.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 47520},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskMeleeMHAuto | core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagIncludeTargetBonusDamage,

		ResourceType: stats.Rage,
		BaseCost:     cost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: cost,
			},
		},

		BonusCritRating:  float64(warrior.Talents.Incite) * 5 * core.CritRatingPerCritChance,
		DamageMultiplier: 1,
		CritMultiplier:   warrior.critMultiplier(mh),
		ThreatMultiplier: 1,
		FlatThreatBonus:  225,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			curTarget := target
			for hitIndex := int32(0); hitIndex < numHits; hitIndex++ {
				baseDamage := flatDamageBonus +
					spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower()) +
					spell.BonusWeaponDamage()
				results[hitIndex] = spell.CalcDamage(sim, curTarget, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

				curTarget = sim.Environment.NextTargetUnit(curTarget)
			}

			curTarget = target
			for hitIndex := int32(0); hitIndex < numHits; hitIndex++ {
				spell.DealDamage(sim, results[hitIndex])
				curTarget = sim.Environment.NextTargetUnit(curTarget)
			}
		},
	})
}

func (warrior *Warrior) QueueHSOrCleave(sim *core.Simulation) {
	if warrior.CurrentRage() < warrior.HeroicStrikeOrCleave.DefaultCast.Cost {
		panic("Not enough rage for HS")
	}
	if warrior.HSOrCleaveQueueAura.IsActive() {
		return
	}
	warrior.HSOrCleaveQueueAura.Activate(sim)
	warrior.PseudoStats.DisableDWMissPenalty = true
}

func (warrior *Warrior) DequeueHSOrCleave(sim *core.Simulation) {
	warrior.HSOrCleaveQueueAura.Deactivate(sim)
	warrior.PseudoStats.DisableDWMissPenalty = false
}

// Returns true if the regular melee swing should be used, false otherwise.
func (warrior *Warrior) TryHSOrCleave(sim *core.Simulation, mhSwingSpell *core.Spell) *core.Spell {
	if !warrior.HSOrCleaveQueueAura.IsActive() {
		return nil
	}

	if sim.CurrentTime < warrior.disableHsCleaveUntil {
		warrior.DequeueHSOrCleave(sim)
		return nil
	}

	if warrior.CurrentRage() < warrior.HeroicStrikeOrCleave.DefaultCast.Cost {
		warrior.DequeueHSOrCleave(sim)
		return nil
	} else if warrior.CurrentRage() < warrior.HSRageThreshold {
		if mhSwingSpell == warrior.AutoAttacks.MHAuto {
			warrior.DequeueHSOrCleave(sim)
			return nil
		}
	}

	warrior.DequeueHSOrCleave(sim)
	return warrior.HeroicStrikeOrCleave
}

func (warrior *Warrior) ShouldQueueHSOrCleave(sim *core.Simulation) bool {
	return warrior.CurrentRage() >= warrior.HSRageThreshold
}

func (warrior *Warrior) RegisterHSOrCleave(useCleave bool, rageThreshold float64) {
	if useCleave {
		warrior.registerCleaveSpell()
	} else {
		warrior.registerHeroicStrikeSpell()
	}

	warrior.HSOrCleaveQueueAura = warrior.RegisterAura(core.Aura{
		Label:    "HS Queue Aura",
		ActionID: warrior.HeroicStrikeOrCleave.ActionID,
		Duration: core.NeverExpires,
	})

	warrior.HSRageThreshold = core.MaxFloat(warrior.HeroicStrikeOrCleave.DefaultCast.Cost, rageThreshold)
}
