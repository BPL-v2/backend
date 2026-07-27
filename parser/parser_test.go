package parser

import (
	"bpl/client"
	clientModel "bpl/client"
	dbModel "bpl/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== Helpers ==========

func makeItem(opts ...func(*clientModel.Item)) *clientModel.Item {
	item := &clientModel.Item{Identified: true}
	for _, opt := range opts {
		opt(item)
	}
	return item
}

func withBaseType(bt string) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.BaseType = bt }
}
func withName(n string) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Name = n }
}
func withTypeLine(tl string) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.TypeLine = tl }
}
func withIlvl(lvl int) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Ilvl = lvl }
}
func withCorrupted(b bool) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Corrupted = new(b) }
}
func withIdentified(b bool) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Identified = b }
}
func withFrameTypeId(ft string) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.FrameTypeId = ft }
}
func withStackSize(s int) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.StackSize = new(s) }
}
func withRarity(r client.ItemRarity) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Rarity = new(r) }
}
func withIcon(icon string) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Icon = icon }
}
func withExplicitMods(mods ...string) func(*clientModel.Item) {
	itemMods := make([]clientModel.ItemMod, len(mods))
	for i, mod := range mods {
		itemMods[i] = clientModel.ItemMod{Description: mod}
	}
	return func(i *clientModel.Item) { i.ExplicitMods = &itemMods }
}
func withImplicitMods(mods ...string) func(*clientModel.Item) {
	itemMods := make([]clientModel.ItemMod, len(mods))
	for i, mod := range mods {
		itemMods[i] = clientModel.ItemMod{Description: mod}
	}
	return func(i *clientModel.Item) { i.ImplicitMods = &itemMods }
}
func withEnchantMods(mods ...string) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.EnchantMods = &mods }
}
func withCraftedMods(mods ...string) func(*clientModel.Item) {
	itemMods := make([]clientModel.ItemMod, len(mods))
	for i, mod := range mods {
		itemMods[i] = clientModel.ItemMod{Description: mod, Flags: &clientModel.ItemModFlags{Crafted: new(true)}}
	}
	return func(i *clientModel.Item) { i.ExplicitMods = &itemMods }
}
func withFracturedMods(mods ...string) func(*clientModel.Item) {
	itemMods := make([]clientModel.ItemMod, len(mods))
	for i, mod := range mods {
		itemMods[i] = clientModel.ItemMod{Description: mod, Flags: &clientModel.ItemModFlags{Fractured: new(true)}}
	}
	return func(i *clientModel.Item) { i.ExplicitMods = &itemMods }
}
func withInfluences(influences map[string]interface{}) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Influences = &influences }
}
func withMutated(b bool) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Mutated = new(b) }
}
func withMutatedMods(mods ...string) func(*clientModel.Item) {
	itemMods := make([]clientModel.ItemMod, len(mods))
	for i, mod := range mods {
		itemMods[i] = clientModel.ItemMod{Description: mod, Flags: &clientModel.ItemModFlags{Mutated: new(true)}}
	}
	return func(i *clientModel.Item) { i.ExplicitMods = &itemMods }
}
func withSplit(b bool) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Split = new(b) }
}
func withDuplicated(b bool) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Duplicated = new(b) }
}

func withIncubatedItem(name string, progress int) func(*clientModel.Item) {
	return func(i *clientModel.Item) {
		i.IncubatedItem = &clientModel.ItemIncubatedItem{Name: name, Progress: progress}
	}
}
func withProperties(props ...clientModel.ItemProperty) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Properties = &props }
}
func withSockets(sockets ...clientModel.ItemSocket) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Sockets = &sockets }
}
func withSocketedItems(items ...clientModel.Item) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.SocketedItems = &items }
}
func withHybrid(h *clientModel.ItemHybrid) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.Hybrid = h }
}
func withAdditionalProperties(props ...clientModel.ItemProperty) func(*clientModel.Item) {
	return func(i *clientModel.Item) { i.AdditionalProperties = &props }
}

func itemValue(name string) clientModel.ItemValue {
	return clientModel.ItemValue{name, 0}
}

func makeCondition(field dbModel.ItemField, op dbModel.Operator, value string) *dbModel.Condition {
	return &dbModel.Condition{Field: field, Operator: op, Value: value}
}

func makeObjective(id int, objType dbModel.ObjectiveType, conditions ...*dbModel.Condition) *dbModel.Objective {
	return &dbModel.Objective{
		Id:            id,
		Name:          "test-obj",
		ObjectiveType: objType,
		Conditions:    conditions,
	}
}

func makePlayerObjective(id int, numberField dbModel.TrackedValue) *dbModel.Objective {
	return &dbModel.Objective{
		Id:            id,
		Name:          "player-obj",
		ObjectiveType: dbModel.ObjectiveTypePlayer,
		TrackedValue:  numberField,
	}
}

func makeTeamObjective(id int, numberField dbModel.TrackedValue) *dbModel.Objective {
	return &dbModel.Objective{
		Id:            id,
		Name:          "team-obj",
		ObjectiveType: dbModel.ObjectiveTypeTeam,
		TrackedValue:  numberField,
	}
}

// ========== StringToBool ==========

func TestStringToBool(t *testing.T) {
	assert.True(t, StringToBool("true"))
	assert.True(t, StringToBool("True"))
	assert.True(t, StringToBool("1"))
	assert.False(t, StringToBool("false"))
	assert.False(t, StringToBool("0"))
	assert.False(t, StringToBool(""))
}

// ========== BoolFieldGetter ==========

func TestBoolFieldGetter(t *testing.T) {
	tests := []struct {
		name     string
		field    dbModel.ItemField
		item     *clientModel.Item
		expected bool
	}{
		{"corrupted true", dbModel.IS_CORRUPTED, makeItem(withCorrupted(true)), true},
		{"corrupted false", dbModel.IS_CORRUPTED, makeItem(withCorrupted(false)), false},
		{"corrupted nil", dbModel.IS_CORRUPTED, makeItem(), false},
		{"identified true", dbModel.IS_IDENTIFIED, makeItem(withIdentified(true)), true},
		{"identified false", dbModel.IS_IDENTIFIED, makeItem(withIdentified(false)), false},
		{"split true", dbModel.IS_SPLIT, makeItem(withSplit(true)), true},
		{"split nil", dbModel.IS_SPLIT, makeItem(), false},
		{"mirrored true", dbModel.IS_MIRRORED, makeItem(withDuplicated(true)), true},
		{"mirrored nil", dbModel.IS_MIRRORED, makeItem(), false},
		{"foulborn true", dbModel.IS_FOULBORN, makeItem(withMutated(true)), true},
		{"foulborn nil", dbModel.IS_FOULBORN, makeItem(), false},
		{"vaal with hybrid", dbModel.IS_VAAL, makeItem(withHybrid(&clientModel.ItemHybrid{IsVaalGem: new(true)})), true},
		{"vaal nil hybrid", dbModel.IS_VAAL, makeItem(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter, err := BoolFieldGetter(tt.field)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, getter(tt.item))
		})
	}

	t.Run("invalid field", func(t *testing.T) {
		_, err := BoolFieldGetter(dbModel.BASE_TYPE)
		assert.Error(t, err)
	})
}

// ========== StringFieldGetter ==========

func TestStringFieldGetter(t *testing.T) {
	tests := []struct {
		name     string
		field    dbModel.ItemField
		item     *clientModel.Item
		expected string
	}{
		{"base type", dbModel.BASE_TYPE, makeItem(withBaseType("Chaos Orb")), "Chaos Orb"},
		{"name", dbModel.NAME, makeItem(withName("Headhunter")), "Headhunter"},
		{"type line", dbModel.TYPE_LINE, makeItem(withTypeLine("Leather Belt")), "Leather Belt"},
		{"rarity set", dbModel.RARITY, makeItem(withRarity("Unique")), "Unique"},
		{"rarity nil", dbModel.RARITY, makeItem(), ""},
		{"icon name", dbModel.ICON_NAME, makeItem(withIcon("https://example.com/path/to/icon.png")), "icon"},
		{"icon name no path", dbModel.ICON_NAME, makeItem(withIcon("icon.png")), "icon"},
		{"sockets", dbModel.SOCKETS, makeItem(withSockets(
			clientModel.ItemSocket{SColour: new(client.ItemSocketSColourR)},
			clientModel.ItemSocket{SColour: new(client.ItemSocketSColourG)},
			clientModel.ItemSocket{SColour: new(client.ItemSocketSColourB)},
		)), "RGB"},
		{"sockets nil", dbModel.SOCKETS, makeItem(), ""},
		{"graft skill name with socketed items", dbModel.GRAFT_SKILL_NAME, makeItem(withSocketedItems(
			clientModel.Item{BaseType: "Fireball"},
		)), "Fireball"},
		{"graft skill name no socketed items", dbModel.GRAFT_SKILL_NAME, makeItem(), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter, err := StringFieldGetter(tt.field)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, getter(tt.item))
		})
	}

	t.Run("ritual map", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "From",
			Values: []clientModel.ItemValue{itemValue("Toxic Sewer")},
		}))
		getter, err := StringFieldGetter(dbModel.RITUAL_MAP)
		require.NoError(t, err)
		assert.Equal(t, "Toxic Sewer", getter(item))
	})

	t.Run("ritual map no properties", func(t *testing.T) {
		getter, err := StringFieldGetter(dbModel.RITUAL_MAP)
		require.NoError(t, err)
		assert.Equal(t, "", getter(makeItem()))
	})

	t.Run("invalid field", func(t *testing.T) {
		_, err := StringFieldGetter(dbModel.IS_CORRUPTED)
		assert.Error(t, err)
	})
}

// ========== IntFieldGetter ==========

func TestIntFieldGetter(t *testing.T) {
	tests := []struct {
		name     string
		field    dbModel.ItemField
		item     *clientModel.Item
		expected int
	}{
		{"ilvl", dbModel.ILVL, makeItem(withIlvl(83)), 83},
		{"frame type set", dbModel.FRAME_TYPE, makeItem(withFrameTypeId("3")), 3},
		{"frame type nil", dbModel.FRAME_TYPE, makeItem(), 0},
		{"talisman tier nil", dbModel.TALISMAN_TIER, makeItem(), 0},
		{"incubator kills", dbModel.INCUBATOR_KILLS, makeItem(withIncubatedItem("test", 500)), 500},
		{"incubator kills nil", dbModel.INCUBATOR_KILLS, makeItem(), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter, err := IntFieldGetter(tt.field)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, getter(tt.item))
		})
	}

	t.Run("map tier", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Map Tier",
			Values: []clientModel.ItemValue{itemValue("16")},
		}))
		getter, err := IntFieldGetter(dbModel.MAP_TIER)
		require.NoError(t, err)
		assert.Equal(t, 16, getter(item))
	})

	t.Run("map tier no properties", func(t *testing.T) {
		getter, err := IntFieldGetter(dbModel.MAP_TIER)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(makeItem()))
	})

	t.Run("quality", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Quality",
			Values: []clientModel.ItemValue{itemValue("+20%")},
		}))
		getter, err := IntFieldGetter(dbModel.QUALITY)
		require.NoError(t, err)
		assert.Equal(t, 20, getter(item))
	})

	t.Run("level with max suffix", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Level",
			Values: []clientModel.ItemValue{itemValue("21 (Max)")},
		}))
		getter, err := IntFieldGetter(dbModel.LEVEL)
		require.NoError(t, err)
		assert.Equal(t, 21, getter(item))
	})

	t.Run("map quantity", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Item Quantity",
			Values: []clientModel.ItemValue{itemValue("+120%")},
		}))
		getter, err := IntFieldGetter(dbModel.MAP_QUANT)
		require.NoError(t, err)
		assert.Equal(t, 120, getter(item))
	})

	t.Run("graft skill level", func(t *testing.T) {
		item := makeItem(withSocketedItems(clientModel.Item{
			Properties: &[]clientModel.ItemProperty{
				{Name: "Level", Values: []clientModel.ItemValue{itemValue("5")}},
			},
		}))
		getter, err := IntFieldGetter(dbModel.GRAFT_SKILL_LEVEL)
		require.NoError(t, err)
		assert.Equal(t, 5, getter(item))
	})

	t.Run("invalid field", func(t *testing.T) {
		_, err := IntFieldGetter(dbModel.BASE_TYPE)
		assert.Error(t, err)
	})
}

// ========== StringArrayFieldGetter ==========

func TestStringArrayFieldGetter(t *testing.T) {
	t.Run("explicit mods", func(t *testing.T) {
		item := makeItem(withExplicitMods("mod1", "mod2"))
		getter, err := StringArrayFieldGetter(dbModel.EXPLICITS)
		require.NoError(t, err)
		assert.Equal(t, []string{"mod1", "mod2"}, getter(item))
	})

	t.Run("explicit mods nil", func(t *testing.T) {
		getter, err := StringArrayFieldGetter(dbModel.EXPLICITS)
		require.NoError(t, err)
		assert.Equal(t, []string{}, getter(makeItem()))
	})

	t.Run("implicit mods", func(t *testing.T) {
		item := makeItem(withImplicitMods("impl1"))
		getter, err := StringArrayFieldGetter(dbModel.IMPLICITS)
		require.NoError(t, err)
		assert.Equal(t, []string{"impl1"}, getter(item))
	})

	t.Run("enchant mods", func(t *testing.T) {
		item := makeItem(withEnchantMods("enchant1"))
		getter, err := StringArrayFieldGetter(dbModel.ENCHANTS)
		require.NoError(t, err)
		assert.Equal(t, []string{"enchant1"}, getter(item))
	})

	t.Run("crafted mods", func(t *testing.T) {
		item := makeItem(withCraftedMods("crafted1"))
		getter, err := StringArrayFieldGetter(dbModel.CRAFTED_MODS)
		require.NoError(t, err)
		assert.Equal(t, []string{"crafted1"}, getter(item))
	})

	t.Run("fractured mods", func(t *testing.T) {
		item := makeItem(withFracturedMods("fractured1"))
		getter, err := StringArrayFieldGetter(dbModel.FRACTURED_MODS)
		require.NoError(t, err)
		assert.Equal(t, []string{"fractured1"}, getter(item))
	})

	t.Run("foulborn mods", func(t *testing.T) {
		item := makeItem(withMutatedMods("foul1", "foul2"))
		getter, err := StringArrayFieldGetter(dbModel.FOULBORN_MODS)
		require.NoError(t, err)
		assert.Equal(t, []string{"foul1", "foul2"}, getter(item))
	})

	t.Run("influences", func(t *testing.T) {
		item := makeItem(withInfluences(map[string]any{"shaper": true}))
		getter, err := StringArrayFieldGetter(dbModel.INFLUENCES)
		require.NoError(t, err)
		assert.Equal(t, []string{"shaper"}, getter(item))
	})

	t.Run("influences nil", func(t *testing.T) {
		getter, err := StringArrayFieldGetter(dbModel.INFLUENCES)
		require.NoError(t, err)
		assert.Equal(t, []string{}, getter(makeItem()))
	})

	t.Run("temple rooms", func(t *testing.T) {
		roomType := 49
		item := makeItem(withAdditionalProperties(clientModel.ItemProperty{
			Type:   &roomType,
			Values: []clientModel.ItemValue{itemValue("Corruption Chamber (Tier 3)")},
		}))
		getter, err := StringArrayFieldGetter(dbModel.TEMPLE_ROOMS)
		require.NoError(t, err)
		assert.Equal(t, []string{"Corruption Chamber"}, getter(item))
	})

	t.Run("temple rooms t3 only", func(t *testing.T) {
		roomType := 49
		item := makeItem(withAdditionalProperties(clientModel.ItemProperty{
			Type: &roomType,
			Values: []clientModel.ItemValue{
				itemValue("Corruption Chamber (Tier 3)"),
				itemValue("Trap Room (Tier 2)"),
			},
		}))
		getter, err := StringArrayFieldGetter(dbModel.TEMPLE_ROOMS_T3)
		require.NoError(t, err)
		assert.Equal(t, []string{"Corruption Chamber"}, getter(item))
	})

	t.Run("ritual bosses", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Monsters:\n{0}",
			Values: []clientModel.ItemValue{itemValue("Boss1\nBoss2")},
		}))
		getter, err := StringArrayFieldGetter(dbModel.RITUAL_BOSSES)
		require.NoError(t, err)
		assert.Equal(t, []string{"Boss1", "Boss2"}, getter(item))
	})

	t.Run("sanctum mods", func(t *testing.T) {
		item := makeItem(withProperties(
			clientModel.ItemProperty{
				Name:   "Minor Afflictions",
				Values: []clientModel.ItemValue{itemValue("Affliction1")},
			},
			clientModel.ItemProperty{
				Name:   "Major Boons",
				Values: []clientModel.ItemValue{itemValue("Boon1")},
			},
		))
		getter, err := StringArrayFieldGetter(dbModel.SANCTUM_MODS)
		require.NoError(t, err)
		result := getter(item)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "Affliction1")
		assert.Contains(t, result, "Boon1")
	})

	t.Run("invalid field", func(t *testing.T) {
		_, err := StringArrayFieldGetter(dbModel.BASE_TYPE)
		assert.Error(t, err)
	})
}

// ========== Comparators ==========

func TestBoolComparator(t *testing.T) {
	t.Run("EQ corrupted true", func(t *testing.T) {
		checker, err := BoolComparator(makeCondition(dbModel.IS_CORRUPTED, dbModel.EQ, "true"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withCorrupted(true))))
		assert.Zero(t, checker(makeItem(withCorrupted(false))))
		assert.Zero(t, checker(makeItem()))
	})

	t.Run("NEQ corrupted true", func(t *testing.T) {
		checker, err := BoolComparator(makeCondition(dbModel.IS_CORRUPTED, dbModel.NEQ, "true"))
		require.NoError(t, err)
		assert.Zero(t, checker(makeItem(withCorrupted(true))))
		assert.NotZero(t, checker(makeItem(withCorrupted(false))))
	})

	t.Run("invalid operator", func(t *testing.T) {
		_, err := BoolComparator(makeCondition(dbModel.IS_CORRUPTED, dbModel.GT, "true"))
		assert.Error(t, err)
	})

	t.Run("invalid field", func(t *testing.T) {
		_, err := BoolComparator(makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "true"))
		assert.Error(t, err)
	})
}

func TestIntComparator(t *testing.T) {
	t.Run("EQ", func(t *testing.T) {
		checker, err := IntComparator(makeCondition(dbModel.ILVL, dbModel.EQ, "83"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withIlvl(83))))
		assert.Zero(t, checker(makeItem(withIlvl(82))))
	})

	t.Run("NEQ", func(t *testing.T) {
		checker, err := IntComparator(makeCondition(dbModel.ILVL, dbModel.NEQ, "83"))
		require.NoError(t, err)
		assert.Zero(t, checker(makeItem(withIlvl(83))))
		assert.NotZero(t, checker(makeItem(withIlvl(82))))
	})

	t.Run("GT", func(t *testing.T) {
		checker, err := IntComparator(makeCondition(dbModel.ILVL, dbModel.GT, "80"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withIlvl(83))))
		assert.Zero(t, checker(makeItem(withIlvl(80))))
		assert.Zero(t, checker(makeItem(withIlvl(79))))
	})

	t.Run("LT", func(t *testing.T) {
		checker, err := IntComparator(makeCondition(dbModel.ILVL, dbModel.LT, "80"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withIlvl(79))))
		assert.Zero(t, checker(makeItem(withIlvl(80))))
	})

	t.Run("IN", func(t *testing.T) {
		checker, err := IntComparator(makeCondition(dbModel.ILVL, dbModel.IN, "80,83,86"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withIlvl(83))))
		assert.Zero(t, checker(makeItem(withIlvl(82))))
	})

	t.Run("NOT_IN", func(t *testing.T) {
		checker, err := IntComparator(makeCondition(dbModel.ILVL, dbModel.NOT_IN, "80,83"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withIlvl(82))))
		assert.Zero(t, checker(makeItem(withIlvl(83))))
	})

	t.Run("invalid value", func(t *testing.T) {
		_, err := IntComparator(makeCondition(dbModel.ILVL, dbModel.EQ, "abc"))
		assert.Error(t, err)
	})

	t.Run("invalid operator", func(t *testing.T) {
		_, err := IntComparator(makeCondition(dbModel.ILVL, dbModel.CONTAINS, "5"))
		assert.Error(t, err)
	})
}

func TestStringComparator(t *testing.T) {
	t.Run("EQ", func(t *testing.T) {
		checker, err := StringComparator(makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withBaseType("Chaos Orb"))))
		assert.Zero(t, checker(makeItem(withBaseType("Exalted Orb"))))
	})

	t.Run("NEQ", func(t *testing.T) {
		checker, err := StringComparator(makeCondition(dbModel.BASE_TYPE, dbModel.NEQ, "Chaos Orb"))
		require.NoError(t, err)
		assert.Zero(t, checker(makeItem(withBaseType("Chaos Orb"))))
		assert.NotZero(t, checker(makeItem(withBaseType("Exalted Orb"))))
	})

	t.Run("IN", func(t *testing.T) {
		checker, err := StringComparator(makeCondition(dbModel.BASE_TYPE, dbModel.IN, "Chaos Orb,Exalted Orb"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withBaseType("Chaos Orb"))))
		assert.NotZero(t, checker(makeItem(withBaseType("Exalted Orb"))))
		assert.Zero(t, checker(makeItem(withBaseType("Mirror of Kalandra"))))
	})

	t.Run("NOT_IN", func(t *testing.T) {
		checker, err := StringComparator(makeCondition(dbModel.BASE_TYPE, dbModel.NOT_IN, "Chaos Orb,Exalted Orb"))
		require.NoError(t, err)
		assert.Zero(t, checker(makeItem(withBaseType("Chaos Orb"))))
		assert.NotZero(t, checker(makeItem(withBaseType("Mirror of Kalandra"))))
	})

	t.Run("CONTAINS", func(t *testing.T) {
		checker, err := StringComparator(makeCondition(dbModel.BASE_TYPE, dbModel.CONTAINS, "Orb"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withBaseType("Chaos Orb"))))
		assert.Zero(t, checker(makeItem(withBaseType("Mirror of Kalandra"))))
	})

	t.Run("MATCHES regex", func(t *testing.T) {
		checker, err := StringComparator(makeCondition(dbModel.BASE_TYPE, dbModel.MATCHES, "^Chaos.*"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withBaseType("Chaos Orb"))))
		assert.Zero(t, checker(makeItem(withBaseType("Exalted Orb"))))
	})

	t.Run("DOES_NOT_MATCH regex", func(t *testing.T) {
		checker, err := StringComparator(makeCondition(dbModel.BASE_TYPE, dbModel.DOES_NOT_MATCH, "^Chaos.*"))
		require.NoError(t, err)
		assert.Zero(t, checker(makeItem(withBaseType("Chaos Orb"))))
		assert.NotZero(t, checker(makeItem(withBaseType("Exalted Orb"))))
	})

	t.Run("LENGTH_EQ", func(t *testing.T) {
		checker, err := StringComparator(makeCondition(dbModel.NAME, dbModel.LENGTH_EQ, "5"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withName("Abcde"))))
		assert.Zero(t, checker(makeItem(withName("Abcd"))))
	})

	t.Run("LENGTH_GT", func(t *testing.T) {
		checker, err := StringComparator(makeCondition(dbModel.NAME, dbModel.LENGTH_GT, "3"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withName("Abcd"))))
		assert.Zero(t, checker(makeItem(withName("Abc"))))
	})

	t.Run("LENGTH_LT", func(t *testing.T) {
		checker, err := StringComparator(makeCondition(dbModel.NAME, dbModel.LENGTH_LT, "3"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withName("Ab"))))
		assert.Zero(t, checker(makeItem(withName("Abc"))))
	})

	t.Run("invalid regex", func(t *testing.T) {
		_, err := StringComparator(makeCondition(dbModel.BASE_TYPE, dbModel.MATCHES, "[invalid"))
		assert.Error(t, err)
	})
}

func TestStringArrayComparator(t *testing.T) {
	t.Run("CONTAINS", func(t *testing.T) {
		checker, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.CONTAINS, "fire"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withExplicitMods("Adds 10 fire damage", "Adds 5 cold damage"))))
		assert.Zero(t, checker(makeItem(withExplicitMods("Adds 5 cold damage"))))
	})

	t.Run("CONTAINS_ALL", func(t *testing.T) {
		checker, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.CONTAINS_ALL, "fire,cold"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withExplicitMods("Adds fire damage", "Adds cold damage"))))
		assert.Zero(t, checker(makeItem(withExplicitMods("Adds fire damage"))))
	})

	t.Run("CONTAINS_MATCH regex", func(t *testing.T) {
		checker, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.CONTAINS_MATCH, "\\d+ to maximum"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withExplicitMods("+50 to maximum Life"))))
		assert.Zero(t, checker(makeItem(withExplicitMods("Adds fire damage"))))
	})

	t.Run("LENGTH_EQ", func(t *testing.T) {
		checker, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.LENGTH_EQ, "2"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withExplicitMods("mod1", "mod2"))))
		assert.Zero(t, checker(makeItem(withExplicitMods("mod1"))))
	})

	t.Run("LENGTH_GT", func(t *testing.T) {
		checker, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.LENGTH_GT, "1"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withExplicitMods("mod1", "mod2"))))
		assert.Zero(t, checker(makeItem(withExplicitMods("mod1"))))
	})

	t.Run("LENGTH_LT", func(t *testing.T) {
		checker, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.LENGTH_LT, "2"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withExplicitMods("mod1"))))
		assert.Zero(t, checker(makeItem(withExplicitMods("mod1", "mod2"))))
	})

	t.Run("DOES_NOT_MATCH", func(t *testing.T) {
		checker, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.DOES_NOT_MATCH, "fire"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withExplicitMods("cold damage"))))
		assert.Zero(t, checker(makeItem(withExplicitMods("fire damage"))))
	})

	t.Run("invalid operator", func(t *testing.T) {
		_, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.GT, "5"))
		assert.Error(t, err)
	})
}

// ========== Comparator (routing) ==========

func TestComparator(t *testing.T) {
	t.Run("routes to bool comparator", func(t *testing.T) {
		checker, err := Comparator(makeCondition(dbModel.IS_CORRUPTED, dbModel.EQ, "true"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withCorrupted(true))))
	})

	t.Run("routes to string comparator", func(t *testing.T) {
		checker, err := Comparator(makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withBaseType("Chaos Orb"))))
	})

	t.Run("routes to int comparator", func(t *testing.T) {
		checker, err := Comparator(makeCondition(dbModel.ILVL, dbModel.GT, "80"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withIlvl(85))))
	})

	t.Run("routes to string array comparator", func(t *testing.T) {
		checker, err := Comparator(makeCondition(dbModel.EXPLICITS, dbModel.CONTAINS, "fire"))
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withExplicitMods("fire damage"))))
	})

	t.Run("invalid field type", func(t *testing.T) {
		_, err := Comparator(makeCondition("INVALID_FIELD", dbModel.EQ, "x"))
		assert.Error(t, err)
	})
}

// ========== ComperatorFromConditions ==========

func TestComperatorFromConditions(t *testing.T) {
	t.Run("empty conditions matches all", func(t *testing.T) {
		checker, err := ComperatorFromConditions(nil)
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem()))
	})

	t.Run("single condition", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"),
		}
		checker, err := ComperatorFromConditions(conditions)
		require.NoError(t, err)
		assert.NotZero(t, checker(makeItem(withBaseType("Chaos Orb"))))
		assert.Zero(t, checker(makeItem(withBaseType("Exalted Orb"))))
	})

	t.Run("multiple conditions ANDed", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Leather Belt"),
			makeCondition(dbModel.IS_CORRUPTED, dbModel.EQ, "true"),
			makeCondition(dbModel.ILVL, dbModel.GT, "80"),
		}
		checker, err := ComperatorFromConditions(conditions)
		require.NoError(t, err)
		// all match
		assert.NotZero(t, checker(makeItem(withBaseType("Leather Belt"), withCorrupted(true), withIlvl(85))))
		// one fails
		assert.Zero(t, checker(makeItem(withBaseType("Leather Belt"), withCorrupted(false), withIlvl(85))))
	})
}

// ========== GetDiscriminators ==========

func TestGetDiscriminators(t *testing.T) {
	t.Run("base type EQ", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"),
			makeCondition(dbModel.IS_CORRUPTED, dbModel.EQ, "true"),
		}
		discs, remaining := GetDiscriminators(conditions)
		assert.Len(t, discs, 1)
		assert.Equal(t, BASE_TYPE, discs[0].field)
		assert.Equal(t, "Chaos Orb", discs[0].value)
		assert.Len(t, remaining, 1)
	})

	t.Run("name IN", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition(dbModel.NAME, dbModel.IN, "Headhunter,Mageblood"),
		}
		discs, remaining := GetDiscriminators(conditions)
		assert.Len(t, discs, 2)
		assert.Equal(t, NAME, discs[0].field)
		assert.Equal(t, "Headhunter", discs[0].value)
		assert.Equal(t, "Mageblood", discs[1].value)
		assert.Len(t, remaining, 0)
	})

	t.Run("item class EQ", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition(dbModel.ITEM_CLASS, dbModel.EQ, "Currency"),
		}
		discs, _ := GetDiscriminators(conditions)
		assert.Len(t, discs, 1)
		assert.Equal(t, ITEM_CLASS, discs[0].field)
	})

	t.Run("no discriminator", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition(dbModel.IS_CORRUPTED, dbModel.EQ, "true"),
		}
		discs, remaining := GetDiscriminators(conditions)
		assert.Len(t, discs, 1)
		assert.Equal(t, NONE, discs[0].field)
		assert.Len(t, remaining, 1)
	})

	t.Run("non-EQ/IN operator not used as discriminator", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition(dbModel.BASE_TYPE, dbModel.CONTAINS, "Orb"),
		}
		discs, _ := GetDiscriminators(conditions)
		assert.Equal(t, NONE, discs[0].field)
		assert.Equal(t, NONE, discs[0].field)
	})
}

// ========== ValidateConditions ==========

func TestValidateConditions(t *testing.T) {
	t.Run("valid conditions", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"),
			makeCondition(dbModel.ILVL, dbModel.GT, "80"),
		}
		assert.NoError(t, ValidateConditions(conditions))
	})

	t.Run("invalid condition", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition("INVALID_FIELD", dbModel.EQ, "x"),
		}
		assert.Error(t, ValidateConditions(conditions))
	})
}

// ========== NewItemChecker + CheckForCompletions ==========

func TestNewItemChecker(t *testing.T) {
	t.Run("creates checker for item objectives", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(1, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)
		assert.NotNil(t, checker)
	})

	t.Run("skips non-item objectives", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			{Id: 1, ObjectiveType: dbModel.ObjectiveTypePlayer, TrackedValue: dbModel.TrackedValueCharacterLevel},
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)
		assert.NotNil(t, checker)
	})

	t.Run("skips objectives with nil conditions", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			{Id: 1, ObjectiveType: dbModel.ObjectiveTypeItem, Conditions: nil},
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)
		assert.NotNil(t, checker)
	})
}

func TestItemCheckerCheckForCompletions(t *testing.T) {
	t.Run("matches by base type discriminator", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(10, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Chaos Orb")))
		require.Len(t, results, 1)
		assert.Equal(t, 10, results[0].ObjectiveId)
		assert.Equal(t, 1, results[0].Number)
	})

	t.Run("no match", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(10, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Exalted Orb")))
		assert.Len(t, results, 0)
	})

	t.Run("matches by name discriminator", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(20, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.NAME, dbModel.EQ, "Headhunter"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withName("Headhunter")))
		require.Len(t, results, 1)
		assert.Equal(t, 20, results[0].ObjectiveId)
	})

	t.Run("uses stack size as number", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(10, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Chaos Orb"), withStackSize(20)))
		require.Len(t, results, 1)
		assert.Equal(t, 20, results[0].Number)
	})

	t.Run("rejects foiled items (frameType 10)", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(10, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Chaos Orb"), withFrameTypeId("10")))
		assert.Len(t, results, 0)
	})

	t.Run("strips Foulborn prefix from name", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(30, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.NAME, dbModel.EQ, "Headhunter"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withName("Foulborn Headhunter")))
		require.Len(t, results, 1)
		assert.Equal(t, 30, results[0].ObjectiveId)
	})

	t.Run("NONE discriminator matches any base type", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(40, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.IS_CORRUPTED, dbModel.EQ, "true"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Whatever"), withCorrupted(true)))
		require.Len(t, results, 1)
		assert.Equal(t, 40, results[0].ObjectiveId)
	})

	t.Run("multiple conditions with discriminator", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(50, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Leather Belt"),
				makeCondition(dbModel.IS_CORRUPTED, dbModel.EQ, "true"),
				makeCondition(dbModel.ILVL, dbModel.GT, "80"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		// matches all conditions
		results := checker.CheckForCompletions(makeItem(withBaseType("Leather Belt"), withCorrupted(true), withIlvl(85)))
		require.Len(t, results, 1)

		// discriminator matches but condition fails
		results = checker.CheckForCompletions(makeItem(withBaseType("Leather Belt"), withCorrupted(false), withIlvl(85)))
		assert.Len(t, results, 0)
	})

	t.Run("multiple objectives can match same item", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(1, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"),
			),
			makeObjective(2, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.IS_IDENTIFIED, dbModel.EQ, "true"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Chaos Orb"), withIdentified(true)))
		assert.Len(t, results, 2)
	})

	t.Run("time-valid objective outside window rejected", func(t *testing.T) {
		past := time.Now().Add(-2 * time.Hour)
		pastEnd := time.Now().Add(-1 * time.Hour)
		objectives := []*dbModel.Objective{
			{
				Id:            60,
				ObjectiveType: dbModel.ObjectiveTypeItem,
				Conditions:    []*dbModel.Condition{makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb")},
				ValidFrom:     &past,
				ValidTo:       &pastEnd,
			},
		}
		checker, err := NewItemChecker(objectives, false) // ignoreTime=false
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Chaos Orb")))
		assert.Len(t, results, 0)
	})

	t.Run("time-valid objective inside window accepted", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		future := time.Now().Add(1 * time.Hour)
		objectives := []*dbModel.Objective{
			{
				Id:            61,
				ObjectiveType: dbModel.ObjectiveTypeItem,
				Conditions:    []*dbModel.Condition{makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb")},
				ValidFrom:     &past,
				ValidTo:       &future,
			},
		}
		checker, err := NewItemChecker(objectives, false)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Chaos Orb")))
		require.Len(t, results, 1)
		assert.Equal(t, 61, results[0].ObjectiveId)
	})

	t.Run("IN discriminator creates multiple lookups", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(70, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.BASE_TYPE, dbModel.IN, "Chaos Orb,Exalted Orb"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Chaos Orb")))
		require.Len(t, results, 1)
		assert.Equal(t, 70, results[0].ObjectiveId)

		results = checker.CheckForCompletions(makeItem(withBaseType("Exalted Orb")))
		require.Len(t, results, 1)
		assert.Equal(t, 70, results[0].ObjectiveId)

		results = checker.CheckForCompletions(makeItem(withBaseType("Mirror")))
		assert.Len(t, results, 0)
	})
}

// ========== Player Parser ==========

func TestMaxAtlasTreeNodes(t *testing.T) {
	t.Run("returns max across trees", func(t *testing.T) {
		p := &Player{
			AtlasPassiveTrees: []clientModel.LeagueAccountAtlasPassiveTree{
				{Hashes: []int{1, 2, 3}},
				{Hashes: []int{1, 2, 3, 4, 5}},
				{Hashes: []int{1}},
			},
		}
		assert.Equal(t, 5, p.MaxAtlasTreeNodes())
	})

	t.Run("empty trees", func(t *testing.T) {
		p := &Player{AtlasPassiveTrees: []clientModel.LeagueAccountAtlasPassiveTree{}}
		assert.Equal(t, 0, p.MaxAtlasTreeNodes())
	})
}

func TestCanMakeRequests(t *testing.T) {
	t.Run("valid token not expired few errors", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:            "valid-token",
			TokenExpiry:      time.Now().Add(1 * time.Hour),
			SuccessiveErrors: 0,
		}
		assert.True(t, p.CanMakeRequests())
	})

	t.Run("expired token", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "valid-token",
			TokenExpiry: time.Now().Add(-1 * time.Hour),
		}
		assert.False(t, p.CanMakeRequests())
	})

	t.Run("empty token", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "",
			TokenExpiry: time.Now().Add(1 * time.Hour),
		}
		assert.False(t, p.CanMakeRequests())
	})

	t.Run("too many errors", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:            "valid-token",
			TokenExpiry:      time.Now().Add(1 * time.Hour),
			SuccessiveErrors: 5,
		}
		assert.False(t, p.CanMakeRequests())
	})
}

func TestShouldUpdateCharacterName(t *testing.T) {
	timings := map[dbModel.TimingKey]time.Duration{
		dbModel.CharacterNameRefetchDelay: 5 * time.Minute,
	}

	t.Run("should update after delay", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
		}
		p.LastUpdateTimes.CharacterName = time.Now().Add(-10 * time.Minute)
		assert.True(t, p.ShouldUpdateCharacterName(timings))
	})

	t.Run("should not update before delay", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
		}
		p.LastUpdateTimes.CharacterName = time.Now().Add(-1 * time.Minute)
		assert.False(t, p.ShouldUpdateCharacterName(timings))
	})

	t.Run("should not update if cannot make requests", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "",
			TokenExpiry: time.Now().Add(1 * time.Hour),
		}
		assert.False(t, p.ShouldUpdateCharacterName(timings))
	})
}

func TestShouldUpdateCharacter(t *testing.T) {
	timings := map[dbModel.TimingKey]time.Duration{
		dbModel.CharacterRefetchDelay:          5 * time.Minute,
		dbModel.CharacterRefetchDelayImportant: 2 * time.Minute,
		dbModel.CharacterRefetchDelayInactive:  30 * time.Minute,
		dbModel.InactivityDuration:             1 * time.Hour,
	}

	t.Run("should not update with empty character name", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			New:         Player{Character: &clientModel.Character{Name: ""}},
		}
		assert.False(t, p.ShouldUpdateCharacter(timings))
	})

	t.Run("normal update after delay", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			LastActive:  time.Now(),
			New: Player{Character: &clientModel.Character{
				Name:  "TestChar",
				Level: 30,
			}},
		}
		p.LastUpdateTimes.Character = time.Now().Add(-10 * time.Minute)
		assert.True(t, p.ShouldUpdateCharacter(timings))
	})

	t.Run("inactive player uses longer delay", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			LastActive:  time.Now().Add(-2 * time.Hour), // inactive
			New: Player{Character: &clientModel.Character{
				Name:  "TestChar",
				Level: 30,
			}},
		}
		// Updated 10 min ago - within inactive delay of 30 min
		p.LastUpdateTimes.Character = time.Now().Add(-10 * time.Minute)
		assert.False(t, p.ShouldUpdateCharacter(timings))

		// Updated 35 min ago - past inactive delay
		p.LastUpdateTimes.Character = time.Now().Add(-35 * time.Minute)
		assert.True(t, p.ShouldUpdateCharacter(timings))
	})
}

func TestShouldUpdateLeagueAccount(t *testing.T) {
	timings := map[dbModel.TimingKey]time.Duration{
		dbModel.LeagueAccountRefetchDelay:          5 * time.Minute,
		dbModel.LeagueAccountRefetchDelayImportant: 2 * time.Minute,
		dbModel.LeagueAccountRefetchDelayInactive:  30 * time.Minute,
		dbModel.InactivityDuration:                 1 * time.Hour,
	}

	t.Run("should not update below level 55", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			New:         Player{Character: &clientModel.Character{Level: 50}},
		}
		assert.False(t, p.ShouldUpdateLeagueAccount(timings))
	})

	t.Run("important update for low atlas nodes", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			LastActive:  time.Now(),
			New: Player{
				Character:         &clientModel.Character{Level: 60},
				AtlasPassiveTrees: []clientModel.LeagueAccountAtlasPassiveTree{{Hashes: make([]int, 50)}},
			},
		}
		// Updated 3 min ago - past important delay of 2 min
		p.LastUpdateTimes.LeagueAccount = time.Now().Add(-3 * time.Minute)
		assert.True(t, p.ShouldUpdateLeagueAccount(timings))
	})

	t.Run("normal update for high atlas nodes", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			LastActive:  time.Now(),
			New: Player{
				Character:         &clientModel.Character{Level: 60},
				AtlasPassiveTrees: []clientModel.LeagueAccountAtlasPassiveTree{{Hashes: make([]int, 150)}},
			},
		}
		// Updated 3 min ago - within normal delay of 5 min
		p.LastUpdateTimes.LeagueAccount = time.Now().Add(-3 * time.Minute)
		assert.False(t, p.ShouldUpdateLeagueAccount(timings))

		// Updated 6 min ago - past normal delay
		p.LastUpdateTimes.LeagueAccount = time.Now().Add(-6 * time.Minute)
		assert.True(t, p.ShouldUpdateLeagueAccount(timings))
	})
}

// ========== GetPlayerChecker ==========

func TestGetPlayerChecker(t *testing.T) {
	t.Run("rejects non-player objective", func(t *testing.T) {
		obj := makeObjective(1, dbModel.ObjectiveTypeItem)
		_, err := GetPlayerChecker(obj)
		assert.Error(t, err)
	})

	t.Run("player level", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueCharacterLevel)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)
		assert.Equal(t, 90, checker(&Player{Character: &clientModel.Character{Level: 90}}))
		assert.Equal(t, 0, checker(&Player{Character: nil}))
	})

	t.Run("delve depth", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueDelveDepth)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)
		assert.Equal(t, 200, checker(&Player{DelveDepth: 200}))
	})

	t.Run("delve depth past 100", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueDelveDepthAfter100)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)
		assert.Equal(t, 50, checker(&Player{DelveDepth: 150}))
		assert.Equal(t, 0, checker(&Player{DelveDepth: 80}))
	})

	t.Run("pantheon", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueTeamPlayersWithPantheonUnlocked)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		assert.Equal(t, 2, checker(&Player{Character: &clientModel.Character{
			Passives: &clientModel.CharacterPassives{
				PantheonMajor: new(clientModel.Arakaali),
				PantheonMinor: new(clientModel.Gruthkul),
			},
		}}))
		assert.Equal(t, 1, checker(&Player{Character: &clientModel.Character{
			Passives: &clientModel.CharacterPassives{PantheonMajor: new(clientModel.Lunaris)},
		}}))
		assert.Equal(t, 0, checker(&Player{Character: &clientModel.Character{}}))
		assert.Equal(t, 0, checker(&Player{}))
	})

	t.Run("fully ascended", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueTeamPlayersWithAllLabsCompleted)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		// Needs GetAscendancyPoints >= 8 - using character with Passives.Hashes
		// Since GetAscendancyPoints depends on static ascendancy node sets, we test with nil character
		assert.Equal(t, 0, checker(&Player{Character: nil}))
		assert.Equal(t, 0, checker(&Player{Character: &clientModel.Character{}}))
	})

	t.Run("PoB stats return 0 with nil PoB", func(t *testing.T) {
		fields := []dbModel.TrackedValue{
			dbModel.TrackedValueEvasion, dbModel.TrackedValueArmour, dbModel.TrackedValueEnergyShield,
			dbModel.TrackedValueMana, dbModel.TrackedValueHP, dbModel.TrackedValueEHP,
			dbModel.TrackedValuePhysicalMaxHit, dbModel.TrackedValueElementalMaxHit,
			dbModel.TrackedValueMovementSpeedBonus, dbModel.TrackedValueFullDPS,
		}
		for _, field := range fields {
			obj := makePlayerObjective(1, field)
			checker, err := GetPlayerChecker(obj)
			require.NoError(t, err)
			assert.Equal(t, 0, checker(&Player{PoB: nil}), "field %s should return 0 for nil PoB", field)
		}
	})

	t.Run("PoB evasion", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueEvasion)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)
		assert.Equal(t, 5000, checker(&Player{PoB: &dbModel.CharacterPob{Evasion: 5000}}))
	})

	t.Run("PoB movement speed", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueMovementSpeedBonus)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)
		// MovementSpeed 130 means +30% inc movement speed (130 - 100)
		assert.Equal(t, 30, checker(&Player{PoB: &dbModel.CharacterPob{MovementSpeed: 130}}))
	})

	t.Run("influence equipped", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueInfluencedItemCount)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		influences := map[string]any{"shaper": true}
		equipment := []clientModel.Item{
			{Influences: &influences},
			{},
		}
		assert.Equal(t, 1, checker(&Player{Character: &clientModel.Character{Equipment: &equipment}}))
		assert.Equal(t, 0, checker(&Player{Character: nil}))
	})

	t.Run("foulborn equipped", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueFoulbornItemCount)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		equipment := []clientModel.Item{
			{Mutated: new(true)},
			{Mutated: new(true)},
			{},
		}
		assert.Equal(t, 2, checker(&Player{Character: &clientModel.Character{Equipment: &equipment}}))
	})

	t.Run("gems equipped", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueSocketedGemCount)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		socketed := []clientModel.Item{{}, {}} // 2 gems (no AbyssJewel flag)
		equipment := []clientModel.Item{
			{SocketedItems: &socketed},
		}
		assert.Equal(t, 2, checker(&Player{Character: &clientModel.Character{Equipment: &equipment}}))
		assert.Equal(t, 0, checker(&Player{Character: nil}))
	})

	t.Run("gems equipped excludes abyss jewels", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueSocketedGemCount)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		socketed := []clientModel.Item{
			{},                      // gem
			{AbyssJewel: new(true)}, // abyss jewel, not a gem
		}
		equipment := []clientModel.Item{
			{SocketedItems: &socketed},
		}
		assert.Equal(t, 1, checker(&Player{Character: &clientModel.Character{Equipment: &equipment}}))
	})

	t.Run("corrupted items equipped", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueCorruptedItemCount)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		equipment := []clientModel.Item{
			{Corrupted: new(true)},
			{Corrupted: new(true)},
			{},
		}
		assert.Equal(t, 2, checker(&Player{Character: &clientModel.Character{Equipment: &equipment}}))
	})

	t.Run("jewels with implicits", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueJewelsWithImplicitsCount)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		implicits := []clientModel.ItemMod{{Description: "implicit1"}}
		jewels := []clientModel.Item{
			{BaseType: "Crimson Jewel", ImplicitMods: &implicits},
			{BaseType: "Crimson Jewel"},
		}
		assert.Equal(t, 1, checker(&Player{Character: &clientModel.Character{Jewels: &jewels}}))
	})

	t.Run("atlas points subtracts node 65225", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueAtlasPoints)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		p := &Player{AtlasPassiveTrees: []clientModel.LeagueAccountAtlasPassiveTree{
			{Hashes: []int{1, 2, 3, 65225}}, // 4 nodes - 20 for 65225 = -16, max(0, -16) = 0
		}}
		assert.Equal(t, 0, checker(p))

		// Multiple trees - takes max across them
		p2 := &Player{AtlasPassiveTrees: []clientModel.LeagueAccountAtlasPassiveTree{
			{Hashes: []int{1, 2, 3, 65225}},          // -16
			{Hashes: append(make([]int, 25), 65225)}, // 26 - 20 = 6
		}}
		assert.Equal(t, 6, checker(p2))
	})

	t.Run("enchanted items equipped", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueEnchantedItemCount)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		enchants := []string{"enchant1"}
		equipment := []clientModel.Item{
			{EnchantMods: &enchants},
			{},
		}
		assert.Equal(t, 1, checker(&Player{Character: &clientModel.Character{Equipment: &equipment}}))
		assert.Equal(t, 0, checker(&Player{Character: nil}))
	})

	t.Run("has rare ascendancy past 90", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueHasRareAscendancyPast90)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		assert.Equal(t, 1, checker(&Player{Character: &clientModel.Character{Level: 95, Class: "Assassin"}}))
		assert.Equal(t, 0, checker(&Player{Character: &clientModel.Character{Level: 95, Class: "Deadeye"}}))
		assert.Equal(t, 0, checker(&Player{Character: &clientModel.Character{Level: 85, Class: "Assassin"}}))
		assert.Equal(t, 0, checker(&Player{Character: nil}))
	})

	t.Run("bloodline ascendancy", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueBloodlineAscendancyUnlocked)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		assert.Equal(t, 1, checker(&Player{Character: &clientModel.Character{
			Passives: &clientModel.CharacterPassives{AlternateAscendancy: new("Warbringer")},
		}}))
		assert.Equal(t, 0, checker(&Player{Character: &clientModel.Character{}}))
		assert.Equal(t, 0, checker(&Player{Character: nil}))
	})

	t.Run("unsupported number field", func(t *testing.T) {
		obj := makePlayerObjective(1, "INVALID_FIELD")
		_, err := GetPlayerChecker(obj)
		assert.Error(t, err)
	})
}

func TestPlayerScore(t *testing.T) {
	obj := makePlayerObjective(1, dbModel.TrackedValuePersonalObjectiveScore)
	checker, err := GetPlayerChecker(obj)
	require.NoError(t, err)

	t.Run("nil character", func(t *testing.T) {
		assert.Equal(t, 0, checker(&Player{Character: nil}))
	})

	t.Run("low level", func(t *testing.T) {
		assert.Equal(t, 0, checker(&Player{Character: &clientModel.Character{Level: 30}}))
	})

	t.Run("level 40 gives 1 point", func(t *testing.T) {
		assert.Equal(t, 1, checker(&Player{Character: &clientModel.Character{Level: 40}}))
	})

	t.Run("level 60 gives 2 points", func(t *testing.T) {
		assert.Equal(t, 2, checker(&Player{Character: &clientModel.Character{Level: 60}}))
	})

	t.Run("level 80 gives 3 points", func(t *testing.T) {
		assert.Equal(t, 3, checker(&Player{Character: &clientModel.Character{Level: 80}}))
	})

	t.Run("level 90 gives 6 points", func(t *testing.T) {
		assert.Equal(t, 6, checker(&Player{Character: &clientModel.Character{Level: 90}}))
	})

	t.Run("level 90 without ascendancy or pob gives 6", func(t *testing.T) {
		p := &Player{
			Character: &clientModel.Character{Level: 90},
		}
		assert.Equal(t, 6, checker(p))
	})
}

// ========== GetTeamChecker ==========

func TestGetTeamChecker(t *testing.T) {
	t.Run("rejects non-team objective", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueCharacterLevel)
		_, err := GetTeamChecker(obj)
		assert.Error(t, err)
	})

	t.Run("sums player values", func(t *testing.T) {
		obj := makeTeamObjective(1, dbModel.TrackedValueCharacterLevel)
		checker, err := GetTeamChecker(obj)
		require.NoError(t, err)

		players := []*Player{
			{Character: &clientModel.Character{Level: 90, Name: "p1"}},
			{Character: &clientModel.Character{Level: 85, Name: "p2"}},
		}
		assert.Equal(t, 175, checker(players))
	})
}

// ========== NewPlayerChecker + CheckForCompletions ==========

func TestNewPlayerChecker(t *testing.T) {
	t.Run("skips non-player objectives", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeTeamObjective(1, dbModel.TrackedValueCharacterLevel),
			makePlayerObjective(2, dbModel.TrackedValueCharacterLevel),
		}
		checker, err := NewPlayerChecker(objectives)
		require.NoError(t, err)
		assert.Len(t, *checker, 1)
	})

	t.Run("detects changes", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makePlayerObjective(1, dbModel.TrackedValueCharacterLevel),
		}
		checker, err := NewPlayerChecker(objectives)
		require.NoError(t, err)

		update := &PlayerUpdate{
			Old: Player{Character: &clientModel.Character{Level: 80}},
			New: Player{Character: &clientModel.Character{Level: 85}},
		}
		results := checker.CheckForCompletions(update)
		require.Len(t, results, 1)
		assert.Equal(t, 1, results[0].ObjectiveId)
		assert.Equal(t, 85, results[0].Number)
	})

	t.Run("no change produces no results", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makePlayerObjective(1, dbModel.TrackedValueCharacterLevel),
		}
		checker, err := NewPlayerChecker(objectives)
		require.NoError(t, err)

		update := &PlayerUpdate{
			Old: Player{Character: &clientModel.Character{Level: 80}},
			New: Player{Character: &clientModel.Character{Level: 80}},
		}
		results := checker.CheckForCompletions(update)
		assert.Len(t, results, 0)
	})
}

// ========== NewTeamChecker + CheckForCompletions ==========

func TestNewTeamChecker(t *testing.T) {
	t.Run("skips non-team objectives", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makePlayerObjective(1, dbModel.TrackedValueCharacterLevel),
			makeTeamObjective(2, dbModel.TrackedValueCharacterLevel),
		}
		checker, err := NewTeamChecker(objectives)
		require.NoError(t, err)
		assert.Len(t, *checker, 1)
	})

	t.Run("detects team-level changes", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeTeamObjective(1, dbModel.TrackedValueCharacterLevel),
		}
		checker, err := NewTeamChecker(objectives)
		require.NoError(t, err)

		updates := []*PlayerUpdate{
			{
				Old: Player{Character: &clientModel.Character{Level: 80, Name: "p1"}},
				New: Player{Character: &clientModel.Character{Level: 85, Name: "p1"}},
			},
			{
				Old: Player{Character: &clientModel.Character{Level: 70, Name: "p2"}},
				New: Player{Character: &clientModel.Character{Level: 70, Name: "p2"}},
			},
		}
		results := checker.CheckForCompletions(updates)
		require.Len(t, results, 1)
		assert.Equal(t, 1, results[0].ObjectiveId)
		assert.Equal(t, 155, results[0].Number) // 85 + 70
	})

	t.Run("no team change produces no results", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeTeamObjective(1, dbModel.TrackedValueCharacterLevel),
		}
		checker, err := NewTeamChecker(objectives)
		require.NoError(t, err)

		updates := []*PlayerUpdate{
			{
				Old: Player{Character: &clientModel.Character{Level: 80, Name: "p1"}},
				New: Player{Character: &clientModel.Character{Level: 80, Name: "p1"}},
			},
		}
		results := checker.CheckForCompletions(updates)
		assert.Len(t, results, 0)
	})
}

// ========== Additional StringFieldGetter coverage ==========

func TestStringFieldGetterAdditional(t *testing.T) {
	t.Run("item class", func(t *testing.T) {
		getter, err := StringFieldGetter(dbModel.ITEM_CLASS)
		require.NoError(t, err)
		assert.Equal(t, "StackableCurrency", getter(makeItem(withBaseType("Chaos Orb"))))
		assert.Equal(t, "", getter(makeItem(withBaseType("UnknownItem"))))
	})

	t.Run("heist target", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Heist Target: {0} ({1})",
			Values: []clientModel.ItemValue{itemValue("Replicas"), itemValue("Unique")},
		}))
		getter, err := StringFieldGetter(dbModel.HEIST_TARGET)
		require.NoError(t, err)
		assert.Equal(t, "Replicas (Unique)", getter(item))
	})

	t.Run("heist target no properties", func(t *testing.T) {
		getter, err := StringFieldGetter(dbModel.HEIST_TARGET)
		require.NoError(t, err)
		assert.Equal(t, "", getter(makeItem()))
	})

	t.Run("heist rogue requirement", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Requires {1} (Level {0})",
			Values: []clientModel.ItemValue{itemValue("Karst"), itemValue("5")},
		}))
		getter, err := StringFieldGetter(dbModel.HEIST_ROGUE_REQUIREMENT)
		require.NoError(t, err)
		assert.Equal(t, "Karst (Level 5)", getter(item))
	})

	t.Run("heist rogue requirement no properties", func(t *testing.T) {
		getter, err := StringFieldGetter(dbModel.HEIST_ROGUE_REQUIREMENT)
		require.NoError(t, err)
		assert.Equal(t, "", getter(makeItem()))
	})

	t.Run("sockets with nil colour", func(t *testing.T) {
		getter, err := StringFieldGetter(dbModel.SOCKETS)
		require.NoError(t, err)
		item := makeItem(withSockets(
			clientModel.ItemSocket{SColour: new(clientModel.ItemSocketSColourR)},
			clientModel.ItemSocket{SColour: nil},
			clientModel.ItemSocket{SColour: new(clientModel.ItemSocketSColourB)},
		))
		assert.Equal(t, "RB", getter(item))
	})

	t.Run("graft skill name empty socketed items", func(t *testing.T) {
		items := []clientModel.Item{}
		getter, err := StringFieldGetter(dbModel.GRAFT_SKILL_NAME)
		require.NoError(t, err)
		item := &clientModel.Item{SocketedItems: &items}
		assert.Equal(t, "", getter(item))
	})

	t.Run("ritual map no matching property", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "SomethingElse",
			Values: []clientModel.ItemValue{itemValue("value")},
		}))
		getter, err := StringFieldGetter(dbModel.RITUAL_MAP)
		require.NoError(t, err)
		assert.Equal(t, "", getter(item))
	})

	t.Run("heist target no matching property", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "SomethingElse",
			Values: []clientModel.ItemValue{itemValue("value")},
		}))
		getter, err := StringFieldGetter(dbModel.HEIST_TARGET)
		require.NoError(t, err)
		assert.Equal(t, "", getter(item))
	})

	t.Run("heist rogue no matching property", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "SomethingElse",
			Values: []clientModel.ItemValue{itemValue("value")},
		}))
		getter, err := StringFieldGetter(dbModel.HEIST_ROGUE_REQUIREMENT)
		require.NoError(t, err)
		assert.Equal(t, "", getter(item))
	})
}

// ========== Additional IntFieldGetter coverage ==========

func TestIntFieldGetterAdditional(t *testing.T) {
	t.Run("map rarity", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Map Rarity",
			Values: []clientModel.ItemValue{itemValue("+80%")},
		}))
		getter, err := IntFieldGetter(dbModel.MAP_RARITY)
		require.NoError(t, err)
		assert.Equal(t, 80, getter(item))
	})

	t.Run("map rarity no properties", func(t *testing.T) {
		getter, err := IntFieldGetter(dbModel.MAP_RARITY)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(makeItem()))
	})

	t.Run("map rarity parse error", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Map Rarity",
			Values: []clientModel.ItemValue{itemValue("abc%")},
		}))
		getter, err := IntFieldGetter(dbModel.MAP_RARITY)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})

	t.Run("map pack size", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Monster Pack Size",
			Values: []clientModel.ItemValue{itemValue("+30%")},
		}))
		getter, err := IntFieldGetter(dbModel.MAP_PACK_SIZE)
		require.NoError(t, err)
		assert.Equal(t, 30, getter(item))
	})

	t.Run("map pack size no properties", func(t *testing.T) {
		getter, err := IntFieldGetter(dbModel.MAP_PACK_SIZE)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(makeItem()))
	})

	t.Run("map pack size parse error", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Monster Pack Size",
			Values: []clientModel.ItemValue{itemValue("bad")},
		}))
		getter, err := IntFieldGetter(dbModel.MAP_PACK_SIZE)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})

	t.Run("facetor lens exp", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Stored Experience: {0}",
			Values: []clientModel.ItemValue{itemValue("1000000")},
		}))
		getter, err := IntFieldGetter(dbModel.FACETOR_LENS_EXP)
		require.NoError(t, err)
		assert.Equal(t, 1000000, getter(item))
	})

	t.Run("facetor lens exp no properties", func(t *testing.T) {
		getter, err := IntFieldGetter(dbModel.FACETOR_LENS_EXP)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(makeItem()))
	})

	t.Run("facetor lens exp parse error", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Stored Experience: {0}",
			Values: []clientModel.ItemValue{itemValue("notanumber")},
		}))
		getter, err := IntFieldGetter(dbModel.FACETOR_LENS_EXP)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})

	t.Run("memory strands", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Memory Strands",
			Values: []clientModel.ItemValue{itemValue("42")},
		}))
		getter, err := IntFieldGetter(dbModel.MEMORY_STRANDS)
		require.NoError(t, err)
		assert.Equal(t, 42, getter(item))
	})

	t.Run("memory strands no properties", func(t *testing.T) {
		getter, err := IntFieldGetter(dbModel.MEMORY_STRANDS)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(makeItem()))
	})

	t.Run("memory strands parse error", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Memory Strands",
			Values: []clientModel.ItemValue{itemValue("xyz")},
		}))
		getter, err := IntFieldGetter(dbModel.MEMORY_STRANDS)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})

	t.Run("map tier parse error", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Map Tier",
			Values: []clientModel.ItemValue{itemValue("abc")},
		}))
		getter, err := IntFieldGetter(dbModel.MAP_TIER)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})

	t.Run("map quantity parse error", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Item Quantity",
			Values: []clientModel.ItemValue{itemValue("bad%")},
		}))
		getter, err := IntFieldGetter(dbModel.MAP_QUANT)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})

	t.Run("quality parse error", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Quality",
			Values: []clientModel.ItemValue{itemValue("bad%")},
		}))
		getter, err := IntFieldGetter(dbModel.QUALITY)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})

	t.Run("level parse error", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Level",
			Values: []clientModel.ItemValue{itemValue("abc")},
		}))
		getter, err := IntFieldGetter(dbModel.LEVEL)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})

	t.Run("graft skill level nil socketed items", func(t *testing.T) {
		getter, err := IntFieldGetter(dbModel.GRAFT_SKILL_LEVEL)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(makeItem()))
	})

	t.Run("graft skill level no properties on socketed item", func(t *testing.T) {
		item := makeItem(withSocketedItems(clientModel.Item{}))
		getter, err := IntFieldGetter(dbModel.GRAFT_SKILL_LEVEL)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})

	t.Run("graft skill level parse error", func(t *testing.T) {
		item := makeItem(withSocketedItems(clientModel.Item{
			Properties: &[]clientModel.ItemProperty{
				{Name: "Level", Values: []clientModel.ItemValue{itemValue("bad")}},
			},
		}))
		getter, err := IntFieldGetter(dbModel.GRAFT_SKILL_LEVEL)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})

	t.Run("map quantity no matching property", func(t *testing.T) {
		item := makeItem(withProperties(clientModel.ItemProperty{
			Name:   "Other Property",
			Values: []clientModel.ItemValue{itemValue("10")},
		}))
		getter, err := IntFieldGetter(dbModel.MAP_QUANT)
		require.NoError(t, err)
		assert.Equal(t, 0, getter(item))
	})
}

// ========== Additional StringComparator coverage ==========

func TestStringComparatorAdditional(t *testing.T) {
	t.Run("invalid regex for DOES_NOT_MATCH", func(t *testing.T) {
		_, err := StringComparator(makeCondition(dbModel.BASE_TYPE, dbModel.DOES_NOT_MATCH, "[invalid"))
		assert.Error(t, err)
	})

	t.Run("LENGTH_EQ invalid value", func(t *testing.T) {
		_, err := StringComparator(makeCondition(dbModel.NAME, dbModel.LENGTH_EQ, "abc"))
		assert.Error(t, err)
	})

	t.Run("LENGTH_GT invalid value", func(t *testing.T) {
		_, err := StringComparator(makeCondition(dbModel.NAME, dbModel.LENGTH_GT, "abc"))
		assert.Error(t, err)
	})

	t.Run("LENGTH_LT invalid value", func(t *testing.T) {
		_, err := StringComparator(makeCondition(dbModel.NAME, dbModel.LENGTH_LT, "abc"))
		assert.Error(t, err)
	})

	t.Run("invalid operator", func(t *testing.T) {
		_, err := StringComparator(makeCondition(dbModel.BASE_TYPE, dbModel.CONTAINS_ALL, "x"))
		assert.Error(t, err)
	})
}

// ========== Additional StringArrayComparator coverage ==========

func TestStringArrayComparatorAdditional(t *testing.T) {
	t.Run("CONTAINS_MATCH invalid regex", func(t *testing.T) {
		_, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.CONTAINS_MATCH, "[invalid"))
		assert.Error(t, err)
	})

	t.Run("DOES_NOT_MATCH invalid regex", func(t *testing.T) {
		_, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.DOES_NOT_MATCH, "[invalid"))
		assert.Error(t, err)
	})

	t.Run("LENGTH_EQ invalid value", func(t *testing.T) {
		_, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.LENGTH_EQ, "abc"))
		assert.Error(t, err)
	})

	t.Run("LENGTH_GT invalid value", func(t *testing.T) {
		_, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.LENGTH_GT, "abc"))
		assert.Error(t, err)
	})

	t.Run("LENGTH_LT invalid value", func(t *testing.T) {
		_, err := StringArrayComparator(makeCondition(dbModel.EXPLICITS, dbModel.LENGTH_LT, "abc"))
		assert.Error(t, err)
	})
}

// ========== Additional ComperatorFromConditions coverage ==========

func TestComperatorFromConditionsAdditional(t *testing.T) {
	t.Run("error in multi-condition list", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb"),
			makeCondition("INVALID_FIELD", dbModel.EQ, "x"),
		}
		_, err := ComperatorFromConditions(conditions)
		assert.Error(t, err)
	})
}

// ========== Additional ValidateConditions coverage ==========

func TestValidateConditionsAdditional(t *testing.T) {
	t.Run("empty conditions valid", func(t *testing.T) {
		assert.NoError(t, ValidateConditions(nil))
	})

	t.Run("invalid operator for field type", func(t *testing.T) {
		conditions := []*dbModel.Condition{
			makeCondition(dbModel.IS_CORRUPTED, dbModel.GT, "true"),
		}
		assert.Error(t, ValidateConditions(conditions))
	})
}

// ========== Additional NewItemChecker coverage ==========

func TestNewItemCheckerAdditional(t *testing.T) {
	t.Run("item class discriminator", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(1, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.ITEM_CLASS, dbModel.EQ, "StackableCurrency"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Chaos Orb")))
		require.Len(t, results, 1)
		assert.Equal(t, 1, results[0].ObjectiveId)
	})

	t.Run("item class discriminator no match", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(1, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.ITEM_CLASS, dbModel.EQ, "StackableCurrency"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Glorious Plate")))
		assert.Len(t, results, 0)
	})

	t.Run("name IN discriminator", func(t *testing.T) {
		objectives := []*dbModel.Objective{
			makeObjective(1, dbModel.ObjectiveTypeItem,
				makeCondition(dbModel.NAME, dbModel.IN, "Headhunter,Mageblood"),
			),
		}
		checker, err := NewItemChecker(objectives, true)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withName("Headhunter")))
		require.Len(t, results, 1)

		results = checker.CheckForCompletions(makeItem(withName("Mageblood")))
		require.Len(t, results, 1)

		results = checker.CheckForCompletions(makeItem(withName("Other")))
		assert.Len(t, results, 0)
	})

	t.Run("ignoreTime true ignores valid from/to", func(t *testing.T) {
		past := time.Now().Add(-2 * time.Hour)
		pastEnd := time.Now().Add(-1 * time.Hour)
		objectives := []*dbModel.Objective{
			{
				Id:            1,
				ObjectiveType: dbModel.ObjectiveTypeItem,
				Conditions:    []*dbModel.Condition{makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb")},
				ValidFrom:     &past,
				ValidTo:       &pastEnd,
			},
		}
		checker, err := NewItemChecker(objectives, true) // ignoreTime=true
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Chaos Orb")))
		require.Len(t, results, 1) // should match even though outside time window
	})

	t.Run("ValidFrom in future rejects", func(t *testing.T) {
		future := time.Now().Add(1 * time.Hour)
		futureEnd := time.Now().Add(2 * time.Hour)
		objectives := []*dbModel.Objective{
			{
				Id:            1,
				ObjectiveType: dbModel.ObjectiveTypeItem,
				Conditions:    []*dbModel.Condition{makeCondition(dbModel.BASE_TYPE, dbModel.EQ, "Chaos Orb")},
				ValidFrom:     &future,
				ValidTo:       &futureEnd,
			},
		}
		checker, err := NewItemChecker(objectives, false)
		require.NoError(t, err)

		results := checker.CheckForCompletions(makeItem(withBaseType("Chaos Orb")))
		assert.Len(t, results, 0)
	})
}

// ========== Additional ShouldUpdateCharacter coverage ==========

func TestShouldUpdateCharacterAdditional(t *testing.T) {
	timings := map[dbModel.TimingKey]time.Duration{
		dbModel.CharacterRefetchDelay:          5 * time.Minute,
		dbModel.CharacterRefetchDelayImportant: 2 * time.Minute,
		dbModel.CharacterRefetchDelayInactive:  30 * time.Minute,
		dbModel.InactivityDuration:             1 * time.Hour,
	}

	t.Run("cannot make requests", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			New:         Player{Character: &clientModel.Character{Name: "TestChar"}},
		}
		assert.False(t, p.ShouldUpdateCharacter(timings))
	})

	t.Run("high level without pantheon uses important delay", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			LastActive:  time.Now(),
			New: Player{Character: &clientModel.Character{
				Name:  "TestChar",
				Level: 45,
				// No pantheon - HasPantheon() returns false for empty Passives
			}},
		}
		// Updated 3 min ago - past important delay of 2 min
		p.LastUpdateTimes.Character = time.Now().Add(-3 * time.Minute)
		assert.True(t, p.ShouldUpdateCharacter(timings))

		// Updated 1 min ago - within important delay of 2 min
		p.LastUpdateTimes.Character = time.Now().Add(-1 * time.Minute)
		assert.False(t, p.ShouldUpdateCharacter(timings))
	})

	t.Run("high level with low ascendancy uses important delay", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			LastActive:  time.Now(),
			New: Player{Character: &clientModel.Character{
				Name:  "TestChar",
				Level: 70,
				// 0 ascendancy points (< 8)
				Passives: &clientModel.CharacterPassives{
					PantheonMajor: new(clientModel.Lunaris),
					PantheonMinor: new(clientModel.Gruthkul),
				},
			}},
		}
		// Updated 3 min ago - past important delay
		p.LastUpdateTimes.Character = time.Now().Add(-3 * time.Minute)
		assert.True(t, p.ShouldUpdateCharacter(timings))

		// Updated 1 min ago - within important delay
		p.LastUpdateTimes.Character = time.Now().Add(-1 * time.Minute)
		assert.False(t, p.ShouldUpdateCharacter(timings))
	})

	t.Run("normal update within delay returns false", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			LastActive:  time.Now(),
			New: Player{Character: &clientModel.Character{
				Name:  "TestChar",
				Level: 30,
			}},
		}
		// Updated 2 min ago - within normal delay of 5 min
		p.LastUpdateTimes.Character = time.Now().Add(-2 * time.Minute)
		assert.False(t, p.ShouldUpdateCharacter(timings))
	})
}

// ========== Additional ShouldUpdateLeagueAccount coverage ==========

func TestShouldUpdateLeagueAccountAdditional(t *testing.T) {
	timings := map[dbModel.TimingKey]time.Duration{
		dbModel.LeagueAccountRefetchDelay:          5 * time.Minute,
		dbModel.LeagueAccountRefetchDelayImportant: 2 * time.Minute,
		dbModel.LeagueAccountRefetchDelayInactive:  30 * time.Minute,
		dbModel.InactivityDuration:                 1 * time.Hour,
	}

	t.Run("cannot make requests", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			New:         Player{Character: &clientModel.Character{Level: 60}},
		}
		assert.False(t, p.ShouldUpdateLeagueAccount(timings))
	})

	t.Run("inactive player uses longer delay", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			LastActive:  time.Now().Add(-2 * time.Hour), // inactive
			New: Player{
				Character:         &clientModel.Character{Level: 60},
				AtlasPassiveTrees: []clientModel.LeagueAccountAtlasPassiveTree{{Hashes: make([]int, 50)}},
			},
		}
		// Updated 10 min ago - within inactive delay of 30 min
		p.LastUpdateTimes.LeagueAccount = time.Now().Add(-10 * time.Minute)
		assert.False(t, p.ShouldUpdateLeagueAccount(timings))

		// Updated 35 min ago - past inactive delay
		p.LastUpdateTimes.LeagueAccount = time.Now().Add(-35 * time.Minute)
		assert.True(t, p.ShouldUpdateLeagueAccount(timings))
	})

	t.Run("important update within delay returns false", func(t *testing.T) {
		p := &PlayerUpdate{
			Token:       "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
			LastActive:  time.Now(),
			New: Player{
				Character:         &clientModel.Character{Level: 60},
				AtlasPassiveTrees: []clientModel.LeagueAccountAtlasPassiveTree{{Hashes: make([]int, 50)}},
			},
		}
		// Updated 1 min ago - within important delay of 2 min
		p.LastUpdateTimes.LeagueAccount = time.Now().Add(-1 * time.Minute)
		assert.False(t, p.ShouldUpdateLeagueAccount(timings))
	})
}

// ========== Quality helper ==========

func TestQuality(t *testing.T) {
	t.Run("armour quality", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueArmourQuality)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		assert.Equal(t, 0, checker(&Player{Character: nil}))

		props := []clientModel.ItemProperty{
			{Name: "Quality", Values: []clientModel.ItemValue{itemValue("+20%")}},
		}
		equipment := []clientModel.Item{
			{BaseType: "Glorious Plate", Properties: &props},
		}
		// Glorious Plate -> BodyArmours -> Armour superclass
		result := checker(&Player{Character: &clientModel.Character{Equipment: &equipment}})
		assert.Equal(t, 20, result)
	})

	t.Run("weapon quality", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueWeaponQuality)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)
		assert.Equal(t, 0, checker(&Player{Character: nil}))
	})

	t.Run("flask quality", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueFlaskQuality)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)
		assert.Equal(t, 0, checker(&Player{Character: nil}))
	})

	t.Run("quality nil equipment", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueArmourQuality)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)
		assert.Equal(t, 0, checker(&Player{Character: &clientModel.Character{Equipment: nil}}))
	})

	t.Run("quality skips non-matching superclass", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueWeaponQuality)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		props := []clientModel.ItemProperty{
			{Name: "Quality", Values: []clientModel.ItemValue{itemValue("+20%")}},
		}
		// Glorious Plate is Armour, not Weapon
		equipment := []clientModel.Item{
			{BaseType: "Glorious Plate", Properties: &props},
		}
		assert.Equal(t, 0, checker(&Player{Character: &clientModel.Character{Equipment: &equipment}}))
	})

	t.Run("quality skips items without properties", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueArmourQuality)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		equipment := []clientModel.Item{
			{BaseType: "Glorious Plate"}, // no properties
		}
		assert.Equal(t, 0, checker(&Player{Character: &clientModel.Character{Equipment: &equipment}}))
	})

	t.Run("quality parse error", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueArmourQuality)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		props := []clientModel.ItemProperty{
			{Name: "Quality", Values: []clientModel.ItemValue{itemValue("bad%")}},
		}
		equipment := []clientModel.Item{
			{BaseType: "Glorious Plate", Properties: &props},
		}
		assert.Equal(t, 0, checker(&Player{Character: &clientModel.Character{Equipment: &equipment}}))
	})

	t.Run("quality sums multiple items", func(t *testing.T) {
		obj := makePlayerObjective(1, dbModel.TrackedValueArmourQuality)
		checker, err := GetPlayerChecker(obj)
		require.NoError(t, err)

		props1 := []clientModel.ItemProperty{
			{Name: "Quality", Values: []clientModel.ItemValue{itemValue("+20%")}},
		}
		props2 := []clientModel.ItemProperty{
			{Name: "Quality", Values: []clientModel.ItemValue{itemValue("+15%")}},
		}
		equipment := []clientModel.Item{
			{BaseType: "Glorious Plate", Properties: &props1},
			{BaseType: "Glorious Plate", Properties: &props2},
		}
		assert.Equal(t, 35, checker(&Player{Character: &clientModel.Character{Equipment: &equipment}}))
	})
}
