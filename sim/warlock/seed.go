package warlock

import (
	"strconv"
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (warlock *Warlock) registerSeedSpell() {
	numTargets := int(warlock.Env.GetNumTargets())

	warlock.Seeds = make([]*core.Spell, numTargets)
	warlock.SeedDots = make([]*core.Dot, numTargets)

	// For this simulation we always assume the seed target didn't die to trigger the seed because we don't simulate health.
	// This effectively lowers the seed AOE cap using the function:
	for i := 0; i < numTargets; i++ {
		warlock.makeSeed(i, numTargets)
	}
}

func (warlock *Warlock) makeSeed(targetIdx int, numTargets int) {
	target := warlock.Env.GetTargetUnit(int32(targetIdx))
	baseCost := 0.34 * warlock.BaseMana
	actionID := core.ActionID{SpellID: 47836, Tag: 1}
	spellSchool := core.SpellSchoolShadow

	seedExplosion := warlock.RegisterSpell(core.SpellConfig{
		ActionID:    actionID,
		SpellSchool: spellSchool,
		ProcMask:    core.ProcMaskSpellDamage,

		BonusCritRating: 0 +
			warlock.masterDemonologistShadowCrit() +
			float64(warlock.Talents.ImprovedCorruption)*core.CritRatingPerCritChance,
		DamageMultiplierAdditive: warlock.staticAdditiveDamageMultiplier(actionID, spellSchool, false),
		CritMultiplier:           warlock.DefaultSpellCritMultiplier(),
		ThreatMultiplier:         1 - 0.1*float64(warlock.Talents.ImprovedDrainSoul),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			dmgFromSP := 0.2129 * spell.SpellPower()
			for _, aoeTarget := range sim.Encounter.Targets {
				// Seeded target is not affected by explosion.
				if &aoeTarget.Unit == target {
					continue
				}

				baseDamage := sim.Roll(1633, 1897) + dmgFromSP
				baseDamage *= sim.Encounter.AOECapMultiplier()
				spell.CalcAndDealDamageMagicHitAndCrit(sim, &aoeTarget.Unit, baseDamage)
			}
		},
	})

	warlock.Seeds[targetIdx] = warlock.RegisterSpell(core.SpellConfig{
		ActionID:     actionID,
		SpellSchool:  spellSchool,
		ProcMask:     core.ProcMaskEmpty,
		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost:     baseCost * (1 - 0.02*float64(warlock.Talents.Suppression)),
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * 2000,
			},
		},

		DamageMultiplierAdditive: warlock.staticAdditiveDamageMultiplier(actionID, spellSchool, true),
		ThreatMultiplier:         1 - 0.1*float64(warlock.Talents.ImprovedDrainSoul),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				if warlock.Rotation.DetonateSeed {
					seedExplosion.Cast(sim, target)
				} else {
					warlock.SeedDots[targetIdx].Apply(sim)
				}
			}
			spell.DealOutcome(sim, result)
		},
	})

	seedDmgTracker := 0.0
	trySeedPop := func(sim *core.Simulation, dmg float64) {
		seedDmgTracker += dmg
		if seedDmgTracker > 1518 {
			warlock.SeedDots[targetIdx].Deactivate(sim)
			seedExplosion.Cast(sim, target)
			seedDmgTracker = 0
		}
	}
	warlock.SeedDots[targetIdx] = core.NewDot(core.Dot{
		Spell: warlock.Seeds[targetIdx],
		Aura: target.RegisterAura(core.Aura{
			Label:    "Seed-" + strconv.Itoa(int(warlock.Index)),
			ActionID: actionID,
			OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, spellEffect *core.SpellEffect) {
				if !spellEffect.Landed() {
					return
				}
				if spell.ActionID.SpellID == actionID.SpellID {
					return // Seed can't pop seed.
				}
				trySeedPop(sim, spellEffect.Damage)
			},
			OnPeriodicDamageDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, spellEffect *core.SpellEffect) {
				trySeedPop(sim, spellEffect.Damage)
			},
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				seedDmgTracker = 0
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				seedDmgTracker = 0
			},
		}),

		NumberOfTicks: 6,
		TickLength:    time.Second * 3,

		OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
			dot.SnapshotBaseDamage = 1518/6 + 0.25*dot.Spell.SpellPower()
			dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex])
		},
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
		},
	})
}
