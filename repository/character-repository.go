package repository

import (
	"bpl/client"
	"bpl/config"
	"bpl/metrics"
	"bpl/utils"
	"bytes"
	"compress/zlib"
	"database/sql/driver"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Character struct {
	Id               string         `gorm:"not null;primaryKey"`
	UserId           *int           `gorm:"null;index"`
	EventId          int            `gorm:"not null;index"`
	Name             string         `gorm:"not null"`
	Level            int            `gorm:"not null"`
	MainSkill        string         `gorm:"not null"`
	Ascendancy       string         `gorm:"not null"`
	AscendancyPoints int            `gorm:"not null"`
	Pantheon         bool           `gorm:"not null"`
	AtlasPoints      int            `gorm:"not null"`
	OldAccountName   *string        `gorm:"null;index"`
	VoidStones       pq.StringArray `gorm:"type:text[];not null"`
}

var fullDpsSkillMultiplierRegex = regexp.MustCompile(`^\(?\s*\d+x\s+(.+?)\s*\)?$`)

func float2Int64(f float64) int64 {
	if f < 0 {
		return -float2Int64(-f) // handle negative values
	}
	if f > float64(int(^uint(0)>>1)) {
		return int64(^uint(0) >> 1) // max int value
	}
	return int64(f)
}

func float2Int32(f float64) int32 {
	if f < 0 {
		return -float2Int32(-f) // handle negative values
	}
	if f > float64(int32(^uint32(0)>>1)) {
		return int32(^uint32(0) >> 1) // max int32 value
	}
	return int32(f)
}

func float2Int8(f float64) int8 {
	if f < 0 {
		return -float2Int8(-f) // handle negative values
	}
	if f > float64(int8(^uint8(0)>>1)) {
		return int8(^uint8(0) >> 1) // max int8 value
	}
	return int8(f)
}

type PoBExport []byte

func (p *PoBExport) FromString(s string) error {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return fmt.Errorf("failed to decode base64 string: %w", err)
	}

	*p = PoBExport(decoded)
	return nil
}

func (p PoBExport) ToString() string {
	encoded := base64.StdEncoding.EncodeToString([]byte(p))
	// Convert to URL-safe format
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	return encoded
}

func (p *PoBExport) Scan(value any) error {
	if value == nil {
		*p = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan PoBExport: expected []byte, got %T", value)
	}

	*p = PoBExport(bytes)
	return nil
}

func (p PoBExport) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return []byte(p), nil
}

type CharacterPob struct {
	Id                int           `gorm:"not null;primaryKey"`
	CharacterId       string        `gorm:"not null;index"`
	Level             int           `gorm:"not null"`
	MainSkill         string        `gorm:"not null"`
	Ascendancy        string        `gorm:"not null"`
	Export            PoBExport     `gorm:"not null;type:bytea"`
	CreatedAt         time.Time     `gorm:"not null;index"`
	UpdatedAt         time.Time     `gorm:"not null"`
	Items             pq.Int32Array `gorm:"not null;type:int2[]"`
	HighIlevelFlasks  int8          `gorm:"not null"`

	DPS           int64 `gorm:"not null"`
	EHP           int32 `gorm:"not null"`
	PhysMaxHit    int32 `gorm:"not null"`
	EleMaxHit     int32 `gorm:"not null"`
	HP            int32 `gorm:"not null"`
	Mana          int32 `gorm:"not null"`
	ES            int32 `gorm:"not null"`
	Armour        int32 `gorm:"not null"`
	Evasion       int32 `gorm:"not null"`
	XP            int64 `gorm:"not null"`
	MovementSpeed int32 `gorm:"not null"`
	AttackBlock   int8  `gorm:"not null"`
	SpellBlock    int8  `gorm:"not null"`
	LowestEleRes  int8  `gorm:"not null"`
}

func (c *CharacterPob) HasEqualStats(other *CharacterPob) bool {
	if other == nil {
		return false
	}
	return c.DPS == other.DPS &&
		c.EHP == other.EHP &&
		c.PhysMaxHit == other.PhysMaxHit &&
		c.EleMaxHit == other.EleMaxHit &&
		c.HP == other.HP &&
		c.Mana == other.Mana &&
		c.ES == other.ES &&
		c.Armour == other.Armour &&
		c.Evasion == other.Evasion &&
		c.XP == other.XP &&
		c.MovementSpeed == other.MovementSpeed &&
		c.AttackBlock == other.AttackBlock &&
		c.SpellBlock == other.SpellBlock &&
		c.LowestEleRes == other.LowestEleRes &&
		c.HighIlevelFlasks == other.HighIlevelFlasks
}

func (p *PoBExport) Decode() (*client.PathOfBuilding, error) {
	z, err := zlib.NewReader(bytes.NewReader(*p))
	if err != nil {
		return nil, fmt.Errorf("zlib decompress error: %w", err)
	}
	defer utils.Closer(z)()
	var pob client.PathOfBuilding
	if err := xml.NewDecoder(z).Decode(&pob); err != nil {
		return nil, fmt.Errorf("xml decode error: %w", err)
	}
	return &pob, nil
}

func (p *CharacterPob) UpdateStats(pob *client.PathOfBuilding) {
	p.DPS = float2Int64(max(pob.Build.PlayerStats.CombinedDPS, pob.Build.PlayerStats.CullingDPS, pob.Build.PlayerStats.FullDPS, pob.Build.PlayerStats.FullDotDPS, pob.Build.PlayerStats.PoisonDPS, pob.Build.PlayerStats.ReservationDPS, pob.Build.PlayerStats.TotalDPS, pob.Build.PlayerStats.TotalDotDPS, pob.Build.PlayerStats.WithBleedDPS, pob.Build.PlayerStats.WithIgniteDPS, pob.Build.PlayerStats.WithPoisonDPS))
	p.EHP = float2Int32(pob.Build.PlayerStats.TotalEHP)
	p.PhysMaxHit = float2Int32(pob.Build.PlayerStats.PhysicalMaximumHitTaken)
	p.EleMaxHit = float2Int32(min(pob.Build.PlayerStats.FireMaximumHitTaken, pob.Build.PlayerStats.ColdMaximumHitTaken, pob.Build.PlayerStats.LightningMaximumHitTaken))
	p.HP = float2Int32(pob.Build.PlayerStats.Life)
	p.Mana = float2Int32(pob.Build.PlayerStats.Mana)
	p.ES = float2Int32(pob.Build.PlayerStats.EnergyShield)
	p.Armour = float2Int32(pob.Build.PlayerStats.Armour)
	p.Evasion = float2Int32(pob.Build.PlayerStats.Evasion)
	p.MovementSpeed = float2Int32(pob.Build.PlayerStats.EffectiveMovementSpeedMod * 100)
	p.MainSkill = pob.GetMainSkill()
	p.AttackBlock = float2Int8(pob.Build.PlayerStats.EffectiveBlockChance)
	p.SpellBlock = float2Int8(pob.Build.PlayerStats.EffectiveSpellBlockChance)
	p.LowestEleRes = float2Int8(min(pob.Build.PlayerStats.FireResist, pob.Build.PlayerStats.ColdResist, pob.Build.PlayerStats.LightningResist))
}

type CharacterRepository interface {
	GetPobByCharacterIdBeforeTimestamp(characterId string, timestamp time.Time) (*CharacterPob, error)
	GetPobs(characterId string) ([]*CharacterPob, error)
	SaveCharacters(characters []*Character) error
	SavePoB(characterPoB *CharacterPob) error
	SavePoBs(characterPobs []*CharacterPob) error
	Save(character *Character) error
	GetCharactersForEvent(eventId int) ([]*Character, error)
	GetCharactersForUser(user *User) ([]*Character, error)
	GetCharacterById(characterId string) (*Character, error)
	GetCharacterHistory(characterId string) ([]*CharacterPob, error)
	GetLatestCharacterPoB(characterId string) (*CharacterPob, error)
	GetCharacterStatsForEvent(eventId int, cutoff time.Time) (map[string]*CharacterPob, error)
	GetLatestPoBsForEvent(eventId int) ([]*CharacterPob, error)
	GetAllHighestLevelCharactersForEachEventAndUser() ([]*Character, error)
	GetPobsFromIdWithLimit(startId int, limit int) ([]*CharacterPob, error)
	GetHighestCharacterLevelForEventsForUsers(userIds []int) (map[int]map[int]int, error)
	DeletePoB(pobId int) error
	GetPoBById(pobId int) (*CharacterPob, error)
	GetPobsOrderedByCharacterAndTime(offset int, limit int) ([]*CharacterPob, error)
	DeletePoBs(ids []int) error
}

type CharacterRepositoryImpl struct {
	DB *gorm.DB
}

func NewCharacterRepository() CharacterRepository {
	return &CharacterRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *CharacterRepositoryImpl) GetPobByCharacterIdBeforeTimestamp(characterId string, timestamp time.Time) (*CharacterPob, error) {
	characterPob := &CharacterPob{}
	err := r.DB.Where("character_id = ? AND created_at < ?", characterId, timestamp).
		Order("created_at DESC").First(characterPob).Error
	if err != nil {
		return nil, err
	}
	return characterPob, nil
}

func (r *CharacterRepositoryImpl) GetPobs(characterId string) ([]*CharacterPob, error) {
	pobs := []*CharacterPob{}
	err := r.DB.Where("character_id = ?", characterId).Order("created_at ASC").Find(&pobs).Error
	if err != nil {
		return nil, fmt.Errorf("error getting PoBs for character %s: %w", characterId, err)
	}
	return pobs, nil
}

func (r *CharacterRepositoryImpl) SaveCharacters(characters []*Character) error {
	if len(characters) == 0 {
		return nil
	}
	return r.DB.CreateInBatches(characters, 500).Error
}

func (r *CharacterRepositoryImpl) SavePoB(characterPoB *CharacterPob) error {
	return r.DB.Save(&characterPoB).Error
}

func (r *CharacterRepositoryImpl) SavePoBs(characterPobs []*CharacterPob) error {
	return r.DB.Save(&characterPobs).Error
}

func (r *CharacterRepositoryImpl) Save(character *Character) error {
	if character.Id == "" || character.Name == "" {
		return fmt.Errorf("character ID and Name must be set")
	}
	return r.DB.Save(&character).Error
}

func (r *CharacterRepositoryImpl) GetCharactersForEvent(eventId int) ([]*Character, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetCharactersForEvent"))
	defer timer.ObserveDuration()
	charData := []*Character{}
	err := r.DB.Find(&charData, Character{EventId: eventId}).Error
	if err != nil {
		return nil, err
	}
	return charData, nil
}

func (r *CharacterRepositoryImpl) GetCharactersForUser(user *User) ([]*Character, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetCharactersForUser"))
	defer timer.ObserveDuration()
	charData := []*Character{}
	accountName := ""
	if user.GetAccountName(ProviderPoE) != nil {
		accountName = strings.Split(*user.GetAccountName(ProviderPoE), "#")[0]
	}
	err := r.DB.Where("user_id = ? or old_account_name = ?", user.Id, accountName).Find(&charData).Error
	if err != nil {
		return nil, err
	}
	return charData, nil
}
func (r *CharacterRepositoryImpl) GetCharacterById(characterId string) (*Character, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetCharacterById"))
	defer timer.ObserveDuration()
	character := &Character{}
	err := r.DB.Where("id = ?", characterId).First(character).Error
	if err != nil {
		return nil, err
	}
	return character, nil
}

func (r *CharacterRepositoryImpl) GetCharacterHistory(characterId string) ([]*CharacterPob, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetCharacterHistory"))
	defer timer.ObserveDuration()
	charData := []*CharacterPob{}
	err := r.DB.
		Select("created_at", "dps", "ehp", "phys_max_hit", "ele_max_hit", "hp", "mana", "es", "armour", "evasion", "xp", "level", "movement_speed", "main_skill").
		Where(CharacterPob{CharacterId: characterId}).
		Find(&charData).Error
	if err != nil {
		return nil, err
	}
	return charData, nil
}

func (r *CharacterRepositoryImpl) GetLatestCharacterPoB(characterId string) (*CharacterPob, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetLatestCharacterPoB"))
	defer timer.ObserveDuration()
	charData := &CharacterPob{}
	err := r.DB.Where("character_id = ?", characterId).Order("created_at DESC").First(charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting latest character stats for character %s: %w", characterId, err)
	}
	return charData, nil
}

func (r *CharacterRepositoryImpl) GetCharacterStatsForEvent(eventId int, cutoff time.Time) (map[string]*CharacterPob, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetCharacterStatsForEvent"))
	defer timer.ObserveDuration()
	charData := []*CharacterPob{}
	query := `SELECT DISTINCT ON (character_id) 
					p.ascendancy,
					p.character_id,
					p.created_at,
					p.dps,
					p.ehp,
					p.phys_max_hit,
					p.ele_max_hit,
					p.hp,
					p.mana,
					p.es,
					p.armour,
					p.evasion,
					p.xp,
					p.level,
					p.movement_speed,
					p.main_skill,
					p.items,
					p.high_ilevel_flasks,
					p.attack_block,
					p.spell_block,
					p.lowest_ele_res
				FROM character_pobs as p
				JOIN characters ON p.character_id = characters.id
				WHERE characters.event_id = ? AND p.created_at < ?
				ORDER BY character_id, created_at DESC
	`
	err := r.DB.Raw(query, eventId, cutoff).Scan(&charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting latest character stats for event %d: %w", eventId, err)
	}
	result := make(map[string]*CharacterPob, len(charData))
	for _, stat := range charData {
		result[stat.CharacterId] = stat
	}
	return result, nil
}

func (r *CharacterRepositoryImpl) GetLatestPoBsForEvent(eventId int) ([]*CharacterPob, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetLatestPoBsForEvent"))
	defer timer.ObserveDuration()
	charData := []*CharacterPob{}
	query := `SELECT DISTINCT ON (character_id) pobs.* from character_pobs as pobs
		JOIN characters ON pobs.character_id = characters.id
		WHERE characters.event_id = ?
		ORDER BY character_id, created_at DESC`
	err := r.DB.Raw(query, eventId).Scan(&charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting latest PoBs for event %d: %w", eventId, err)
	}
	return charData, nil
}

func (r *CharacterRepositoryImpl) GetAllHighestLevelCharactersForEachEventAndUser() ([]*Character, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetAllHighestLevelCharactersForEachEventAndUser"))
	defer timer.ObserveDuration()
	charData := []*Character{}
	query := `SELECT c.* FROM characters c
		JOIN (
			SELECT event_id, user_id, MAX(level) as max_level
			FROM characters
			WHERE user_id IS NOT NULL
			GROUP BY event_id, user_id
		) as max_chars
		ON c.event_id = max_chars.event_id AND c.user_id = max_chars.user_id AND c.level = max_chars.max_level
		ORDER BY c.event_id, c.user_id`
	err := r.DB.Raw(query).Scan(&charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting all highest level characters for each event and user: %w", err)
	}
	return charData, nil
}

func (r *CharacterRepositoryImpl) GetPobsFromIdWithLimit(startId int, limit int) ([]*CharacterPob, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetPobsFromIdWithLimit"))
	defer timer.ObserveDuration()
	charData := []*CharacterPob{}
	err := r.DB.Where("id >= ?", startId).Order("id ASC").Limit(limit).Find(&charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting PoBs starting from id %d with limit %d: %w", startId, limit, err)
	}
	return charData, nil
}

func (r *CharacterRepositoryImpl) GetHighestCharacterLevelForEventsForUsers(userIds []int) (map[int]map[int]int, error) {
	type Result struct {
		EventId int
		UserId  int
		Level   int
	}
	results := []Result{}
	err := r.DB.Raw(`
		SELECT event_id, user_id, MAX(level) as level
		FROM characters
		WHERE user_id IN ?
		GROUP BY event_id, user_id
	`, userIds).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("error getting highest character level for events for users: %w", err)
	}
	resultMap := make(map[int]map[int]int)
	for _, r := range results {
		if _, ok := resultMap[r.UserId]; !ok {
			resultMap[r.UserId] = make(map[int]int)
		}
		resultMap[r.UserId][r.EventId] = r.Level
	}
	return resultMap, nil
}

func (r *CharacterRepositoryImpl) DeletePoB(pobId int) error {
	return r.DB.Delete(&CharacterPob{}, pobId).Error
}

func (r *CharacterRepositoryImpl) GetPobsOrderedByCharacterAndTime(offset int, limit int) ([]*CharacterPob, error) {
	charData := []*CharacterPob{}
	err := r.DB.Order("character_id ASC, created_at ASC").Offset(offset).Limit(limit).Find(&charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting PoBs ordered by character and time (offset %d, limit %d): %w", offset, limit, err)
	}
	return charData, nil
}

func (r *CharacterRepositoryImpl) DeletePoBs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.DB.Delete(&CharacterPob{}, ids).Error
}

func (r *CharacterRepositoryImpl) GetPoBById(pobId int) (*CharacterPob, error) {
	charData := &CharacterPob{}
	err := r.DB.Where("id = ?", pobId).First(charData).Error
	if err != nil {
		return nil, fmt.Errorf("error getting PoB with id %d: %w", pobId, err)
	}
	return charData, nil
}
