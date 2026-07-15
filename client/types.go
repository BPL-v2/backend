package client

import (
	"bpl/utils"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Realm string

const (
	PC   Realm = "pc"
	Sony Realm = "sony"
	Xbox Realm = "xbox"
	PoE2 Realm = "poe2"
)

type LeagueRule struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type LeagueCategory struct {
	Id      string `json:"id"`
	Current *bool  `json:"current,omitempty"`
}

type League struct {
	Id            string          `json:"id"`
	Realm         *Realm          `json:"realm,omitempty"`
	Description   *string         `json:"description,omitempty"`
	Category      *LeagueCategory `json:"category,omitempty"`
	Rules         *[]LeagueRule   `json:"rules,omitempty"`
	RegisterAt    *string         `json:"registerAt,omitempty"`
	Event         *bool           `json:"event,omitempty"`
	URL           *string         `json:"url,omitempty"`
	StartAt       *string         `json:"startAt,omitempty"`
	EndAt         *string         `json:"endAt,omitempty"`
	TimedEvent    *bool           `json:"timedEvent,omitempty"`
	ScoreEvent    *bool           `json:"scoreEvent,omitempty"`
	DelveEvent    *bool           `json:"delveEvent,omitempty"`
	AncestorEvent *bool           `json:"ancestorEvent,omitempty"`
	LeagueEvent   *bool           `json:"leagueEvent,omitempty"`
}

type LadderEntryCharacterDepth struct {
	Default *int `json:"default,omitempty"`
	Solo    *int `json:"solo,omitempty"`
}

type LadderEntryCharacter struct {
	Id         string                     `json:"id"`
	Name       string                     `json:"name"`
	Level      int                        `json:"level"`
	Class      string                     `json:"class"`
	Time       *int                       `json:"time,omitempty"`
	Score      *int                       `json:"score,omitempty"`
	Progress   *map[string]any            `json:"progress,omitempty"`
	Experience *int                       `json:"experience,omitempty"`
	Depth      *LadderEntryCharacterDepth `json:"depth,omitempty"`
}

type LadderEntry struct {
	Rank       int                  `json:"rank"`
	Dead       *bool                `json:"dead,omitempty"`
	Retired    *bool                `json:"retired,omitempty"`
	Ineligible *bool                `json:"ineligible,omitempty"`
	Public     *bool                `json:"public,omitempty"`
	Character  LadderEntryCharacter `json:"character"`
	Account    *Account             `json:"account,omitempty"`
}

type PrivateLeague struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type EventLadderEntry struct {
	Rank          int           `json:"rank"`
	Ineligible    *bool         `json:"ineligible,omitempty"`
	Time          *int          `json:"time,omitempty"`
	PrivateLeague PrivateLeague `json:"private_league"`
}

type AccountChallenges struct {
	Set       string `json:"set"`
	Completed int    `json:"completed"`
	Max       int    `json:"max"`
}

type AccountTwitchStream struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
}

type AccountTwitch struct {
	Name   string               `json:"name"`
	Stream *AccountTwitchStream `json:"stream,omitempty"`
}

type Guild struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type Account struct {
	Name       string             `json:"name"`
	Realm      *Realm             `json:"realm,omitempty"`
	Guild      *Guild             `json:"guild,omitempty"`
	Challenges *AccountChallenges `json:"challenges,omitempty"`
	Twitch     *AccountTwitch     `json:"twitch,omitempty"`
}

type PvPLadderTeamMember struct {
	Account   Account      `json:"account"`
	Character PvPCharacter `json:"character"`
	Public    *bool        `json:"public,omitempty"`
}

type PvPLadderTeamEntry struct {
	Rank                     int                   `json:"rank"`
	Rating                   *int                  `json:"rating,omitempty"`
	Points                   *int                  `json:"points,omitempty"`
	GamesPlayed              *int                  `json:"games_played,omitempty"`
	CumulativeOpponentPoints *int                  `json:"cumulative_opponent_points,omitempty"`
	LastGameTime             *string               `json:"last_game_time,omitempty"`
	Members                  []PvPLadderTeamMember `json:"members"`
}

type PvPMatch struct {
	Id            string  `json:"id"`
	Realm         *Realm  `json:"realm,omitempty"`
	StartAt       *string `json:"startAt,omitempty"`
	EndAt         *string `json:"endAt,omitempty"`
	URL           *string `json:"url,omitempty"`
	Description   string  `json:"description"`
	GlickoRatings bool    `json:"glickoRatings"`
	PvP           bool    `json:"pvp"`
	Style         string  `json:"style"`
	RegisterAt    *string `json:"registerAt,omitempty"`
	Complete      *bool   `json:"complete,omitempty"`
	Upcoming      *bool   `json:"upcoming,omitempty"`
	InProgress    *bool   `json:"inProgress,omitempty"`
}

type PublicStashChange struct {
	Id                string  `json:"id"`
	Public            bool    `json:"public"`
	AccountName       *string `json:"accountName,omitempty"`
	TeamId            int     `json:"teamId,omitempty"`
	Stash             *string `json:"stash,omitempty"`
	LastCharacterName *string `json:"lastCharacterName,omitempty"`
	StashType         string  `json:"stashType"`
	League            *string `json:"league,omitempty"`
	Items             []Item  `json:"items"`
}

func (p PublicStashChange) GetHash() [32]byte {
	var idAggregate strings.Builder
	idAggregate.WriteString(p.Id)
	for _, item := range p.Items {
		idAggregate.WriteString(item.Id)
		if item.StackSize != nil {
			idAggregate.WriteString(fmt.Sprint(*item.StackSize))
		}
	}
	return sha256.Sum256([]byte(idAggregate.String()))
}

type Ladder struct {
	Total       int           `json:"total"`
	CachedSince *string       `json:"cached_since,omitempty"`
	Entries     []LadderEntry `json:"entries"`
}

type EventLadder struct {
	Total   int                `json:"total"`
	Entries []EventLadderEntry `json:"entries"`
}

type PvPLadder struct {
	Total   int                  `json:"total"`
	Entries []PvPLadderTeamEntry `json:"entries"`
}

type ProfileGuild struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type ProfileTwitch struct {
	Name   string  `json:"name"`
	Stream *string `json:"stream,omitempty"`
}

type Profile struct {
	UUId   string         `json:"uuid"`
	Name   string         `json:"name"`
	Realm  *Realm         `json:"realm,omitempty"`
	Guild  *ProfileGuild  `json:"guild,omitempty"`
	Twitch *ProfileTwitch `json:"twitch,omitempty"`
}

type ItemFilterValidation struct {
	Valid     bool    `json:"valid"`
	Version   *string `json:"version,omitempty"`
	Validated *string `json:"validated,omitempty"`
}

type ItemFilter struct {
	Id          string                `json:"id"`
	FilterName  string                `json:"filter_name"`
	Realm       Realm                 `json:"realm"`
	Description string                `json:"description"`
	Version     string                `json:"version"`
	Type        string                `json:"type"`
	Public      *bool                 `json:"public,omitempty"`
	Filter      *string               `json:"filter,omitempty"`
	Validation  *ItemFilterValidation `json:"validation,omitempty"`
}

type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type ItemSocketType string

const (
	ItemSocketTypeGem   ItemSocketType = "gem"
	ItemSocketTypeJewel ItemSocketType = "jewel"
	ItemSocketTypeRune  ItemSocketType = "rune"
)

type ItemSocketItem string

// PoE2 only; emerald, sapphire, ruby, rune, soulcore, activegem, or supportgem
const (
	ItemSocketItemEmerald    ItemSocketItem = "emerald"
	ItemSocketItemSapphire   ItemSocketItem = "sapphire"
	ItemSocketItemRuby       ItemSocketItem = "ruby"
	ItemSocketItemRune       ItemSocketItem = "rune"
	ItemSocketItemSoulcore   ItemSocketItem = "soulcore"
	ItemSocketItemActiveGem  ItemSocketItem = "activegem"
	ItemSocketItemSupportGem ItemSocketItem = "supportgem"
)

type ItemSocket struct {
	Group   int             `json:"group"`
	Attr    *string         `json:"attr,omitempty"`
	SColour *string         `json:"sColour,omitempty"`
	Type    *ItemSocketType `json:"type,omitempty"` // PoE2 only
	Item    *ItemSocketItem `json:"item,omitempty"` // PoE2 only
}

type ItemValue []any

func (v ItemValue) Name() string {
	return v[0].(string)
}

func (v ItemValue) Id() int {
	return v[1].(int)
}

type ItemProperty struct {
	Name        string      `json:"name"`
	Values      []ItemValue `json:"values"`
	DisplayMode *int        `json:"displayMode,omitempty"`
	Progress    *float64    `json:"progress,omitempty"`
	Type        *int        `json:"type,omitempty"`
	Suffix      *string     `json:"suffix,omitempty"`
	Icon        *string     `json:"icon,omitempty"` // PoE2 only
}

type ItemInfluences struct {
	Elder   *bool `json:"elder,omitempty"`
	Shaper  *bool `json:"shaper,omitempty"`
	Searing *bool `json:"searing,omitempty"`
	Tangled *bool `json:"tangled,omitempty"`
}

type ItemReward struct {
	Label   string         `json:"label"`
	Rewards map[string]int `json:"rewards"`
}

type ItemLogbookModFaction struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ItemLogbookMod struct {
	Name    string                `json:"name"`
	Faction ItemLogbookModFaction `json:"faction"`
	Mods    []string              `json:"mods"`
}

type ItemUltimatumMod struct {
	Type string `json:"type"`
	Tier int    `json:"tier"`
}

type ItemIncubatedItem struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Progress int    `json:"progress"`
	Total    int    `json:"total"`
}

type ItemScourged struct {
	Tier     int  `json:"tier"`
	Level    *int `json:"level,omitempty"`
	Progress *int `json:"progress,omitempty"`
	Total    *int `json:"total,omitempty"`
}

type ItemCrucible struct {
	Layout string                  `json:"layout"`
	Nodes  map[string]CrucibleNode `json:"nodes"`
}

type ItemHybrid struct {
	IsVaalGem    *bool           `json:"isVaalGem,omitempty"`
	BaseTypeName string          `json:"baseTypeName"`
	Properties   *[]ItemProperty `json:"properties,omitempty"`
	ExplicitMods *[]string       `json:"explicitMods,omitempty"`
	SecDescrText *string         `json:"secDescrText,omitempty"`
}

type ItemExtended struct {
	Prefixes *int `json:"prefixes,omitempty"`
	Suffixes *int `json:"suffixes,omitempty"`
}

type GemSocket string

const (
	W GemSocket = "W"
)

type Item struct {
	Support                *bool               `json:"support,omitempty"`
	StackSize              *int                `json:"stackSize,omitempty"`
	Elder                  *bool               `json:"elder,omitempty"`
	Shaper                 *bool               `json:"shaper,omitempty"`
	Searing                *bool               `json:"searing,omitempty"`
	Tangled                *bool               `json:"tangled,omitempty"`
	AbyssJewel             *bool               `json:"abyssJewel,omitempty"`
	Delve                  *bool               `json:"delve,omitempty"`
	Fractured              *bool               `json:"fractured,omitempty"`
	Synthesised            *bool               `json:"synthesised,omitempty"`
	Sockets                *[]ItemSocket       `json:"sockets,omitempty"`
	SocketedItems          *[]Item             `json:"socketedItems,omitempty"`
	Name                   string              `json:"name"`
	TypeLine               string              `json:"typeLine"`
	BaseType               string              `json:"baseType"`
	Rarity                 *string             `json:"rarity,omitempty"`
	ItemLevel              *int                `json:"itemLevel,omitempty"`
	Ilvl                   int                 `json:"ilvl"`
	Duplicated             *bool               `json:"duplicated,omitempty"`
	Split                  *bool               `json:"split,omitempty"`
	Corrupted              *bool               `json:"corrupted,omitempty"`
	Unmodifiable           *bool               `json:"unmodifiable,omitempty"`
	Properties             *[]ItemProperty     `json:"properties,omitempty"`
	NotableProperties      *[]ItemProperty     `json:"notableProperties,omitempty"`
	AdditionalProperties   *[]ItemProperty     `json:"additionalProperties,omitempty"`
	WeaponRequirements     *[]ItemProperty     `json:"weaponRequirements,omitempty"`     // PoE2 only
	SupportGemRequirements *[]ItemProperty     `json:"supportGemRequirements,omitempty"` // PoE2 only
	GrantedSkills          *[]ItemProperty     `json:"grantedSkills,omitempty"`          // PoE2 only
	TalismanTier           *int                `json:"talismanTier,omitempty"`
	Rewards                *[]ItemReward       `json:"rewards,omitempty"`
	UtilityMods            *[]string           `json:"utilityMods,omitempty"`
	LogbookMods            *[]ItemLogbookMod   `json:"logbookMods,omitempty"`
	EnchantMods            *[]string           `json:"enchantMods,omitempty"`
	ScourgeMods            *[]string           `json:"scourgeMods,omitempty"`
	ImplicitMods           *[]string           `json:"implicitMods,omitempty"`
	UltimatumMods          *[]ItemUltimatumMod `json:"ultimatumMods,omitempty"`
	ExplicitMods           *[]string           `json:"explicitMods,omitempty"`
	CraftedMods            *[]string           `json:"craftedMods,omitempty"`
	FracturedMods          *[]string           `json:"fracturedMods,omitempty"`
	CosmeticMods           *[]string           `json:"cosmeticMods,omitempty"`
	VeiledMods             *[]string           `json:"veiledMods,omitempty"`
	Veiled                 *bool               `json:"veiled,omitempty"`
	IsRelic                *bool               `json:"isRelic,omitempty"`
	FoilVariation          *int                `json:"foilVariation,omitempty"`
	Foreseeing             *bool               `json:"foreseeing,omitempty"`
	IncubatedItem          *ItemIncubatedItem  `json:"incubatedItem,omitempty"`
	Ruthless               *bool               `json:"ruthless,omitempty"`
	FrameType              *int                `json:"frameType,omitempty"`
	FrameTypeId            string              `json:"frameTypeId"`
	Hybrid                 *ItemHybrid         `json:"hybrid,omitempty"`
	Extended               *ItemExtended       `json:"extended,omitempty"`
	Socket                 *int                `json:"socket,omitempty"`
	Colour                 *string             `json:"colour,omitempty"`
	GemSockets             *[]GemSocket        `json:"gemSockets,omitempty"`
	Influences             *map[string]bool    `json:"influences,omitempty"`
	MemoryItem             *bool               `json:"memoryItem,omitempty"`
	X                      *int                `json:"x,omitempty"`
	Y                      *int                `json:"y,omitempty"`
	InventoryId            *string             `json:"inventoryId,omitempty"`
	Id                     string              `json:"id"`
	Identified             bool                `json:"identified"`
	Icon                   string              `json:"icon"`
	Mutated                *bool               `json:"mutated,omitempty"`
	MutatedMods            *[]string           `json:"mutatedMods,omitempty"`
	DoubleCorrupted        *bool               `json:"doubleCorrupted,omitempty"` // PoE2 only
	BondedMods             *[]string           `json:"bondedMods,omitempty"`      // PoE2 only
	Desecrated             *bool               `json:"desecrated,omitempty"`      // PoE2 only
	DesecratedMods         *[]string           `json:"desecratedMods,omitempty"`  // PoE2 only
	RuneMods               *[]string           `json:"runeMods,omitempty"`        // PoE2 only
	W                      int                 `json:"w,omitempty"`
	H                      int                 `json:"h,omitempty"`

	ObjectiveId int `json:"objectiveId,omitempty"` // filled by us and not ggg

	Requirements          *[]ItemProperty `json:"requirements,omitempty"`
	GemTabs               *[]GemTab       `json:"gemTabs,omitempty"`          // PoE2 only
	GemBackground         *string         `json:"gemBackground,omitempty"`    // PoE2 only
	GemSkill              *string         `json:"gemSkill,omitempty"`         // PoE2 only
	UnidentifiedTier      *int            `json:"unidentifiedTier,omitempty"` // PoE2 only
	Realm                 *Realm          `json:"realm,omitempty"`            // PoE2 only
	Verified              bool            `json:"verified"`
	MaxStackSize          *int            `json:"maxStackSize,omitempty"`
	StackSizeText         *string         `json:"stackSizeText,omitempty"`
	League                string          `json:"league"`
	Note                  *string         `json:"note,omitempty"`
	ForumNote             *string         `json:"forum_note,omitempty"`
	LockedToCharacter     *bool           `json:"lockedToCharacter,omitempty"`
	LockedToAccount       *bool           `json:"lockedToAccount,omitempty"`
	CisRaceReward         *bool           `json:"cisRaceReward,omitempty"`
	SeaRaceReward         *bool           `json:"seaRaceReward,omitempty"`
	ThRaceReward          *bool           `json:"thRaceReward,omitempty"`
	NextLevelRequirements *[]ItemProperty `json:"nextLevelRequirements,omitempty"`
	SecDescrText          *string         `json:"secDescrText,omitempty"`
	DescrText             *string         `json:"descrText,omitempty"`
	FlavourText           *[]string       `json:"flavourText,omitempty"`
	FlavourTextParsed     *[]any          `json:"flavourTextParsed,omitempty"`
	FlavourTextNote       *string         `json:"flavourTextNote,omitempty"`
	ProphecyText          *string         `json:"prophecyText,omitempty"`
	Replica               *bool           `json:"replica,omitempty"`
	Scourged              *ItemScourged   `json:"scourged,omitempty"`
	ArtFilename           *string         `json:"artFilename,omitempty"`
	SocketedIcon          *string         `json:"socketedIcon,omitempty"`         // PoE2 only
	TamedBeastProperties  *[]ItemProperty `json:"tamedBeastProperties,omitempty"` // PoE2 only
	BuiltInSupport        *string         `json:"builtInSupport,omitempty"`       // PoE1 only; Supported by level 1 x
	MonsterLevel          *int            `json:"monsterLevel,omitempty"`         // PoE1 only; used for items that always display their monster level
	IconTierText          *string         `json:"iconTierText,omitempty"`         // usually roman numerals
	Sanctified            *bool           `json:"sanctified,omitempty"`           // PoE2 only
}

func (i *Item) GetPositionIndex() string {
	return fmt.Sprintf("%s-%d-%d", utils.Deref(i.InventoryId), utils.Deref(i.X), utils.Deref(i.Y))
}

func (i *Item) Equals(other Item) bool {
	if i.Name != other.Name || i.TypeLine != other.TypeLine || i.BaseType != other.BaseType {
		return false
	}
	return modsEqual(i.ImplicitMods, other.ImplicitMods) &&
		modsEqual(i.ExplicitMods, other.ExplicitMods) &&
		modsEqual(i.CraftedMods, other.CraftedMods) &&
		modsEqual(i.EnchantMods, other.EnchantMods) &&
		modsEqual(i.FracturedMods, other.FracturedMods)
}

func modsEqual(a, b *[]string) bool {
	if (a != nil && b == nil) || (a == nil && b != nil) {
		return false
	}
	return (a == nil && b == nil) || utils.ArrayEqualsUnordered(*a, *b)
}

func (i *Item) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value: not a byte slice")
	}
	return json.Unmarshal(bytes, i)
}

// Value implements the driver.Valuer interface for Item
func (i Item) Value() (driver.Value, error) {
	return json.Marshal(i)
}

type GemTab struct {
	Name  *string   `json:"name"`
	Pages []GemPage `json:"pages"`
}

type GemPage struct {
	SkillName   *string         `json:"skillName"`
	Description *string         `json:"description"`
	Properties  *[]ItemProperty `json:"properties"`
	Stats       *[]string       `json:"stats"`
}

type Specialisations struct {
	Set1       *[]int `json:"set1,omitempty"`
	Set2       *[]int `json:"set2,omitempty"`
	Shapeshift *[]int `json:"shapeshift,omitempty"`
}

type Passives struct {
	Hashes              []int                    `json:"hashes"`
	HashesEx            []int                    `json:"hashes_ex"`
	MasteryEffects      map[int]int              `json:"mastery_effects"`
	SkillOverrides      map[string]PassiveNode   `json:"skill_overrides"`
	BanditChoice        *string                  `json:"bandit_choice,omitempty"`
	PantheonMajor       *string                  `json:"pantheon_major,omitempty"`
	PantheonMinor       *string                  `json:"pantheon_minor,omitempty"`
	JewelData           map[string]ItemJewelData `json:"jewel_data"`
	AlternateAscendancy *string                  `json:"alternate_ascendancy,omitempty"`
	Specialisations     *Specialisations         `json:"specialisation,omitempty"` // PoE2 only
	QuestStats          *[]string                `json:"quest_stats,omitempty"`    // PoE2 only; passives granted via quests
}

type Metadata struct {
	Version string `json:"version"`
}

type ItemJewelDataSubgraph struct {
	Groups map[string]PassiveGroup `json:"groups"`
	Nodes  map[string]PassiveNode  `json:"nodes"`
}

type ItemJewelData struct {
	Type         string                 `json:"type"`
	Radius       *int                   `json:"radius,omitempty"`
	RadiusMin    *int                   `json:"radiusMin,omitempty"`
	RadiusVisual *string                `json:"radiusVisual,omitempty"`
	Subgraph     *ItemJewelDataSubgraph `json:"subgraph,omitempty"`
}

type StashTabMetadata struct {
	Public *bool   `json:"public,omitempty"`
	Folder *bool   `json:"folder,omitempty"`
	Colour *string `json:"colour,omitempty"`
	Layout *any    `json:"layout,omitempty"`
	Items  *int    `json:"items,omitempty"`
}

type StashTabLayoutItem struct {
	Section *string  `json:"section,omitempty"`
	X       float64  `json:"x,omitempty"`
	Y       float64  `json:"y,omitempty"`
	W       int      `json:"w,omitempty"`
	H       int      `json:"h,omitempty"`
	Scale   *float64 `json:"scale,omitempty"`
	Hidden  *bool    `json:"hidden,omitempty"`
}

type StashTab struct {
	Id       string           `json:"id"`
	Parent   *string          `json:"parent,omitempty"`
	Name     string           `json:"name"`
	Type     string           `json:"type"`
	Index    *int             `json:"index,omitempty"`
	Metadata StashTabMetadata `json:"metadata"`
	Children *[]StashTab      `json:"children,omitempty"`
	Items    *[]Item          `json:"items,omitempty"`
}

type GuildStashTabGGG struct {
	*StashTab
	Items    *[]Item             `json:"items,omitempty"`
	Children *[]GuildStashTabGGG `json:"children,omitempty"`
}

func (g *GuildStashTabGGG) FlatMap() []*GuildStashTabGGG {
	var result []*GuildStashTabGGG
	result = append(result, g)
	if g.Children != nil {
		for _, child := range *g.Children {
			result = append(result, child.FlatMap()...)
		}
	}
	return result
}

func (g *GuildStashTabGGG) GetHash() [32]byte {
	var idAggregate strings.Builder
	if g.StashTab != nil {
		idAggregate.WriteString(g.Id)
	}
	for _, item := range utils.Deref(g.Items) {
		idAggregate.WriteString(item.Id)
		if item.StackSize != nil {
			idAggregate.WriteString(fmt.Sprint(*item.StackSize))
		}
	}
	return sha256.Sum256([]byte(idAggregate.String()))
}

type AtlasPassiveTree struct {
	Name   string `json:"name"`
	Hashes []int  `json:"hashes"`
}

type AtlasPassives struct {
	Hashes []int `json:"hashes"`
}

type LeagueAccount struct {
	AtlasPassives     *AtlasPassives     `json:"atlas_passives,omitempty"`
	AtlasPassiveTrees []AtlasPassiveTree `json:"atlas_passive_trees"`
}

type PassiveGroup struct {
	X       float64  `json:"x"`
	Y       float64  `json:"y"`
	Orbits  []int    `json:"orbits"`
	IsProxy *bool    `json:"isProxy,omitempty"`
	Proxy   *string  `json:"proxy,omitempty"`
	Nodes   []string `json:"nodes"`
}

type PassiveNodeMasteryEffect struct {
	Effect       int       `json:"effect"`
	Stats        []string  `json:"stats"`
	ReminderText *[]string `json:"reminderText,omitempty"`
}

type PassiveNodeExpansionJewel struct {
	Size   *int    `json:"size,omitempty"`
	Index  *int    `json:"index,omitempty"`
	Proxy  *string `json:"proxy,omitempty"`
	Parent *string `json:"parent,omitempty"`
}

type PassiveNode struct {
	Skill                  *string                     `json:"skill,omitempty"` // actually an int but it's a string in the ggg response
	Name                   *string                     `json:"name,omitempty"`
	Icon                   *string                     `json:"icon,omitempty"`
	IsKeystone             *bool                       `json:"isKeystone,omitempty"`
	IsNotable              *bool                       `json:"isNotable,omitempty"`
	IsMastery              *bool                       `json:"isMastery,omitempty"`
	InactiveIcon           *string                     `json:"inactiveIcon,omitempty"`
	ActiveIcon             *string                     `json:"activeIcon,omitempty"`
	ActiveEffectImage      *string                     `json:"activeEffectImage,omitempty"`
	MasteryEffects         *[]PassiveNodeMasteryEffect `json:"masteryEffects,omitempty"`
	IsBlighted             *bool                       `json:"isBlighted,omitempty"`
	IsTattoo               *bool                       `json:"isTattoo,omitempty"`
	IsProxy                *bool                       `json:"isProxy,omitempty"`
	IsJewelSocket          *bool                       `json:"isJewelSocket,omitempty"`
	ExpansionJewel         *PassiveNodeExpansionJewel  `json:"expansionJewel,omitempty"`
	Recipe                 *[]string                   `json:"recipe,omitempty"`
	GrantedStrength        *int                        `json:"grantedStrength,omitempty"`
	GrantedDexterity       *int                        `json:"grantedDexterity,omitempty"`
	GrantedIntelligence    *int                        `json:"grantedIntelligence,omitempty"`
	AscendancyName         *string                     `json:"ascendancyName,omitempty"`
	IsAscendancyStart      *bool                       `json:"isAscendancyStart,omitempty"`
	IsMultipleChoice       *bool                       `json:"isMultipleChoice,omitempty"`
	IsMultipleChoiceOption *bool                       `json:"isMultipleChoiceOption,omitempty"`
	GrantedPassivePoints   *int                        `json:"grantedPassivePoints,omitempty"`
	Stats                  *[]string                   `json:"stats,omitempty"`
	ReminderText           *[]string                   `json:"reminderText,omitempty"`
	FlavourText            *[]string                   `json:"flavourText,omitempty"`
	ClassStartIndex        *int                        `json:"classStartIndex,omitempty"`
	Group                  *string                     `json:"group,omitempty"`
	Orbit                  *int                        `json:"orbit,omitempty"`
	OrbitIndex             *int                        `json:"orbitIndex,omitempty"`
	Out                    []string                    `json:"out"`
	In                     []string                    `json:"in"`
}

type CrucibleNode struct {
	Skill        *int      `json:"skill,omitempty"`
	Tier         *int      `json:"tier,omitempty"`
	Icon         *string   `json:"icon,omitempty"`
	Allocated    *bool     `json:"allocated,omitempty"`
	IsNotable    *bool     `json:"isNotable,omitempty"`
	IsReward     *bool     `json:"isReward,omitempty"`
	Stats        *[]string `json:"stats,omitempty"`
	ReminderText *[]string `json:"reminderText,omitempty"`
	Orbit        *int      `json:"orbit,omitempty"`
	OrbitIndex   *int      `json:"orbitIndex,omitempty"`
	Out          []string  `json:"out"`
	In           []string  `json:"in"`
}

type PvPCharacter struct {
	Id     string  `json:"id"`
	Name   string  `json:"name"`
	Level  int     `json:"level"`
	Class  string  `json:"class"`
	League *string `json:"league,omitempty"`
	Score  *int    `json:"score,omitempty"`
}

type GGGCharacter struct {
	Id         string   `json:"id"`
	Name       string   `json:"name"`
	Realm      Realm    `json:"realm"`
	Class      string   `json:"class"`
	League     *string  `json:"league,omitempty"`
	Level      int      `json:"level"`
	Experience int      `json:"experience"`
	Ruthless   *bool    `json:"ruthless,omitempty"`
	Expired    *bool    `json:"expired,omitempty"`
	Deleted    *bool    `json:"deleted,omitempty"`
	Current    *bool    `json:"current,omitempty"`
	Equipment  *[]Item  `json:"equipment"`
	Skills     *[]Item  `json:"skills,omitempty"` // PoE2 only
	Inventory  *[]Item  `json:"inventory"`
	Rucksack   *[]Item  `json:"rucksack,omitempty"`
	Jewels     *[]Item  `json:"jewels"`
	Passives   Passives `json:"passives"`
	Metadata   Metadata `json:"metadata"`
}

func (c *GGGCharacter) GetAllItems() []Item {
	if c == nil {
		return []Item{}
	}
	items := make([]Item, 0)
	if c.Equipment != nil {
		for _, item := range *c.Equipment {
			items = append(items, item)
			if item.SocketedItems != nil {
				items = append(items, *item.SocketedItems...)
			}
		}
	}
	if c.Jewels != nil {
		items = append(items, *c.Jewels...)
	}
	return items
}

func (c *GGGCharacter) GetVoidStones() utils.Set[string] {
	voidStones := utils.Set[string]{}
	if c.Inventory == nil {
		return voidStones
	}
	for _, item := range *c.Inventory {
		if strings.Contains(item.BaseType, "Voidstone") {
			voidStones.Add(item.BaseType)
		}
	}
	return voidStones
}

func (c *GGGCharacter) GetNumberOfHighIlvlFlasks() int {
	if c.Equipment == nil {
		return 0
	}
	count := 0
	for _, item := range *c.Equipment {
		if strings.Contains(item.BaseType, "Flask") && item.Ilvl >= 84 {
			count++
		}
	}
	return count
}

func (c *GGGCharacter) HasSameEquipment(other *GGGCharacter) bool {
	if other == nil {
		return false
	}
	if (c.Equipment == nil && other.Equipment == nil) && (c.Jewels == nil && other.Jewels == nil) {
		return true
	}
	if (c.Equipment == nil && other.Equipment != nil) || (c.Equipment != nil && other.Equipment == nil) {
		return false
	}
	if (c.Jewels == nil && other.Jewels != nil) || (c.Jewels != nil && other.Jewels == nil) {
		return false
	}
	if len(*c.Equipment) != len(*other.Equipment) || len(*c.Jewels) != len(*other.Jewels) {
		return false
	}
	itemMap := make(map[string]Item)
	for _, item := range *c.Equipment {
		itemMap[item.GetPositionIndex()] = item
	}
	for _, item := range *c.Jewels {
		itemMap[item.GetPositionIndex()] = item
	}
	for _, item := range *other.Equipment {
		otherItem, exists := itemMap[item.GetPositionIndex()]
		if !exists || !item.Equals(otherItem) {
			return false
		}
	}
	for _, item := range *other.Jewels {
		otherItem, exists := itemMap[item.GetPositionIndex()]
		if !exists || !item.Equals(otherItem) {
			return false
		}
	}
	return c.HasSameGems(other)
}

func (c *GGGCharacter) HasSameGems(other *GGGCharacter) bool {
	if other == nil {
		return false
	}
	gems := utils.Set[string]{}
	for _, item := range utils.Deref(c.Equipment) {
		for _, socketedItem := range utils.Deref(item.SocketedItems) {
			gems[socketedItem.TypeLine] = true
		}
	}
	otherGems := utils.Set[string]{}
	for _, item := range utils.Deref(other.Equipment) {
		for _, socketedItem := range utils.Deref(item.SocketedItems) {
			otherGems[socketedItem.TypeLine] = true
		}
	}
	return len(gems.Difference(otherGems)) == 0
}

type MinimalCharacter struct {
	Id         string  `json:"id"`
	Name       string  `json:"name"`
	Realm      Realm   `json:"realm"`
	Class      string  `json:"class"`
	League     *string `json:"league,omitempty"`
	Level      int     `json:"level"`
	Experience int     `json:"experience"`
	Ruthless   *bool   `json:"ruthless,omitempty"`
	Expired    *bool   `json:"expired,omitempty"`
	Deleted    *bool   `json:"deleted,omitempty"`
	Current    *bool   `json:"current,omitempty"`
}

type PvPMatchLadder struct {
	Total   int                  `json:"total"`
	Entries []PvPLadderTeamEntry `json:"entries"`
}

type Twitch struct {
	Name string `json:"name"`
}

type ListLeaguesResponse struct {
	Leagues []League `json:"leagues"`
}

type ListCharactersResponse struct {
	Characters []MinimalCharacter `json:"characters"`
}

type GetLeagueResponse struct {
	League *League `json:"league"`
}

type GetLeagueLadderResponse struct {
	League *League `json:"league"`
	Ladder *Ladder `json:"ladder"`
}

type GetLeagueEventLadderResponse struct {
	League *League      `json:"league"`
	Ladder *EventLadder `json:"ladder"`
}

type GetPvPMatchesResponse struct {
	Matches []PvPMatch `json:"matches"`
}

type GetPvPMatchResponse struct {
	Match *PvPMatch `json:"match"`
}

type GetPvPMatchLadderResponse struct {
	Match  PvPMatch       `json:"match"`
	Ladder PvPMatchLadder `json:"ladder"`
}

type GetAccountProfileResponse struct {
	UUId   string  `json:"uuid"`
	Name   string  `json:"name"`
	Realm  *Realm  `json:"realm"`
	Guild  *Guild  `json:"guild"`
	Twitch *Twitch `json:"twitch"`
}

type GetCharacterResponse struct {
	Character *GGGCharacter `json:"character"`
}

type ListAccountStashesResponse struct {
	Stashes []PublicStashChange `json:"stashes"`
}

type GetAccountStashResponse struct {
	Stash *PublicStashChange `json:"stash"`
}

type ListItemFiltersResponse struct {
	Filters []ItemFilter `json:"filters"`
}

type GetItemFilterResponse struct {
	Filter *ItemFilter `json:"filter"`
}

type CreateItemFilterResponse struct {
	Filter ItemFilter `json:"filter"`
}

type UpdateItemFilterResponse struct {
	Filter ItemFilter `json:"filter"`
	Error  *Error     `json:"error"`
}

type CreateFilterBody struct {
	FilterName  string  `json:"filter_name"`
	Realm       Realm   `json:"realm"`
	Description *string `json:"description"`
	Version     *string `json:"version"`
	Type        *string `json:"type"`
	Public      *bool   `json:"public"`
	Filter      string  `json:"filter"`
}

type UpdateFilterBody struct {
	FilterName  *string `json:"filter_name"`
	Realm       *Realm  `json:"realm"`
	Description *string `json:"description"`
	Version     *string `json:"version"`
	Type        *string `json:"type"`
	Public      *bool   `json:"public"`
	Filter      *string `json:"filter"`
}

type GetLeagueAccountResponse struct {
	LeagueAccount LeagueAccount `json:"league_account"`
}

type ListGuildStashesResponse struct {
	Stashes []GuildStashTabGGG `json:"stashes"`
}

type GetGuildStashResponse struct {
	Stash *GuildStashTabGGG `json:"stash"`
}

type GetPublicStashTabsResponse struct {
	NextChangeId string              `json:"next_change_id"`
	Stashes      []PublicStashChange `json:"stashes"`
}

type ClientCredentialsGrantResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   *int   `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Username    string `json:"username"`
	Sub         string `json:"sub"`
	Scope       string `json:"scope"`
}

type AccessTokenGrantResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Username     string `json:"username"`
	Sub          string `json:"sub"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

type GetPoE2LadderResponse struct {
	Context *GetLeagueLadderResponse `json:"context"`
}
