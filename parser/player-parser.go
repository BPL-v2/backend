package parser

import (
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Player struct {
	AtlasPassiveTrees []client.AtlasPassiveTree
	DelveDepth        int
	Character         *client.GGGCharacter
	PoB               *repository.CharacterPob
	VoidStones        utils.Set[string]
}

type PlayerUpdate struct {
	UserId           int
	AccountName      string
	TeamId           int
	Token            string
	TokenExpiry      time.Time
	Mu               sync.Mutex
	SuccessiveErrors int
	LastActive       time.Time

	New Player
	Old Player

	LastUpdateTimes struct {
		CharacterName time.Time
		Character     time.Time
		LeagueAccount time.Time
		PoB           time.Time
	}
}

func (p *Player) MaxAtlasTreeNodes() int {
	return utils.Max(utils.Map(p.AtlasPassiveTrees, func(tree client.AtlasPassiveTree) int {
		return len(tree.Hashes)
	})...)
}

func (p *PlayerUpdate) CanMakeRequests() bool {
	return p.TokenExpiry.After(time.Now()) && p.Token != "" && p.SuccessiveErrors < 5
}

func (p *PlayerUpdate) ShouldUpdateCharacterName(timings map[repository.TimingKey]time.Duration) bool {
	if !p.CanMakeRequests() {
		return false
	}
	return time.Since(p.LastUpdateTimes.CharacterName) > timings[repository.CharacterNameRefetchDelay]
}

func (p *PlayerUpdate) ShouldUpdateCharacter(timings map[repository.TimingKey]time.Duration) bool {
	if !p.CanMakeRequests() {
		return false
	}
	if p.New.Character.Name == "" {
		return false
	}
	if p.LastActive.Before(time.Now().Add(-timings[repository.InactivityDuration])) {
		return time.Since(p.LastUpdateTimes.Character) > timings[repository.CharacterRefetchDelayInactive]
	}
	if p.New.Character.Level > 40 && !p.New.Character.HasPantheon() {
		return time.Since(p.LastUpdateTimes.Character) > timings[repository.CharacterRefetchDelayImportant]
	}
	if p.New.Character.Level > 68 && p.New.Character.GetAscendancyPoints() < 8 {
		return time.Since(p.LastUpdateTimes.Character) > timings[repository.CharacterRefetchDelayImportant]
	}
	return time.Since(p.LastUpdateTimes.Character) > timings[repository.CharacterRefetchDelay]
}

func (p *PlayerUpdate) ShouldUpdateLeagueAccount(timings map[repository.TimingKey]time.Duration) bool {
	if !p.CanMakeRequests() {
		return false
	}
	if p.New.Character.Level < 55 {
		return false
	}
	if p.LastActive.Before(time.Now().Add(-timings[repository.InactivityDuration])) {
		return time.Since(p.LastUpdateTimes.LeagueAccount) > timings[repository.LeagueAccountRefetchDelayInactive]
	}

	if p.New.MaxAtlasTreeNodes() < 100 {
		return time.Since(p.LastUpdateTimes.LeagueAccount) > timings[repository.LeagueAccountRefetchDelayImportant]
	}

	return time.Since(p.LastUpdateTimes.LeagueAccount) > timings[repository.LeagueAccountRefetchDelay]
}

type TeamObjectiveChecker func(p []*Player) int

type PlayerObjectiveChecker func(p *Player) int

var rareAscendancies = []string{
	"Assassin",
	"Juggernaut",
	"Gladiator",
	"Trickster",
	"Guardian",
	"Champion",
	"Occultist",
	"Warden",
	"Inquisitor",
	"Saboteur",
	"Ascendant",
}

func progressiveDelveMultiplier(depth int) float64 {
	if depth < 0 {
		return 0
	}
	if depth < 50 {
		return 1
	}
	if depth < 150 {
		return 1.2
	}
	if depth < 250 {
		return 1.4
	}
	if depth < 350 {
		return 1.6
	}
	if depth < 450 {
		return 1.8
	}
	return 2
}

func progressiveDelveDepth(p *Player) int {
	return int(float64(p.DelveDepth-100) * progressiveDelveMultiplier(p.DelveDepth-100))
}

func calculateOldPO(p *Player) int {
	score := 0
	if p.Character == nil {
		return 0
	}
	ascendancyPoints := p.Character.GetAscendancyPoints()
	if p.Character.Level >= 40 {
		score += 1
	}
	if p.Character.Level >= 60 {
		score += 1
	}
	if p.Character.Level >= 80 {
		score += 1
	}
	if p.Character.Level >= 90 {
		score += 3
	}
	if ascendancyPoints >= 4 {
		score += 1
	}
	if ascendancyPoints >= 6 {
		score += 1
	}
	if ascendancyPoints >= 8 {
		score += 1
	}
	if p.MaxAtlasTreeNodes() >= 40 {
		score += 3
	}
	if score > 9 {
		return 9
	}
	return score
}

func poChecker() (PlayerObjectiveChecker, error) {
	levelFunc, err := parserForTrackedValue(repository.TrackedValueCharacterLevel)
	ascendancyFunc, err := parserForTrackedValue(repository.TrackedValueAscendancyPoints)

	armourFunc, err := parserForTrackedValue(repository.TrackedValueArmour)
	evasionFunc, err := parserForTrackedValue(repository.TrackedValueEvasion)
	movementSpeedFunc, err := parserForTrackedValue(repository.TrackedValueMovementSpeedBonus)
	esFunc, err := parserForTrackedValue(repository.TrackedValueEnergyShield)
	hpFunc, err := parserForTrackedValue(repository.TrackedValueHP)
	manaFunc, err := parserForTrackedValue(repository.TrackedValueMana)
	dpsFunc, err := parserForTrackedValue(repository.TrackedValueFullDPS)
	ehpFunc, err := parserForTrackedValue(repository.TrackedValueEHP)
	blockFunc, err := parserForTrackedValue(repository.TrackedValueAttackBlockChance)
	maxResFunc, err := parserForTrackedValue(repository.TrackedValueLowestElementalResistance)
	eleMaxHitFunc, err := parserForTrackedValue(repository.TrackedValueElementalMaxHit)
	physMaxHitFunc, err := parserForTrackedValue(repository.TrackedValuePhysicalMaxHit)
	voidStonesFunc, err := parserForTrackedValue(repository.TrackedValueVoidStoneCount)
	if err != nil {
		return nil, err
	}
	return func(p *Player) int {
		level := levelFunc(p)
		ascendancy := ascendancyFunc(p)
		armour := armourFunc(p)
		evasion := evasionFunc(p)
		movementSpeed := movementSpeedFunc(p)
		es := esFunc(p)
		hp := hpFunc(p)
		mana := manaFunc(p)
		dps := dpsFunc(p)
		ehp := ehpFunc(p)
		block := blockFunc(p)
		maxRes := maxResFunc(p)
		eleMaxHit := eleMaxHitFunc(p)
		physMaxHit := physMaxHitFunc(p)
		voidStones := voidStonesFunc(p)
		score := 0
		// general progression
		if level >= 40 {
			score += 1
		}
		if level >= 60 {
			score += 1
		}
		if level >= 80 {
			score += 1
		}
		if level >= 90 {
			score += 3
		}
		if ascendancy >= 4 {
			score += 1
		}
		if ascendancy >= 6 {
			score += 1
		}
		if ascendancy >= 8 {
			score += 1
		}
		score = min(score, 9)

		// custom progression
		if armour >= 15_000 {
			score += 1
		}
		if armour >= 30_000 {
			score += 1
		}
		if armour >= 100_000 {
			score += 2
		}
		if evasion >= 15_000 {
			score += 1
		}
		if evasion >= 30_000 {
			score += 1
		}
		if evasion >= 100_000 {
			score += 2
		}
		if level >= 95 {
			score += 2
		}
		if level >= 98 {
			score += 2
		}
		if voidStones >= 4 {
			score += 4
		}
		if movementSpeed >= 50 {
			score += 1
		}
		if movementSpeed >= 100 {
			score += 1
		}
		if movementSpeed >= 150 {
			score += 2
		}
		if es >= 4000 {
			score += 1
		}
		if es >= 8000 {
			score += 1
		}
		if es >= 15_000 {
			score += 2
		}
		if hp >= 4000 {
			score += 1
		}
		if hp >= 5000 {
			score += 1
		}
		if hp >= 7000 {
			score += 2
		}
		if mana >= 2000 {
			score += 1
		}
		if mana >= 4000 {
			score += 1
		}
		if mana >= 8000 {
			score += 2
		}
		if dps >= 2_000_000 {
			score += 1
		}
		if dps >= 5_000_000 {
			score += 1
		}
		if dps >= 50_000_000 {
			score += 2
		}
		if ehp >= 50_000 {
			score += 1
		}
		if ehp >= 150_000 {
			score += 1
		}
		if ehp >= 400_000 {
			score += 2
		}
		if block >= 60 {
			score += 1
		}
		if block >= 70 {
			score += 1
		}
		if block >= 80 {
			score += 2
		}
		if maxRes >= 80 {
			score += 1
		}
		if maxRes >= 85 {
			score += 1
		}
		if maxRes >= 90 {
			score += 2
		}
		if eleMaxHit >= 40_000 {
			score += 1
		}
		if eleMaxHit >= 80_000 {
			score += 1
		}
		if eleMaxHit >= 120_000 {
			score += 2
		}
		if physMaxHit >= 12_000 {
			score += 1
		}
		if physMaxHit >= 16_000 {
			score += 1
		}
		if physMaxHit >= 20_000 {
			score += 2
		}
		return min(score, 17)
	}, nil
}

func parserForTrackedValue(numberField repository.TrackedValue) (PlayerObjectiveChecker, error) {
	switch numberField {
	case repository.TrackedValueCharacterLevel:
		return func(p *Player) int {
			if p.Character == nil {
				return 0
			}
			return p.Character.Level
		}, nil
	case repository.TrackedValueDelveDepth:
		return func(p *Player) int {
			return p.DelveDepth
		}, nil
	case repository.TrackedValueDelveDepthAfter100:
		return func(p *Player) int {
			return max(p.DelveDepth-100, 0)
		}, nil
	case repository.TrackedValueWeightedDelveDepth:
		return progressiveDelveDepth, nil
	case repository.TrackedValueTeamPlayersWithPantheonUnlocked:
		return func(p *Player) int {
			count := 0
			if p.Character == nil {
				return count
			}
			if p.Character.Passives.PantheonMajor != nil {
				count++
			}
			if p.Character.Passives.PantheonMinor != nil {
				count++
			}
			return count
		}, nil
	case repository.TrackedValueAscendancyPoints:
		return func(p *Player) int {
			if p.Character == nil {
				return 0
			}
			return p.Character.GetAscendancyPoints()
		}, nil
	case repository.TrackedValueTeamPlayersWithAllLabsCompleted:
		return func(p *Player) int {
			if p.Character == nil || p.Character.GetAscendancyPoints() < 8 {
				return 0
			}
			return 1
		}, nil
	case repository.TrackedValuePersonalObjectiveScore:
		return poChecker()
	case repository.TrackedValueInfluencedItemCount:
		return func(p *Player) int {
			return itemCount(p.Character, func(item client.Item) bool {
				return item.Influences != nil && len(*item.Influences) > 0
			})
		}, nil
	case repository.TrackedValueFoulbornItemCount:
		return func(p *Player) int {
			return itemCount(p.Character, func(item client.Item) bool {
				return item.Mutated != nil
			})
		}, nil
	case repository.TrackedValueSocketedGemCount:
		return func(p *Player) int {
			if p.Character == nil || p.Character.Equipment == nil {
				return 0
			}
			count := 0
			for _, item := range *p.Character.Equipment {
				if item.SocketedItems != nil {
					for _, socketed := range *item.SocketedItems {
						if socketed.AbyssJewel == nil {
							count++
						}
					}
				}
			}
			return count
		}, nil
	case repository.TrackedValueCorruptedItemCount:
		return func(p *Player) int {
			return itemCount(p.Character, func(item client.Item) bool {
				return item.Corrupted != nil
			})
		}, nil
	case repository.TrackedValueJewelsWithImplicitsCount:
		return func(p *Player) int {
			return itemCount(p.Character, func(item client.Item) bool {
				return strings.HasSuffix(item.BaseType, "Jewel") && item.ImplicitMods != nil && len(*item.ImplicitMods) > 0
			})
		}, nil
	case repository.TrackedValueAtlasPoints:
		return func(p *Player) int {
			total := 0
			for _, tree := range p.AtlasPassiveTrees {
				points := len(tree.Hashes)
				if slices.Contains(tree.Hashes, 65225) {
					points -= 20
				}
				total = max(total, points)
			}
			return total
		}, nil
	case repository.TrackedValueArmourQuality:
		return func(p *Player) int {
			return quality(p.Character, "Armour")
		}, nil
	case repository.TrackedValueWeaponQuality:
		return func(p *Player) int {
			return quality(p.Character, "Weapon")
		}, nil
	case repository.TrackedValueFlaskQuality:
		return func(p *Player) int {
			return quality(p.Character, "Flask")
		}, nil
	case repository.TrackedValueEnchantedItemCount:
		return func(p *Player) int {
			sum := 0
			if p.Character == nil {
				return sum
			}
			for _, item := range p.Character.GetAllItems() {
				if item.EnchantMods != nil && len(*item.EnchantMods) > 0 {
					sum++
				}
			}
			return sum
		}, nil
	case repository.TrackedValueEvasion:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.Evasion)
		}, nil
	case repository.TrackedValueArmour:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.Armour)
		}, nil
	case repository.TrackedValueEnergyShield:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.ES)
		}, nil
	case repository.TrackedValueMana:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.Mana)
		}, nil
	case repository.TrackedValueHP:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.HP)
		}, nil

	case repository.TrackedValueEHP:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.EHP)
		}, nil
	case repository.TrackedValuePhysicalMaxHit:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.PhysMaxHit)
		}, nil
	case repository.TrackedValueElementalMaxHit:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.EleMaxHit)
		}, nil
	case repository.TrackedValueAttackBlockChance:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.AttackBlock)
		}, nil
	case repository.TrackedValueSpellBlockChance:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.SpellBlock)
		}, nil
	case repository.TrackedValueLowestElementalResistance:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.LowestEleRes)
		}, nil
	case repository.TrackedValueVoidStoneCount:
		return func(p *Player) int {
			return len(p.VoidStones)
		}, nil
	case repository.TrackedValueHighItemLevelFlaskCount:
		return func(p *Player) int {
			return itemCount(p.Character, func(item client.Item) bool {
				return strings.Contains(item.BaseType, "Flask") && item.Ilvl >= 84
			})
		}, nil
	case repository.TrackedValueMovementSpeedBonus:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}

			return int(p.PoB.MovementSpeed) - 100
		}, nil
	case repository.TrackedValueFullDPS:
		return func(p *Player) int {
			if p.PoB == nil {
				return 0
			}
			return int(p.PoB.DPS)
		}, nil
	case repository.TrackedValueHasRareAscendancyPast90:
		return func(p *Player) int {
			if p.Character == nil || p.Character.Level < 90 || !slices.Contains(rareAscendancies, p.Character.Class) {
				return 0
			}
			return 1
		}, nil
	case repository.TrackedValueBloodlineAscendancyPoints:
		return func(p *Player) int {
			if p.Character == nil {
				return 0
			}
			return p.Character.GetBloodlinePoints()
		}, nil
	case repository.TrackedValueBloodlineAscendancyUnlocked:
		return func(p *Player) int {
			if p.Character == nil || p.Character.Passives.AlternateAscendancy == nil {
				return 0
			}
			return 1
		}, nil
	default:
		return nil, fmt.Errorf("unsupported number field")
	}
}

func GetPlayerChecker(objective *repository.Objective) (PlayerObjectiveChecker, error) {
	if (objective.ObjectiveType != repository.ObjectiveTypePlayer) && (objective.ObjectiveType != repository.ObjectiveTypeTeam) {
		return nil, fmt.Errorf("not a player objective")
	}
	return parserForTrackedValue(objective.TrackedValue)
}

func quality(character *client.GGGCharacter, superclass string) int {
	if character == nil || character.Equipment == nil {
		return 0
	}
	totalQuality := 0
	for _, item := range *character.Equipment {
		if SuperClasses[ItemClasses[item.BaseType]] != superclass || item.Properties == nil {
			continue
		}
		for _, property := range *item.Properties {
			if strings.Contains(property.Name, "Quality") {
				quality, err := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(property.Values[0].Name(), "%", ""), "+", ""))
				if err != nil {
					continue
				}
				totalQuality += quality
			}
		}
	}
	return totalQuality
}

func itemCount(character *client.GGGCharacter, predicate func(item client.Item) bool) int {
	if character == nil {
		return 0
	}
	count := 0
	if character.Equipment != nil {
		for _, item := range *character.Equipment {
			if predicate(item) {
				count++
			}
		}
	}
	if character.Jewels != nil {
		for _, item := range *character.Jewels {
			if predicate(item) {
				count++
			}
		}
	}
	return count
}

func GetTeamChecker(objective *repository.Objective) (TeamObjectiveChecker, error) {
	if objective.ObjectiveType != repository.ObjectiveTypeTeam {
		return nil, fmt.Errorf("not a team objective")
	}
	checker, err := GetPlayerChecker(objective)
	if err != nil {
		return nil, err
	}
	return func(p []*Player) int {
		sum := 0
		for _, player := range p {
			c := checker(player)
			sum += c
		}
		return sum
	}, nil
}

type PlayerChecker map[int]PlayerObjectiveChecker
type TeamChecker map[int]TeamObjectiveChecker

func NewPlayerChecker(objectives []*repository.Objective) (*PlayerChecker, error) {
	checkers := make(map[int]PlayerObjectiveChecker)
	for _, objective := range objectives {
		if objective.ObjectiveType != repository.ObjectiveTypePlayer {
			continue
		}
		checker, err := GetPlayerChecker(objective)
		if err != nil {
			return nil, err
		}
		checkers[objective.Id] = checker
	}
	return (*PlayerChecker)(&checkers), nil
}

func NewTeamChecker(objectives []*repository.Objective) (*TeamChecker, error) {
	checkers := make(map[int]TeamObjectiveChecker)
	for _, objective := range objectives {
		if objective.ObjectiveType != repository.ObjectiveTypeTeam {
			continue
		}
		checker, err := GetTeamChecker(objective)
		if err != nil {
			return nil, err
		}
		checkers[objective.Id] = checker
	}
	return (*TeamChecker)(&checkers), nil
}

func (pc *PlayerChecker) CheckForCompletions(update *PlayerUpdate) []*CheckResult {
	results := make([]*CheckResult, 0)
	for id, checker := range *pc {
		new := checker(&update.New)
		if new != checker(&update.Old) {
			results = append(results, &CheckResult{
				ObjectiveId: id,
				Number:      new,
			})
		}
	}
	return results
}

func (tc *TeamChecker) CheckForCompletions(updates []*PlayerUpdate) []*CheckResult {
	results := make([]*CheckResult, 0)
	oldTeam := make([]*Player, 0)
	newTeam := make([]*Player, 0)
	for _, update := range updates {
		oldTeam = append(oldTeam, &update.Old)
		newTeam = append(newTeam, &update.New)
	}
	for id, checker := range *tc {
		new := checker(newTeam)
		if new != checker(oldTeam) {
			results = append(results, &CheckResult{
				ObjectiveId: id,
				Number:      new,
			})
		}
	}
	return results
}
