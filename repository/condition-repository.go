package repository

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Operator string
type ItemField string

const (
	BASE_TYPE               ItemField = "BASE_TYPE"
	NAME                    ItemField = "NAME"
	ITEM_CLASS              ItemField = "ITEM_CLASS"
	ICON_NAME               ItemField = "ICON_NAME"
	TYPE_LINE               ItemField = "TYPE_LINE"
	QUALITY                 ItemField = "QUALITY"
	INTANGIBILITY           ItemField = "INTANGIBILITY"
	LEVEL                   ItemField = "LEVEL"
	RARITY                  ItemField = "RARITY"
	ILVL                    ItemField = "ILVL"
	FRAME_TYPE              ItemField = "FRAME_TYPE"
	TALISMAN_TIER           ItemField = "TALISMAN_TIER"
	MAP_TIER                ItemField = "MAP_TIER"
	MAP_QUANT               ItemField = "MAP_QUANT"
	MAP_RARITY              ItemField = "MAP_RARITY"
	MAP_PACK_SIZE           ItemField = "MAP_PACK_SIZE"
	HEIST_TARGET            ItemField = "HEIST_TARGET"
	HEIST_ROGUE_REQUIREMENT ItemField = "HEIST_ROGUE_REQUIREMENT"
	ENCHANTS                ItemField = "ENCHANT_MODS"
	EXPLICITS               ItemField = "EXPLICIT_MODS"
	IMPLICITS               ItemField = "IMPLICIT_MODS"
	CRAFTED_IMPLICIT_MODS   ItemField = "CRAFTED_IMPLICIT_MODS"
	CRAFTED_MODS            ItemField = "CRAFTED_MODS"
	FRACTURED_MODS          ItemField = "FRACTURED_MODS"
	VESTIGIAL_MODS          ItemField = "VESTIGIAL_MODS"
	INFLUENCES              ItemField = "INFLUENCES"
	MAX_LINKS               ItemField = "MAX_LINKS"
	SOCKETS                 ItemField = "SOCKETS" // as string like "RGBW"
	INCUBATOR_KILLS         ItemField = "INCUBATOR_KILLS"
	IS_CORRUPTED            ItemField = "IS_CORRUPTED"
	IS_VESTIGAL             ItemField = "IS_VESTIGAL"
	IS_VAAL                 ItemField = "IS_VAAL"
	IS_SPLIT                ItemField = "IS_SPLIT"
	IS_IDENTIFIED           ItemField = "IS_IDENTIFIED"
	IS_MIRRORED             ItemField = "IS_MIRRORED"
	SANCTUM_MODS            ItemField = "SANCTUM_AFFLICTIONS"
	TEMPLE_ROOMS            ItemField = "TEMPLE_ROOMS"
	TEMPLE_ROOMS_T3         ItemField = "TEMPLE_ROOMS_T3"
	RITUAL_BOSSES           ItemField = "RITUAL_VESSEL_BOSSES"
	RITUAL_MAP              ItemField = "RITUAL_VESSEL_MAP"
	FACETOR_LENS_EXP        ItemField = "FACETOR_LENS_EXP"
	MEMORY_STRANDS          ItemField = "MEMORY_STRANDS"
	IS_FOULBORN             ItemField = "IS_FOULBORN"
	FOULBORN_MODS           ItemField = "FOULBORN_MODS"
	GRAFT_SKILL_NAME        ItemField = "GRAFT_SKILL_NAME"
	GRAFT_SKILL_LEVEL       ItemField = "GRAFT_SKILL_LEVEL"
)

type FieldType string

const (
	String      FieldType = "string"
	Int         FieldType = "int"
	Bool        FieldType = "bool"
	StringArray FieldType = "string[]"
)

var FieldToType = map[ItemField]FieldType{
	BASE_TYPE:               String,
	NAME:                    String,
	ICON_NAME:               String,
	ITEM_CLASS:              String,
	TYPE_LINE:               String,
	QUALITY:                 Int,
	INTANGIBILITY:           Int,
	LEVEL:                   Int,
	RARITY:                  String,
	ILVL:                    Int,
	FRAME_TYPE:              String,
	TALISMAN_TIER:           Int,
	MAP_TIER:                Int,
	MAP_QUANT:               Int,
	MAP_RARITY:              Int,
	MAP_PACK_SIZE:           Int,
	HEIST_TARGET:            String,
	HEIST_ROGUE_REQUIREMENT: String,
	ENCHANTS:                StringArray,
	EXPLICITS:               StringArray,
	IMPLICITS:               StringArray,
	CRAFTED_IMPLICIT_MODS:   StringArray,
	CRAFTED_MODS:            StringArray,
	FRACTURED_MODS:          StringArray,
	INFLUENCES:              StringArray,
	VESTIGIAL_MODS:          StringArray,
	MAX_LINKS:               Int,
	SOCKETS:                 String,
	INCUBATOR_KILLS:         Int,
	FACETOR_LENS_EXP:        Int,
	IS_CORRUPTED:            Bool,
	IS_VAAL:                 Bool,
	IS_SPLIT:                Bool,
	IS_IDENTIFIED:           Bool,
	IS_MIRRORED:             Bool,
	IS_VESTIGAL:             Bool,
	SANCTUM_MODS:            StringArray,
	TEMPLE_ROOMS:            StringArray,
	TEMPLE_ROOMS_T3:         StringArray,
	RITUAL_BOSSES:           StringArray,
	RITUAL_MAP:              StringArray,
	MEMORY_STRANDS:          Int,
	IS_FOULBORN:             Bool,
	FOULBORN_MODS:           StringArray,
	GRAFT_SKILL_NAME:        String,
	GRAFT_SKILL_LEVEL:       Int,
}

var OperatorsForTypes = map[FieldType][]Operator{
	String:      {EQ, NEQ, IN, NOT_IN, MATCHES, CONTAINS, LENGTH_EQ, LENGTH_GT, LENGTH_LT, DOES_NOT_MATCH},
	Int:         {EQ, NEQ, GT, LT, IN, NOT_IN},
	Bool:        {EQ, NEQ},
	StringArray: {CONTAINS, CONTAINS_ALL, CONTAINS_MATCH, LENGTH_EQ, LENGTH_GT, LENGTH_LT, DOES_NOT_MATCH},
}

const (
	EQ             Operator = "EQ"
	NEQ            Operator = "NEQ"
	GT             Operator = "GT"
	LT             Operator = "LT"
	IN             Operator = "IN"
	NOT_IN         Operator = "NOT_IN"
	MATCHES        Operator = "MATCHES"
	CONTAINS       Operator = "CONTAINS"
	CONTAINS_ALL   Operator = "CONTAINS_ALL"
	CONTAINS_MATCH Operator = "CONTAINS_MATCH"
	LENGTH_EQ      Operator = "LENGTH_EQ"
	LENGTH_GT      Operator = "LENGTH_GT"
	LENGTH_LT      Operator = "LENGTH_LT"
	DOES_NOT_MATCH Operator = "DOES_NOT_MATCH"
)

type Condition struct {
	Field    ItemField `json:"field"`
	Operator Operator  `json:"operator"`
	Value    string    `json:"value"`
}
type Conditions []*Condition

func (c *Conditions) Scan(value any) error {
	if value == nil {
		*c = []*Condition{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: not a byte slice")
	}

	return json.Unmarshal(bytes, c)
}

func (c Conditions) Value() (driver.Value, error) {
	if c == nil {
		return json.Marshal([]Condition{})
	}
	return json.Marshal(c)
}
