package client

import (
	"bpl/utils"
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
)

type PoBItem struct {
	Rarity   string `xml:"-"`
	Name     string `xml:"-"`
	BaseType string `xml:"-"`
}

type pobItemXML struct {
	ID   int    `xml:"id,attr"`
	Text string `xml:",chardata"`
}

func (p *PoBItem) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var aux pobItemXML
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}
	lines := strings.Split(aux.Text, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	if len(cleaned) > 0 {
		if after, ok := strings.CutPrefix(cleaned[0], "Rarity:"); ok {
			p.Rarity = strings.TrimSpace(after)
		} else {
			p.Rarity = cleaned[0]
		}
	}
	switch strings.ToUpper(p.Rarity) {
	case "NORMAL":
		if len(cleaned) > 1 {
			p.BaseType = cleaned[1]
		}
		p.Name = ""
	case "MAGIC":
		if len(cleaned) > 1 {
			p.Name = cleaned[1]
		}
		p.BaseType = ""
	default:
		if len(cleaned) > 1 {
			p.Name = cleaned[1]
		}
		if len(cleaned) > 2 {
			p.BaseType = cleaned[2]
		}
	}
	return nil
}

type Slot struct {
	ItemId int    `xml:"itemId,attr"`
	Name   string `xml:"name,attr"`
}

type PathOfBuilding struct {
	Build  Build     `xml:"Build"`
	Skills Skills    `xml:"Skills"`
	Tree   Tree      `xml:"Tree"`
	Items  []PoBItem `xml:"Items>Item"`
	Slots  []Slot    `xml:"Items>ItemSet>Slot"`
}

func (p *PathOfBuilding) NumEquipmentPieces() int {
	requiredSlots := []string{
		"Flask 1",
		"Flask 2",
		"Flask 3",
		"Flask 4",
		"Flask 5",
		"Belt",
		"Amulet",
		"Ring 1",
		"Ring 2",
		"Boots",
		"Gloves",
		"Body Armour",
		"Helmet",
	}
	if strings.Contains(p.GetMainSkill(), "Concoction") {
		requiredSlots = append(requiredSlots, "Weapon 2")
	} else {
		requiredSlots = append(requiredSlots, "Weapon 1")
	}
	count := 0
	for _, slot := range p.Slots {
		if slices.Contains(requiredSlots, slot.Name) && slot.ItemId != 0 {
			count++
		}
	}
	return count
}

func (current *PathOfBuilding) SwappedWeaponsAndLostDps(previous *PathOfBuilding) bool {
	if current.Build.PlayerStats.FullDPS > previous.Build.PlayerStats.FullDPS {
		return false
	}
	currentW1 := ""
	currentW2 := ""
	previousW1Swap := ""
	previousW2Swap := ""
	for _, slot := range current.Slots {
		if slot.Name == "Weapon 1" && slot.ItemId != 0 {
			currentW1 = current.Items[slot.ItemId-1].Name + current.Items[slot.ItemId-1].BaseType
		}
		if slot.Name == "Weapon 2" && slot.ItemId != 0 {
			currentW2 = current.Items[slot.ItemId-1].Name + current.Items[slot.ItemId-1].BaseType
		}
	}
	for _, slot := range previous.Slots {
		if slot.Name == "Weapon 1 Swap" && slot.ItemId != 0 {
			previousW1Swap = previous.Items[slot.ItemId-1].Name + previous.Items[slot.ItemId-1].BaseType
		}
		if slot.Name == "Weapon 2 Swap" && slot.ItemId != 0 {
			previousW2Swap = previous.Items[slot.ItemId-1].Name + previous.Items[slot.ItemId-1].BaseType
		}
	}
	return currentW1 == previousW1Swap && currentW2 == previousW2Swap
}

func (p *PathOfBuilding) GetMainSkill() string {
	if p.Skills.ActiveSkillSet == 0 || len(p.Skills.SkillSets) == 0 || len(p.Skills.SkillSets[0].Skills) < p.Skills.ActiveSkillSet {
		return ""
	}
	for _, gem := range p.Skills.SkillSets[0].Skills[p.Build.MainSocketGroup-1].Gems {
		return gem.NameSpec
	}
	return ""
}

func (p *PathOfBuilding) GetPassives() []int {
	nodes := strings.Split(p.Tree.Spec.Nodes, ",")
	passives := make([]int, 0, len(nodes))
	for _, n := range nodes {
		id, err := strconv.Atoi(n)
		if err != nil {
			continue
		}
		passives = append(passives, id)
	}
	masteryEffects := strings.SplitSeq(p.Tree.Spec.MasteryEffects[1:len(p.Tree.Spec.MasteryEffects)-1], "},{")
	for me := range masteryEffects {
		parts := strings.Split(me, ",")
		if len(parts) != 2 {
			continue
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		passives = append(passives, id)
	}
	return passives
}

type Tree struct {
	Spec Spec `xml:"Spec"`
}

type FullDpsSkill struct {
	Value     float64 `xml:"value,attr"`
	Stat      string  `xml:"stat,attr"`
	Source    string  `xml:"source,attr"`
	Skillpart string  `xml:"skillpart,attr"`
}

type Build struct {
	PlayerStats     PlayerStats
	FullDpsSkils    []FullDpsSkill
	MainSocketGroup int `xml:"mainSocketGroup,attr"`
}

// <Spec secondaryAscendClassId="nil" masteryEffects="{25535,30612},{53188,64875},{2828,11521},{34927,55308},{53517,45653},{32241,34242},{38921,4500},{47197,28638},{64406,7137}" nodes="4184,40291,54447,15167,26725,34423,50340,57197,64426,19595,20832,49254,2828,44967,19919,2550,33287,48118,24383,37569,50422,55743,47197,14057,53517,36542,6712,33435,40653,1031,11924,57264,24362,4397,6043,25989,53188,58218,9052,18715,34661,32241,62017,44429,53793,34506,24872,4367,27166,60501,5743,3533,30302,41472,25970,6764,31931,5935,8135,32932,48822,21958,19501,11420,12536,61471,61198,15631,5152,4917,14040,48929,32738,62214,30380,13559,24324,58402,11088,55649,44184,14021,36949,16023,61259,59928,53118,33755,54279,31080,44799,15064,9386,31462,49412,2219,14936,25535,29781,54396,65108,64210,21974,60031,34927,885,6446,38921,37114,40366,10221,25831,10031,64406,56461,55332,6052,6967,1203,17735,7938,42760,26866,10893,62363,7374,38119,11128,22285,61804" treeVersion="3_26" ascendClassId="2" classId="3">

type Spec struct {
	MasteryEffects string `xml:"masteryEffects,attr"`
	Nodes          string `xml:"nodes,attr"`
	// TreeVersion    string `xml:"treeVersion,attr"`
	// AscendClassId  string `xml:"ascendClassId,attr"`
	// ClassId        string `xml:"classId,attr"`
}

type PlayerStats struct {
	AverageDamage                   float64
	AverageBurstDamage              float64
	Speed                           float64
	PreEffectiveCritChance          float64
	CritChance                      float64
	CritMultiplier                  float64
	HitChance                       float64
	TotalDPS                        float64
	TotalDot                        float64
	WithBleedDPS                    float64
	WithIgniteDPS                   float64
	PoisonDPS                       float64
	PoisonDamage                    float64
	WithPoisonDPS                   float64
	TotalDotDPS                     float64
	CullingDPS                      float64
	ReservationDPS                  float64
	CombinedDPS                     float64
	AreaOfEffectRadiusMetres        float64
	ManaCost                        float64
	ManaPercentCost                 float64
	ManaPerSecondCost               float64
	ManaPercentPerSecondCost        float64
	LifeCost                        float64
	LifePercentCost                 float64
	LifePerSecondCost               float64
	LifePercentPerSecondCost        float64
	ESCost                          float64
	ESPerSecondCost                 float64
	ESPercentPerSecondCost          float64
	RageCost                        float64
	SoulCost                        float64
	Str                             float64
	ReqStr                          float64
	Dex                             float64
	ReqDex                          float64
	Int                             float64
	ReqInt                          float64
	Devotion                        float64
	TotalEHP                        float64
	PhysicalMaximumHitTaken         float64
	LightningMaximumHitTaken        float64
	FireMaximumHitTaken             float64
	ColdMaximumHitTaken             float64
	ChaosMaximumHitTaken            float64
	MainHandAccuracy                float64
	Life                            float64
	SpecLifeInc                     float64
	LifeUnreserved                  float64
	LifeRecoverable                 float64
	LifeUnreservedPercent           float64
	LifeRegenRecovery               float64
	LifeLeechGainRate               float64
	Mana                            float64
	SpecManaInc                     float64
	ManaUnreserved                  float64
	ManaUnreservedPercent           float64
	ManaRegenRecovery               float64
	ManaLeechGainRate               float64
	EnergyShield                    float64
	EnergyShieldRecoveryCap         float64
	SpecEnergyShieldInc             float64
	EnergyShieldRegenRecovery       float64
	EnergyShieldLeechGainRate       float64
	Ward                            float64
	RageRegenRecovery               float64
	TotalBuildDegen                 float64
	TotalNetRegen                   float64
	NetLifeRegen                    float64
	NetManaRegen                    float64
	NetEnergyShieldRegen            float64
	Evasion                         float64
	SpecEvasionInc                  float64
	MeleeEvadeChance                float64
	ProjectileEvadeChance           float64
	Armour                          float64
	SpecArmourInc                   float64
	PhysicalDamageReduction         float64
	EffectiveBlockChance            float64
	EffectiveSpellBlockChance       float64
	AttackDodgeChance               float64
	SpellDodgeChance                float64
	EffectiveSpellSuppressionChance float64
	FireResist                      float64
	FireResistOverCap               float64
	ColdResist                      float64
	ColdResistOverCap               float64
	LightningResist                 float64
	LightningResistOverCap          float64
	ChaosResist                     float64
	ChaosResistOverCap              float64
	EffectiveMovementSpeedMod       float64
	FullDPS                         float64
	FullDotDPS                      float64
	PowerCharges                    float64
	PowerChargesMax                 float64
	FrenzyCharges                   float64
	FrenzyChargesMax                float64
	EnduranceCharges                float64
	EnduranceChargesMax             float64
}

type playerStatXML struct {
	Stat  string  `xml:"stat,attr"`
	Value float64 `xml:"value,attr"`
}

// Custom XML unmarshal to handle "inf" values
func (ps *playerStatXML) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var aux struct {
		Stat  string `xml:"stat,attr"`
		Value string `xml:"value,attr"`
	}
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}
	ps.Stat = aux.Stat
	if aux.Value == "inf" {
		ps.Value = float64(math.MaxFloat64)
	} else {
		val, err := strconv.ParseFloat(aux.Value, 64)
		if err != nil {
			return err
		}
		ps.Value = val
	}
	return nil
}

func (ps *PlayerStats) SetStat(stat string, value float64) {
	switch stat {
	case "AverageDamage":
		ps.AverageDamage = value
	case "AverageBurstDamage":
		ps.AverageBurstDamage = value
	case "Speed":
		ps.Speed = value
	case "PreEffectiveCritChance":
		ps.PreEffectiveCritChance = value
	case "CritChance":
		ps.CritChance = value
	case "CritMultiplier":
		ps.CritMultiplier = value
	case "HitChance":
		ps.HitChance = value
	case "TotalDPS":
		ps.TotalDPS = value
	case "TotalDot":
		ps.TotalDot = value
	case "WithBleedDPS":
		ps.WithBleedDPS = value
	case "WithIgniteDPS":
		ps.WithIgniteDPS = value
	case "PoisonDPS":
		ps.PoisonDPS = value
	case "PoisonDamage":
		ps.PoisonDamage = value
	case "WithPoisonDPS":
		ps.WithPoisonDPS = value
	case "TotalDotDPS":
		ps.TotalDotDPS = value
	case "CullingDPS":
		ps.CullingDPS = value
	case "ReservationDPS":
		ps.ReservationDPS = value
	case "CombinedDPS":
		ps.CombinedDPS = value
	case "AreaOfEffectRadiusMetres":
		ps.AreaOfEffectRadiusMetres = value
	case "ManaCost":
		ps.ManaCost = value
	case "ManaPercentCost":
		ps.ManaPercentCost = value
	case "ManaPerSecondCost":
		ps.ManaPerSecondCost = value
	case "ManaPercentPerSecondCost":
		ps.ManaPercentPerSecondCost = value
	case "LifeCost":
		ps.LifeCost = value
	case "LifePercentCost":
		ps.LifePercentCost = value
	case "LifePerSecondCost":
		ps.LifePerSecondCost = value
	case "LifePercentPerSecondCost":
		ps.LifePercentPerSecondCost = value
	case "ESCost":
		ps.ESCost = value
	case "ESPerSecondCost":
		ps.ESPerSecondCost = value
	case "ESPercentPerSecondCost":
		ps.ESPercentPerSecondCost = value
	case "RageCost":
		ps.RageCost = value
	case "SoulCost":
		ps.SoulCost = value
	case "Str":
		ps.Str = value
	case "ReqStr":
		ps.ReqStr = value
	case "Dex":
		ps.Dex = value
	case "ReqDex":
		ps.ReqDex = value
	case "Int":
		ps.Int = value
	case "ReqInt":
		ps.ReqInt = value
	case "Devotion":
		ps.Devotion = value
	case "TotalEHP":
		ps.TotalEHP = value
	case "PhysicalMaximumHitTaken":
		ps.PhysicalMaximumHitTaken = value
	case "LightningMaximumHitTaken":
		ps.LightningMaximumHitTaken = value
	case "FireMaximumHitTaken":
		ps.FireMaximumHitTaken = value
	case "ColdMaximumHitTaken":
		ps.ColdMaximumHitTaken = value
	case "ChaosMaximumHitTaken":
		ps.ChaosMaximumHitTaken = value
	case "MainHandAccuracy":
		ps.MainHandAccuracy = value
	case "Life":
		ps.Life = value
	case "Spec:LifeInc":
		ps.SpecLifeInc = value
	case "LifeUnreserved":
		ps.LifeUnreserved = value
	case "LifeRecoverable":
		ps.LifeRecoverable = value
	case "LifeUnreservedPercent":
		ps.LifeUnreservedPercent = value
	case "LifeRegenRecovery":
		ps.LifeRegenRecovery = value
	case "LifeLeechGainRate":
		ps.LifeLeechGainRate = value
	case "Mana":
		ps.Mana = value
	case "Spec:ManaInc":
		ps.SpecManaInc = value
	case "ManaUnreserved":
		ps.ManaUnreserved = value
	case "ManaUnreservedPercent":
		ps.ManaUnreservedPercent = value
	case "ManaRegenRecovery":
		ps.ManaRegenRecovery = value
	case "ManaLeechGainRate":
		ps.ManaLeechGainRate = value
	case "EnergyShield":
		ps.EnergyShield = value
	case "EnergyShieldRecoveryCap":
		ps.EnergyShieldRecoveryCap = value
	case "Spec:EnergyShieldInc":
		ps.SpecEnergyShieldInc = value
	case "EnergyShieldRegenRecovery":
		ps.EnergyShieldRegenRecovery = value
	case "EnergyShieldLeechGainRate":
		ps.EnergyShieldLeechGainRate = value
	case "Ward":
		ps.Ward = value
	case "RageRegenRecovery":
		ps.RageRegenRecovery = value
	case "TotalBuildDegen":
		ps.TotalBuildDegen = value
	case "TotalNetRegen":
		ps.TotalNetRegen = value
	case "NetLifeRegen":
		ps.NetLifeRegen = value
	case "NetManaRegen":
		ps.NetManaRegen = value
	case "NetEnergyShieldRegen":
		ps.NetEnergyShieldRegen = value
	case "Evasion":
		ps.Evasion = value
	case "Spec:EvasionInc":
		ps.SpecEvasionInc = value
	case "MeleeEvadeChance":
		ps.MeleeEvadeChance = value
	case "ProjectileEvadeChance":
		ps.ProjectileEvadeChance = value
	case "Armour":
		ps.Armour = value
	case "Spec:ArmourInc":
		ps.SpecArmourInc = value
	case "PhysicalDamageReduction":
		ps.PhysicalDamageReduction = value
	case "EffectiveBlockChance":
		ps.EffectiveBlockChance = value
	case "EffectiveSpellBlockChance":
		ps.EffectiveSpellBlockChance = value
	case "AttackDodgeChance":
		ps.AttackDodgeChance = value
	case "SpellDodgeChance":
		ps.SpellDodgeChance = value
	case "EffectiveSpellSuppressionChance":
		ps.EffectiveSpellSuppressionChance = value
	case "FireResist":
		ps.FireResist = value
	case "FireResistOverCap":
		ps.FireResistOverCap = value
	case "ColdResist":
		ps.ColdResist = value
	case "ColdResistOverCap":
		ps.ColdResistOverCap = value
	case "LightningResist":
		ps.LightningResist = value
	case "LightningResistOverCap":
		ps.LightningResistOverCap = value
	case "ChaosResist":
		ps.ChaosResist = value
	case "ChaosResistOverCap":
		ps.ChaosResistOverCap = value
	case "EffectiveMovementSpeedMod":
		ps.EffectiveMovementSpeedMod = value
	case "FullDPS":
		ps.FullDPS = value
	case "FullDotDPS":
		ps.FullDotDPS = value
	case "PowerCharges":
		ps.PowerCharges = value
	case "PowerChargesMax":
		ps.PowerChargesMax = value
	case "FrenzyCharges":
		ps.FrenzyCharges = value
	case "FrenzyChargesMax":
		ps.FrenzyChargesMax = value
	case "EnduranceCharges":
		ps.EnduranceCharges = value
	case "EnduranceChargesMax":
		ps.EnduranceChargesMax = value
	}
}

func (b *Build) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "mainSocketGroup" {
			value, err := strconv.Atoi(attr.Value)
			if err != nil {
				return fmt.Errorf("invalid mainSocketGroup value %q: %w", attr.Value, err)
			}
			b.MainSocketGroup = value
			break
		}
	}

	for {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "PlayerStat" {
				var stat playerStatXML
				if err := d.DecodeElement(&stat, &se); err != nil {
					return err
				}
				b.PlayerStats.SetStat(stat.Stat, stat.Value)
			}
			if se.Name.Local == "FullDPSSkill" {
				var skill FullDpsSkill
				if err := d.DecodeElement(&skill, &se); err != nil {
					return err
				}
				b.FullDpsSkils = append(b.FullDpsSkils, skill)
			}
		}
		if end, ok := t.(xml.EndElement); ok && end.Name.Local == start.Name.Local {
			break
		}
	}
	return nil
}

// --- Skills Section Structs ---
type Skills struct {
	ActiveSkillSet      int        `xml:"activeSkillSet,attr"`
	SortGemsByDPS       string     `xml:"sortGemsByDPS,attr"`
	SortGemsByDPSField  string     `xml:"sortGemsByDPSField,attr"`
	ShowSupportGemTypes string     `xml:"showSupportGemTypes,attr"`
	ShowAltQualityGems  string     `xml:"showAltQualityGems,attr"`
	DefaultGemLevel     string     `xml:"defaultGemLevel,attr"`
	DefaultGemQuality   string     `xml:"defaultGemQuality,attr"`
	SkillSets           []SkillSet `xml:"SkillSet"`
}

type SkillSet struct {
	ID     int     `xml:"id,attr"`
	Skills []Skill `xml:"Skill"`
}

type Skill struct {
	Label                string `xml:"label,attr"`
	Slot                 string `xml:"slot,attr"`
	MainActiveSkillCalcs string `xml:"mainActiveSkillCalcs,attr"`
	MainActiveSkill      string `xml:"mainActiveSkill,attr"`
	IncludeInFullDPS     string `xml:"includeInFullDPS,attr"`
	Enabled              string `xml:"enabled,attr"`
	Gems                 []Gem  `xml:"Gem"`
}

type Gem struct {
	GemID         string `xml:"gemId,attr"`
	VariantID     string `xml:"variantId,attr"`
	EnableGlobal1 string `xml:"enableGlobal1,attr"`
	NameSpec      string `xml:"nameSpec,attr"`
	QualityID     string `xml:"qualityId,attr"`
	Enabled       string `xml:"enabled,attr"`
	EnableGlobal2 string `xml:"enableGlobal2,attr"`
	Quality       string `xml:"quality,attr"`
	SkillID       string `xml:"skillId,attr"`
	Count         string `xml:"count,attr"`
	Level         string `xml:"level,attr"`
	SkillPart     *int   `xml:"skillPart,attr"`
}

func DecodePoBExport(input string) (*PathOfBuilding, error) {
	input = strings.ReplaceAll(input, "-", "+")
	input = strings.ReplaceAll(input, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %w, input: %s", err, input)
	}
	b := bytes.NewReader(decoded)
	z, err := zlib.NewReader(b)
	if err != nil {
		return nil, fmt.Errorf("zlib decompress error: %w", err)
	}
	defer utils.Closer(z)()
	var pob PathOfBuilding
	if err := xml.NewDecoder(z).Decode(&pob); err != nil {
		return nil, fmt.Errorf("xml decode error: %w", err)
	}
	return &pob, nil
}
