package repository

import (
	"bpl/config"
	"bpl/utils"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ObjectiveType string

const (
	ObjectiveTypeItem       ObjectiveType = "ITEM"
	ObjectiveTypePlayer     ObjectiveType = "PLAYER"
	ObjectiveTypeTeam       ObjectiveType = "TEAM"
	ObjectiveTypeSubmission ObjectiveType = "SUBMISSION"
	ObjectiveTypeCategory   ObjectiveType = "CATEGORY"
)

type CountingMethod string

const (
	CountingMethodLatestValue          CountingMethod = "LATEST_VALUE"
	CountingMethodFirstCompletion      CountingMethod = "FIRST_COMPLETION"
	CountingMethodFirstFreshCompletion CountingMethod = "FIRST_FRESH_COMPLETION"
	CountingMethodHighestValue         CountingMethod = "HIGHEST_VALUE"
	CountingMethodLowestValue          CountingMethod = "LOWEST_VALUE"
	CountingMethodValueChangeInWindow  CountingMethod = "VALUE_CHANGE_IN_WINDOW"
	CountingMethodChildResult          CountingMethod = "CHILD_RESULT"
)

type TrackedValue string

const (
	TrackedValueStackSize TrackedValue = "STACK_SIZE"

	TrackedValueFossilFuelHigh TrackedValue = "FOSSIL_FUEL_HIGH"
	TrackedValueFossilFuelMid  TrackedValue = "FOSSIL_FUEL_MID"

	TrackedValueCharacterLevel                  TrackedValue = "CHARACTER_LEVEL"
	TrackedValueDelveDepth                      TrackedValue = "DELVE_DEPTH"
	TrackedValueDelveDepthAfter100              TrackedValue = "DELVE_DEPTH_AFTER_100"
	TrackedValueWeightedDelveDepth              TrackedValue = "WEIGHTED_DELVE_DEPTH"
	TrackedValueTeamPlayersWithPantheonUnlocked TrackedValue = "TEAM_PLAYERS_WITH_PANTHEON_UNLOCKED"
	TrackedValueAscendancyPoints                TrackedValue = "ASCENDANCY_POINTS"
	TrackedValueTeamPlayersWithAllLabsCompleted TrackedValue = "TEAM_PLAYERS_WITH_ALL_LABS_COMPLETED"
	TrackedValueBloodlineAscendancyPoints       TrackedValue = "BLOODLINE_ASCENDANCY_POINTS"
	TrackedValueBloodlineAscendancyUnlocked     TrackedValue = "BLOODLINE_ASCENDANCY_UNLOCKED"
	TrackedValuePersonalObjectiveScore          TrackedValue = "PERSONAL_OBJECTIVE_SCORE"
	TrackedValueHasRareAscendancyPast90         TrackedValue = "HAS_RARE_ASCENDANCY_PAST_90"
	TrackedValueVoidStoneCount                  TrackedValue = "VOID_STONE_COUNT"

	TrackedValueWeaponQuality TrackedValue = "WEAPON_QUALITY"
	TrackedValueArmourQuality TrackedValue = "ARMOUR_QUALITY"
	TrackedValueFlaskQuality  TrackedValue = "FLASK_QUALITY"

	TrackedValueEvasion                   TrackedValue = "EVASION"
	TrackedValueEnergyShield              TrackedValue = "ENERGY_SHIELD"
	TrackedValueArmour                    TrackedValue = "ARMOUR"
	TrackedValueHP                        TrackedValue = "HP"
	TrackedValueMana                      TrackedValue = "MANA"
	TrackedValueFullDPS                   TrackedValue = "FULL_DPS"
	TrackedValueEHP                       TrackedValue = "EHP"
	TrackedValueMovementSpeedBonus        TrackedValue = "MOVEMENT_SPEED_BONUS"
	TrackedValuePhysicalMaxHit            TrackedValue = "PHYSICAL_MAX_HIT"
	TrackedValueElementalMaxHit           TrackedValue = "ELEMENTAL_MAX_HIT"
	TrackedValueAttackBlockChance         TrackedValue = "ATTACK_BLOCK_CHANCE"
	TrackedValueSpellBlockChance          TrackedValue = "SPELL_BLOCK_CHANCE"
	TrackedValueHighItemLevelFlaskCount   TrackedValue = "HIGH_ITEM_LEVEL_FLASK_COUNT"
	TrackedValueLowestElementalResistance TrackedValue = "LOWEST_ELEMENTAL_RESISTANCE"

	TrackedValueAtlasPoints TrackedValue = "ATLAS_POINTS"

	TrackedValueInfluencedItemCount      TrackedValue = "INFLUENCED_ITEM_COUNT"
	TrackedValueFoulbornItemCount        TrackedValue = "FOULBORN_ITEM_COUNT"
	TrackedValueSocketedGemCount         TrackedValue = "SOCKETED_GEM_COUNT"
	TrackedValueCorruptedItemCount       TrackedValue = "CORRUPTED_ITEM_COUNT"
	TrackedValueJewelsWithImplicitsCount TrackedValue = "JEWELS_WITH_IMPLICITS_COUNT"
	TrackedValueEnchantedItemCount       TrackedValue = "ENCHANTED_ITEM_COUNT"

	TrackedValueSubmittedValue TrackedValue = "SUBMITTED_VALUE"

	TrackedValueCompletedChildObjectiveCount TrackedValue = "COMPLETED_CHILD_OBJECTIVE_COUNT"
	TrackedValueChildNumerValueSum           TrackedValue = "CHILD_NUMER_VALUE_SUM"
)

var playerObjectiveTrackedValues = []TrackedValue{
	TrackedValueCharacterLevel,
	TrackedValueDelveDepth,
	TrackedValueDelveDepthAfter100,
	TrackedValueWeightedDelveDepth,
	TrackedValueTeamPlayersWithPantheonUnlocked,
	TrackedValueAscendancyPoints,
	TrackedValueBloodlineAscendancyUnlocked,
	TrackedValueBloodlineAscendancyPoints,
	TrackedValueTeamPlayersWithAllLabsCompleted,
	TrackedValuePersonalObjectiveScore,
	TrackedValueWeaponQuality,
	TrackedValueArmourQuality,
	TrackedValueFlaskQuality,
	TrackedValueEvasion,
	TrackedValueEnergyShield,
	TrackedValueArmour,
	TrackedValueHP,
	TrackedValueMana,
	TrackedValueFullDPS,
	TrackedValueEHP,
	TrackedValueMovementSpeedBonus,
	TrackedValuePhysicalMaxHit,
	TrackedValueElementalMaxHit,
	TrackedValueAtlasPoints,
	TrackedValueInfluencedItemCount,
	TrackedValueFoulbornItemCount,
	TrackedValueSocketedGemCount,
	TrackedValueCorruptedItemCount,
	TrackedValueJewelsWithImplicitsCount,
	TrackedValueHasRareAscendancyPast90,
	TrackedValueEnchantedItemCount,
}

var ObjectiveTypeToTrackedValues = map[ObjectiveType][]TrackedValue{
	ObjectiveTypeItem:       {TrackedValueStackSize, TrackedValueFossilFuelHigh, TrackedValueFossilFuelMid},
	ObjectiveTypePlayer:     playerObjectiveTrackedValues,
	ObjectiveTypeTeam:       playerObjectiveTrackedValues,
	ObjectiveTypeSubmission: {TrackedValueSubmittedValue},
	ObjectiveTypeCategory:   {TrackedValueCompletedChildObjectiveCount, TrackedValueChildNumerValueSum},
}

type SyncStatus string

const (
	SyncStatusSynced   SyncStatus = "SYNCED"
	SyncStatusSyncing  SyncStatus = "SYNCING"
	SyncStatusDesynced SyncStatus = "DESYNCED"
)

type ObjectiveScoringRule struct {
	ObjectiveId   int `gorm:"primaryKey"`
	ScoringRuleId int `gorm:"primaryKey"`
}

// ObjectiveDetails holds optional, per-objective extras that only apply to a
// small subset of objectives (e.g. gem/ascendancy restrictions, an
// explanation of how the tracked value is computed). New fields can be
// added here without a schema migration, since the whole struct is stored
// as one jsonb column.
type ObjectiveDetails struct {
	GemsLimited             bool    `json:"gems_limited,omitempty"`
	AscendanciesLimited     bool    `json:"ascendancies_limited,omitempty"`
	TrackedValueExplanation *string `json:"tracked_value_explanation,omitempty"`
}

func (d *ObjectiveDetails) Scan(value any) error {
	if value == nil {
		*d = ObjectiveDetails{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: not a byte slice")
	}
	return json.Unmarshal(bytes, d)
}

func (d ObjectiveDetails) Value() (driver.Value, error) {
	return json.Marshal(d)
}

type Objective struct {
	Id             int              `gorm:"primaryKey"`
	Name           string           `gorm:"not null"`
	Extra          string           `gorm:"null"`
	RequiredAmount int              `gorm:"not null"`
	Conditions     Conditions       `gorm:"type:jsonb"`
	ParentId       *int             `gorm:"null"`
	EventId        int              `gorm:"not null;references:events(id)"`
	ObjectiveType  ObjectiveType    `gorm:"not null"`
	TrackedValue   TrackedValue     `gorm:"not null"`
	CountingMethod CountingMethod   `gorm:"not null"`
	ValidFrom      *time.Time       `gorm:"null"`
	ValidTo        *time.Time       `gorm:"null"`
	ScoringRules   []*ScoringRule   `gorm:"many2many:objective_scoring_rules;joinForeignKey:objective_id;joinReferences:scoring_rule_id"`
	HideProgress   bool             `gorm:"not null;default:false"`
	SyncStatus     SyncStatus       `gorm:"not null;default:DESYNCED"`
	Children       []*Objective     `gorm:"foreignKey:ParentId;constraint:OnDelete:CASCADE"`
	Details        ObjectiveDetails `gorm:"type:jsonb;not null;default:'{}'"`
}

func (o *Objective) FlatMap() []*Objective {
	result := []*Objective{o}
	for _, child := range o.Children {
		result = append(result, child.FlatMap()...)
	}
	return result
}

type ObjectiveRepository interface {
	SaveObjective(objective *Objective) (*Objective, error)
	SaveObjectives(objectives []*Objective) ([]*Objective, error)
	GetObjectiveById(objectiveId int, preloads ...string) (*Objective, error)
	DeleteObjective(objectiveId int) error
	DeleteObjectivesByEventId(eventId int) error
	RemoveScoringRule(scoringId int) error
	AssociateScoringRules(objectiveId int, presetIds []int) error
	StartSync(objectiveIds []int) error
	FinishSync(objectiveIds []int) error
	GetObjectivesByEventId(eventId int, preloads ...string) (*Objective, error)
	GetObjectivesByEventIdFlat(eventId int, preloads ...string) ([]*Objective, error)
	GetAllObjectives(preloads ...string) ([]*Objective, error)
}

type ObjectiveRepositoryImpl struct {
	DB *gorm.DB
}

func NewObjectiveRepository() ObjectiveRepository {
	return &ObjectiveRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *ObjectiveRepositoryImpl) SaveObjective(objective *Objective) (*Objective, error) {
	objective.SyncStatus = SyncStatusDesynced
	result := r.DB.Save(objective)
	if result.Error != nil {
		return nil, result.Error
	}
	return objective, nil
}

func (r *ObjectiveRepositoryImpl) SaveObjectives(objectives []*Objective) ([]*Objective, error) {
	result := r.DB.Save(objectives)
	return objectives, result.Error
}

func (r *ObjectiveRepositoryImpl) GetObjectiveById(objectiveId int, preloads ...string) (*Objective, error) {
	var objective Objective
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.First(&objective, "id = ?", objectiveId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &objective, nil
}

func (r *ObjectiveRepositoryImpl) DeleteObjective(objectiveId int) error {
	result := r.DB.Delete(&Objective{Id: objectiveId})
	return result.Error
}

func (r *ObjectiveRepositoryImpl) DeleteObjectivesByEventId(eventId int) error {
	result := r.DB.Where("event_id = ?", eventId).Delete(&Objective{})
	return result.Error
}

func (r *ObjectiveRepositoryImpl) RemoveScoringRule(scoringId int) error {
	return r.DB.Where("scoring_rule_id = ?", scoringId).Delete(&ObjectiveScoringRule{}).Error
}

func (r *ObjectiveRepositoryImpl) AssociateScoringRules(objectiveId int, ruleIds []int) error {
	if len(ruleIds) == 0 {
		return nil
	}
	err := r.DB.Where("objective_id = ?", objectiveId).Delete(&ObjectiveScoringRule{}).Error
	if err != nil {
		return err
	}
	if len(ruleIds) == 0 {
		return nil
	}
	associations := make([]ObjectiveScoringRule, len(ruleIds))
	for i, ruleId := range ruleIds {
		associations[i] = ObjectiveScoringRule{
			ObjectiveId:   objectiveId,
			ScoringRuleId: ruleId,
		}
	}

	return r.DB.Create(&associations).Error
}

func (r *ObjectiveRepositoryImpl) StartSync(objectiveIds []int) error {
	result := r.DB.
		Model(&Objective{}).Where("id IN ? and sync_status = ?", objectiveIds, SyncStatusDesynced).
		Update("sync_status", SyncStatusSyncing)
	return result.Error
}

func (r *ObjectiveRepositoryImpl) FinishSync(objectiveIds []int) error {
	if len(objectiveIds) == 0 {
		return nil
	}
	result := r.DB.
		Model(&Objective{}).Where("id IN ? and sync_status = ?", objectiveIds, SyncStatusSyncing).
		Update("sync_status", SyncStatusSynced)
	return result.Error
}

func (r *ObjectiveRepositoryImpl) GetObjectivesByEventId(eventId int, preloads ...string) (*Objective, error) {
	objectives, err := r.GetObjectivesByEventIdFlat(eventId, preloads...)
	if err != nil {
		return nil, fmt.Errorf("failed to get objectives for event %d: %w", eventId, err)
	}
	idMap := make(map[int]*Objective)
	for _, objective := range objectives {
		idMap[objective.Id] = objective
	}
	for _, objective := range objectives {
		if objective.ParentId != nil {
			parent, exists := idMap[*objective.ParentId]
			if exists {
				parent.Children = append(parent.Children, objective)
			}
		}
	}
	rootObjective, found := utils.FindFirst(objectives, func(o *Objective) bool {
		return o.ParentId == nil
	})
	if !found {
		return nil, fmt.Errorf("no root objective found for event %d", eventId)
	}
	return rootObjective, nil
}

func (r *ObjectiveRepositoryImpl) GetObjectivesByEventIdFlat(eventId int, preloads ...string) ([]*Objective, error) {
	var objectives []*Objective
	query := r.DB.Model(&Objective{}).Where("event_id = ?", eventId)
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.Find(&objectives)
	if result.Error != nil {
		return nil, result.Error
	}
	return objectives, nil
}

func (r *ObjectiveRepositoryImpl) GetAllObjectives(preloads ...string) ([]*Objective, error) {
	var objectives []*Objective
	query := r.DB.Model(&Objective{})
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.Find(&objectives)
	if result.Error != nil {
		return nil, result.Error
	}
	return objectives, nil
}
