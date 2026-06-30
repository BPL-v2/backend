package parser

import (
	clientModel "bpl/client"
	dbModel "bpl/repository"
	"bpl/utils"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type itemChecker func(item *clientModel.Item) int

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func BoolFieldGetter(field dbModel.ItemField) (func(item *clientModel.Item) bool, error) {
	switch field {
	case dbModel.IS_CORRUPTED:
		return func(item *clientModel.Item) bool {
			if item.Corrupted != nil {
				return *item.Corrupted
			}
			return false
		}, nil
	case dbModel.IS_VAAL:
		return func(item *clientModel.Item) bool {
			if item.Hybrid != nil && item.Hybrid.IsVaalGem != nil {
				return *item.Hybrid.IsVaalGem
			}
			return false
		}, nil
	case dbModel.IS_SPLIT:
		return func(item *clientModel.Item) bool {
			if item.Split != nil {
				return *item.Split
			}
			return false
		}, nil
	case dbModel.IS_IDENTIFIED:
		return func(item *clientModel.Item) bool {
			return item.Identified
		}, nil
	case dbModel.IS_FOULBORN:
		return func(item *clientModel.Item) bool {
			if item.Mutated != nil {
				return *item.Mutated
			}
			return false
		}, nil
	case dbModel.IS_MIRRORED:
		return func(item *clientModel.Item) bool {
			if item.Duplicated != nil {
				return *item.Duplicated
			}
			return false
		}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid boolean field", field)
	}
}

func StringFieldGetter(field dbModel.ItemField) (func(item *clientModel.Item) string, error) {
	switch field {
	case dbModel.BASE_TYPE:
		return func(item *clientModel.Item) string {
			return item.BaseType
		}, nil
	case dbModel.NAME:
		return func(item *clientModel.Item) string {
			return item.Name
		}, nil
	case dbModel.ITEM_CLASS:
		return func(item *clientModel.Item) string {
			return ItemClasses[item.BaseType]
		}, nil
	case dbModel.TYPE_LINE:
		return func(item *clientModel.Item) string {
			return item.TypeLine
		}, nil
	case dbModel.RARITY:
		return func(item *clientModel.Item) string {
			if item.Rarity != nil {
				return *item.Rarity
			}
			return ""
		}, nil
	case dbModel.SOCKETS:
		return func(item *clientModel.Item) string {
			if item.Sockets != nil {
				var socketString strings.Builder
				for _, socket := range *item.Sockets {
					if socket.SColour != nil {
						socketString.WriteString(*socket.SColour)
					}
				}
				return socketString.String()
			}
			return ""
		}, nil

	case dbModel.RITUAL_MAP:
		return func(item *clientModel.Item) string {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "From" {
						return property.Values[0].Name()
					}
				}
			}
			return ""
		}, nil
	case dbModel.ICON_NAME:
		return func(item *clientModel.Item) string {
			parts := strings.Split(item.Icon, "/")
			if len(parts) > 0 {
				return strings.Split(parts[len(parts)-1], ".")[0]
			}
			return ""
		}, nil
	case dbModel.HEIST_TARGET:
		return func(item *clientModel.Item) string {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Heist Target: {0} ({1})" {
						return property.Values[0].Name() + " (" + property.Values[1].Name() + ")"
					}
				}
			}
			return ""
		}, nil
	case dbModel.HEIST_ROGUE_REQUIREMENT:
		return func(item *clientModel.Item) string {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Requires {1} (Level {0})" {
						return property.Values[0].Name() + " (Level " + property.Values[1].Name() + ")"
					}
				}
			}
			return ""
		}, nil
	case dbModel.GRAFT_SKILL_NAME:
		return func(item *clientModel.Item) string {
			if item.SocketedItems == nil || len(*item.SocketedItems) == 0 {
				return ""
			}
			return (*item.SocketedItems)[0].BaseType
		}, nil
	case dbModel.FRAME_TYPE:
		return func(item *clientModel.Item) string {
			return item.FrameTypeId
		}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid string field", field)
	}
}

func StringArrayFieldGetter(field dbModel.ItemField) (func(item *clientModel.Item) []string, error) {
	switch field {
	case dbModel.FOULBORN_MODS:
		return func(item *clientModel.Item) []string {
			if item.MutatedMods != nil {
				return *item.MutatedMods
			}
			return []string{}
		}, nil
	case dbModel.ENCHANTS:
		return func(item *clientModel.Item) []string {
			if item.EnchantMods != nil {
				return *item.EnchantMods
			}
			return []string{}
		}, nil
	case dbModel.EXPLICITS:
		return func(item *clientModel.Item) []string {
			if item.ExplicitMods != nil {
				return *item.ExplicitMods
			}
			return []string{}
		}, nil
	case dbModel.IMPLICITS:
		return func(item *clientModel.Item) []string {
			if item.ImplicitMods != nil {
				return *item.ImplicitMods
			}
			return []string{}
		}, nil
	case dbModel.CRAFTED_MODS:
		return func(item *clientModel.Item) []string {
			if item.CraftedMods != nil {
				return *item.CraftedMods
			}
			return []string{}
		}, nil
	case dbModel.FRACTURED_MODS:
		return func(item *clientModel.Item) []string {
			if item.FracturedMods != nil {
				return *item.FracturedMods
			}
			return []string{}
		}, nil
	case dbModel.SANCTUM_MODS:
		return func(item *clientModel.Item) []string {
			mods := make([]string, 0)
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if slices.Contains([]string{"Minor Afflictions", "Major Afflictions", "Minor Boons", "Major Boons"}, property.Name) {
						for _, value := range property.Values {
							mods = append(mods, value.Name())
						}
					}
				}
			}
			return mods
		}, nil
	case dbModel.TEMPLE_ROOMS:
		return func(item *clientModel.Item) []string {
			rooms := make([]string, 0)
			if item.AdditionalProperties != nil {
				for _, property := range *item.AdditionalProperties {
					if property.Type != nil && *property.Type == 49 {
						// we can also only look for open rooms by requiring value.ID == 0
						for _, value := range property.Values {
							rooms = append(rooms, strings.Split(value.Name(), " (Tier")[0])
						}
					}
				}
			}
			return rooms
		}, nil
	case dbModel.TEMPLE_ROOMS_T3:
		return func(item *clientModel.Item) []string {
			rooms := make([]string, 0)
			if item.AdditionalProperties != nil {
				for _, property := range *item.AdditionalProperties {
					if property.Type != nil && *property.Type == 49 {
						// we can also only look for open rooms by requiring value.ID == 0
						for _, value := range property.Values {
							split := strings.Split(value.Name(), " (Tier ")
							if len(split) == 2 && split[1][0] == '3' {
								rooms = append(rooms, split[0])
							}
						}
					}
				}
			}
			return rooms
		}, nil
	case dbModel.RITUAL_BOSSES:
		return func(item *clientModel.Item) []string {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Monsters:\n{0}" {
						return strings.Split(property.Values[0].Name(), "\n")
					}
				}
			}
			return make([]string, 0)
		}, nil
	case dbModel.INFLUENCES:
		return func(item *clientModel.Item) []string {
			influences := make([]string, 0)
			if item.Influences != nil {
				for influence := range *item.Influences {
					influences = append(influences, influence)
				}
			}
			return influences
		}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid string array field", field)
	}
}

func IntFieldGetter(field dbModel.ItemField) (func(item *clientModel.Item) int, error) {
	switch field {
	case dbModel.ILVL:
		return func(item *clientModel.Item) int {
			return item.Ilvl
		}, nil
	case dbModel.TALISMAN_TIER:
		return func(item *clientModel.Item) int {
			if item.TalismanTier != nil {
				return *item.TalismanTier
			}
			return 0
		}, nil
	case dbModel.MAP_TIER:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Map Tier" {
						tier, err := strconv.Atoi(property.Values[0].Name())
						if err != nil {
							log.Printf("Error parsing map tier %s", property.Values[0].Name())
							return 0
						}
						return tier
					}
				}
			}
			return 0
		}, nil
	case dbModel.MAP_QUANT:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Item Quantity" {
						quantity, err := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(property.Values[0].Name(), "%", ""), "+", ""))
						if err != nil {
							log.Printf("Error parsing map quantity %s", property.Values[0].Name())
							return 0
						}
						return quantity
					}
				}
			}
			return 0
		}, nil
	case dbModel.MAP_RARITY:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Map Rarity" {
						rarity, err := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(property.Values[0].Name(), "%", ""), "+", ""))
						if err != nil {
							log.Printf("Error parsing map rarity %s", property.Values[0].Name())
							return 0
						}
						return rarity
					}
				}
			}
			return 0
		}, nil
	case dbModel.MAP_PACK_SIZE:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Monster Pack Size" {
						size, err := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(property.Values[0].Name(), "%", ""), "+", ""))
						if err != nil {
							log.Printf("Error parsing monster pack size %s", property.Values[0].Name())
							return 0
						}
						return size
					}
				}
			}
			return 0
		}, nil
	case dbModel.INCUBATOR_KILLS:
		return func(item *clientModel.Item) int {
			if item.IncubatedItem != nil {
				return item.IncubatedItem.Progress
			}
			return 0
		}, nil

	case dbModel.FACETOR_LENS_EXP:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Stored Experience: {0}" {
						exp, err := strconv.Atoi(property.Values[0].Name())
						if err != nil {
							log.Printf("Error parsing facetor lens exp %s", property.Values[0].Name())
							return 0
						}
						return exp
					}
				}
			}
			return 0
		}, nil
	case dbModel.QUALITY:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if strings.Contains(property.Name, "Quality") {
						quality, err := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(property.Values[0].Name(), "%", ""), "+", ""))
						if err != nil {
							log.Printf("Error parsing quality %s", property.Values[0].Name())
							return 0
						}
						return quality
					}
				}
			}
			return 0
		}, nil
	case dbModel.LEVEL:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Level" {
						level, err := strconv.Atoi(strings.ReplaceAll(property.Values[0].Name(), " (Max)", ""))
						if err != nil {
							log.Printf("Error parsing level %s", property.Values[0].Name())
							return 0
						}
						return level
					}
				}
			}
			return 0
		}, nil
	case dbModel.MEMORY_STRANDS:
		return func(item *clientModel.Item) int {
			if item.Properties != nil {
				for _, property := range *item.Properties {
					if property.Name == "Memory Strands" {
						strands, err := strconv.Atoi(property.Values[0].Name())
						if err != nil {
							log.Printf("Error parsing memory strands %s", property.Values[0].Name())
							return 0
						}
						return strands
					}
				}
			}
			return 0
		}, nil
	case dbModel.GRAFT_SKILL_LEVEL:
		return func(item *clientModel.Item) int {
			if item.SocketedItems == nil {
				return 0
			}
			for _, socketedItem := range *item.SocketedItems {
				if socketedItem.Properties == nil {
					continue
				}
				for _, property := range *socketedItem.Properties {
					if property.Name == "Level" {
						level, err := strconv.Atoi(strings.ReplaceAll(property.Values[0].Name(), " (Max)", ""))
						if err != nil {
							log.Printf("Error parsing graft skill level %s", property.Values[0].Name())
							return 0
						}
						return level
					}
				}
			}
			return 0
		}, nil
	case dbModel.FRAME_TYPE:
		return func(item *clientModel.Item) int {
			if item.FrameTypeId == "" {
				return 0
			}
			frameType, err := strconv.Atoi(item.FrameTypeId)
			if err != nil {
				return 0
			}
			return frameType
		}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid integer field", field)
	}
}

func StringToBool(value string) bool {
	if value == "true" || value == "True" || value == "1" {
		return true
	}
	return false
}

func BoolComparator(condition *dbModel.Condition) (itemChecker, error) {
	getter, err := BoolFieldGetter(condition.Field)
	if err != nil {
		return nil, err
	}
	value := StringToBool(condition.Value)
	switch condition.Operator {
	case dbModel.EQ:
		return func(item *clientModel.Item) int {
			return boolToInt(getter(item) == value)
		}, nil
	case dbModel.NEQ:
		return func(item *clientModel.Item) int {
			return boolToInt(getter(item) != value)
		}, nil
	default:
		return nil, fmt.Errorf("%s is an invalid operator for boolean field %s", condition.Operator, condition.Field)
	}
}

func IntComparator(condition *dbModel.Condition) (itemChecker, error) {
	getter, err := IntFieldGetter(condition.Field)
	if err != nil {
		return nil, err
	}
	var values = strings.Split(condition.Value, ",")
	intValues := make([]int, len(values))
	for i, v := range values {
		intValue, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		intValues[i] = intValue
	}
	intValue := intValues[0]

	switch condition.Operator {
	case dbModel.EQ:
		return func(item *clientModel.Item) int {
			return boolToInt(getter(item) == intValue)
		}, nil
	case dbModel.NEQ:
		return func(item *clientModel.Item) int {
			return boolToInt(getter(item) != intValue)
		}, nil
	case dbModel.GT:
		return func(item *clientModel.Item) int {
			return boolToInt(getter(item) > intValue)
		}, nil
	case dbModel.LT:
		return func(item *clientModel.Item) int {
			return boolToInt(getter(item) < intValue)
		}, nil
	case dbModel.IN:
		return func(item *clientModel.Item) int {
			return boolToInt(slices.Contains(intValues, getter(item)))
		}, nil
	case dbModel.NOT_IN:
		return func(item *clientModel.Item) int {
			return boolToInt(!slices.Contains(intValues, getter(item)))
		}, nil
	default:
		return nil, fmt.Errorf("%s is an invalid operator for integer field %s", condition.Operator, condition.Field)
	}
}

func StringComparator(condition *dbModel.Condition) (itemChecker, error) {
	getter, err := StringFieldGetter(condition.Field)
	if err != nil {
		return nil, err
	}

	switch condition.Operator {
	case dbModel.EQ:
		return func(item *clientModel.Item) int {
			return boolToInt(getter(item) == condition.Value)
		}, nil
	case dbModel.NEQ:
		return func(item *clientModel.Item) int {
			return boolToInt(getter(item) != condition.Value)
		}, nil
	case dbModel.IN:
		var values = strings.Split(condition.Value, ",")
		return func(item *clientModel.Item) int {
			return boolToInt(slices.Contains(values, getter(item)))
		}, nil
	case dbModel.NOT_IN:
		var values = strings.Split(condition.Value, ",")
		return func(item *clientModel.Item) int {
			return boolToInt(!slices.Contains(values, getter(item)))
		}, nil
	case dbModel.MATCHES:
		expression, err := regexp.Compile(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) int {
			return boolToInt(expression.MatchString(getter(item)))
		}, nil
	case dbModel.CONTAINS:
		return func(item *clientModel.Item) int {
			return boolToInt(strings.Contains(getter(item), condition.Value))
		}, nil
	case dbModel.LENGTH_EQ:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) int {
			return boolToInt(len(getter(item)) == length)
		}, nil
	case dbModel.LENGTH_GT:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) int {
			return boolToInt(len(getter(item)) > length)
		}, nil
	case dbModel.LENGTH_LT:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) int {
			return boolToInt(len(getter(item)) < length)
		}, nil
	case dbModel.DOES_NOT_MATCH:
		expression, err := regexp.Compile(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) int {
			return boolToInt(!expression.MatchString(getter(item)))
		}, nil
	default:
		return nil, fmt.Errorf("%s is an invalid operator for string field %s", condition.Operator, condition.Field)
	}
}

func StringArrayComparator(condition *dbModel.Condition) (itemChecker, error) {
	getter, err := StringArrayFieldGetter(condition.Field)
	if err != nil {
		return nil, err
	}
	switch condition.Operator {
	case dbModel.CONTAINS:
		return func(item *clientModel.Item) int {
			for _, actualValue := range getter(item) {
				if strings.Contains(actualValue, condition.Value) {
					return 1
				}
			}
			return 0
		}, nil
	case dbModel.CONTAINS_ALL:
		values := utils.Map(strings.Split(condition.Value, ","), func(s string) string {
			return strings.Trim(s, " ")
		})
		return func(item *clientModel.Item) int {
			for _, expectedValue := range values {
				found := false
				for _, actualValue := range getter(item) {
					if strings.Contains(actualValue, expectedValue) {
						found = true
						break
					}
				}
				if !found {
					return 0
				}
			}
			return 1
		}, nil
	case dbModel.CONTAINS_MATCH:
		expression, err := regexp.Compile(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) int {
			return boolToInt(slices.ContainsFunc(getter(item), expression.MatchString))
		}, nil
	case dbModel.LENGTH_EQ:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) int {
			return boolToInt(len(getter(item)) == length)
		}, nil
	case dbModel.LENGTH_GT:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) int {
			return boolToInt(len(getter(item)) > length)
		}, nil
	case dbModel.LENGTH_LT:
		length, err := strconv.Atoi(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) int {
			return boolToInt(len(getter(item)) < length)
		}, nil
	case dbModel.DOES_NOT_MATCH:
		expression, err := regexp.Compile(condition.Value)
		if err != nil {
			return nil, err
		}
		return func(item *clientModel.Item) int {
			return boolToInt(!slices.ContainsFunc(getter(item), expression.MatchString))
		}, nil
	default:
		return nil, fmt.Errorf("%s is an invalid operator for string array field %s", condition.Operator, condition.Field)
	}
}

func Comparator(condition *dbModel.Condition) (itemChecker, error) {
	switch dbModel.FieldToType[condition.Field] {
	case dbModel.Bool:
		return BoolComparator(condition)
	case dbModel.String:
		return StringComparator(condition)
	case dbModel.StringArray:
		return StringArrayComparator(condition)
	case dbModel.Int:
		return IntComparator(condition)
	default:
		return nil, fmt.Errorf("Comparator: invalid field type %s", condition.Field)
	}
}

func ComperatorFromConditions(conditions []*dbModel.Condition) (itemChecker, error) {
	if len(conditions) == 0 {
		return func(item *clientModel.Item) int {
			return 1
		}, nil
	}
	if len(conditions) == 1 {
		return Comparator(conditions[0])
	}
	checkers := make([]itemChecker, len(conditions))
	for i, condition := range conditions {
		checker, err := Comparator(condition)
		if err != nil {
			return nil, err
		}
		checkers[i] = checker
	}
	return func(item *clientModel.Item) int {
		for _, checker := range checkers {
			if checker(item) == 0 {
				return 0
			}
		}
		return 1
	}, nil
}

type DiscriminatorField int

const (
	BASE_TYPE  DiscriminatorField = iota
	NAME       DiscriminatorField = iota
	ITEM_CLASS DiscriminatorField = iota
	NONE       DiscriminatorField = iota
)

func toDiscriminatorField(field dbModel.ItemField) DiscriminatorField {
	switch field {
	case dbModel.BASE_TYPE:
		return BASE_TYPE
	case dbModel.NAME:
		return NAME
	case dbModel.ITEM_CLASS:
		return ITEM_CLASS
	default:
		return NONE
	}
}

type Discriminator struct {
	field DiscriminatorField
	value string
}

func GetDiscriminators(conditions []*dbModel.Condition) ([]*Discriminator, []*dbModel.Condition) {
	for i, condition := range conditions {
		if condition.Field == dbModel.BASE_TYPE || condition.Field == dbModel.NAME || condition.Field == dbModel.ITEM_CLASS {
			if condition.Operator == dbModel.EQ {
				discriminators := []*Discriminator{
					{field: toDiscriminatorField(condition.Field), value: condition.Value},
				}
				remainingConditions := append(conditions[:i], conditions[i+1:]...)
				return discriminators, remainingConditions
			}
			if condition.Operator == dbModel.IN {
				values := strings.Split(condition.Value, ",")
				discriminators := make([]*Discriminator, 0, len(values))
				for _, value := range values {
					discriminators = append(discriminators, &Discriminator{field: toDiscriminatorField(condition.Field), value: value})
				}
				remainingConditions := append(conditions[:i], conditions[i+1:]...)
				return discriminators, remainingConditions
			}
		}
	}
	return []*Discriminator{{field: NONE, value: ""}}, conditions
}

func ValidateConditions(conditions []*dbModel.Condition) error {
	for _, condition := range conditions {
		if _, err := Comparator(condition); err != nil {
			return err
		}
	}
	return nil
}

type ItemObjectiveChecker struct {
	Objective *dbModel.Objective
	Function  itemChecker
	ValidFrom *time.Time
	ValidTo   *time.Time
}

func (oc *ItemObjectiveChecker) Check(item *clientModel.Item) int {
	now := time.Now()
	if (oc.ValidFrom != nil && oc.ValidFrom.After(now)) || (oc.ValidTo != nil && oc.ValidTo.Before(now)) {
		return 0
	}
	return oc.Function(item)
}

type CheckResult struct {
	ObjectiveId int
	Number      int
}

type ItemChecker struct {
	Funcmap map[DiscriminatorField]map[string][]*ItemObjectiveChecker
}

func NewItemChecker(objectives []*dbModel.Objective, ignoreTime bool) (*ItemChecker, error) {
	funcMap := map[DiscriminatorField]map[string][]*ItemObjectiveChecker{
		BASE_TYPE:  make(map[string][]*ItemObjectiveChecker),
		NAME:       make(map[string][]*ItemObjectiveChecker),
		ITEM_CLASS: make(map[string][]*ItemObjectiveChecker),
		NONE:       make(map[string][]*ItemObjectiveChecker),
	}
	addChecker := func(field DiscriminatorField, key string, checker *ItemObjectiveChecker) {
		funcMap[field][key] = append(funcMap[field][key], checker)
	}
	for _, objective := range objectives {
		switch objective.ObjectiveType {
		case dbModel.ObjectiveTypeItem:
			if objective.Conditions == nil {
				continue
			}
			discriminators, remainingConditions := GetDiscriminators(objective.Conditions)
			conditionFn, err := ComperatorFromConditions(remainingConditions)
			if err != nil {
				fmt.Printf("Error getting comparator for objective %d: %s\n", objective.Id, err)
				continue
			}
			multiplier := 1
			switch objective.TrackedValue {
			case dbModel.TrackedValueFossilFuelHigh:
				multiplier = 10
			case dbModel.TrackedValueFossilFuelMid:
				multiplier = 2
			}
			fn := func(item *clientModel.Item) int {
				if conditionFn(item) == 0 {
					return 0
				}
				stackSize := 1
				if item.StackSize != nil {
					stackSize = *item.StackSize
				}
				return stackSize * multiplier
			}
			for _, discriminator := range discriminators {
				if _, ok := funcMap[discriminator.field]; !ok {
					return nil, fmt.Errorf("invalid discriminator field")
				}
				checker := &ItemObjectiveChecker{Objective: objective, Function: fn}
				if !ignoreTime {
					checker.ValidFrom = objective.ValidFrom
					checker.ValidTo = objective.ValidTo
				}
				addChecker(discriminator.field, discriminator.value, checker)
			}
		}
	}
	return &ItemChecker{Funcmap: funcMap}, nil
}

func (ic *ItemChecker) CheckForCompletions(item *clientModel.Item) []*CheckResult {
	results := make([]*CheckResult, 0)
	item.Name = strings.ReplaceAll(item.Name, "Foulborn ", "")
	if checkers, ok := ic.Funcmap[BASE_TYPE][item.BaseType]; ok {
		results = append(results, applyCheckers(checkers, item)...)
	}
	if checkers, ok := ic.Funcmap[NAME][item.Name]; ok {
		results = append(results, applyCheckers(checkers, item)...)
	}
	if checkers, ok := ic.Funcmap[ITEM_CLASS][ItemClasses[item.BaseType]]; ok {
		results = append(results, applyCheckers(checkers, item)...)
	}
	if checkers, ok := ic.Funcmap[NONE][""]; ok {
		results = append(results, applyCheckers(checkers, item)...)
	}
	return results
}

func applyCheckers(checkers []*ItemObjectiveChecker, item *clientModel.Item) []*CheckResult {
	results := make([]*CheckResult, 0)
	// sort out foiled items (SupporterFoil or numeric frame type 10)
	if item.FrameTypeId == "SupporterFoil" || item.FrameTypeId == "10" {
		return results
	}
	for _, checker := range checkers {
		n := checker.Check(item)
		if n == 0 {
			continue
		}
		if item.Split != nil && *item.Split && checker.Objective.RequiredAmount > 1 {
			// ignore split items for collections
			continue
		}
		results = append(results, &CheckResult{
			ObjectiveId: checker.Objective.Id,
			Number:      n,
		})
	}
	return results
}
