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

// regular nodes
var ascendancyNodesPoE1 = utils.ToSet([]int{193, 258, 607, 662, 699, 758, 869, 922, 982, 1105, 1564, 1675, 1697, 1729, 1731, 1734, 1945, 1953, 1977, 2060, 2336, 2521, 2598, 2643, 2768, 2872, 3154, 3184, 3458, 3554, 3651, 3872, 4194, 4242, 4245, 4494, 4849, 4917, 5082, 5087, 5220, 5415, 5443, 5452, 5502, 5643, 5819, 5865, 5926, 5929, 6028, 6038, 6052, 6064, 6344, 6728, 6778, 6982, 7403, 7442, 7618, 8066, 8081, 8138, 8243, 8281, 8419, 8592, 8638, 8656, 8967, 8982, 9014, 9258, 9271, 9327, 9437, 9971, 10099, 10143, 10238, 10538, 10539, 10635, 10916, 11046, 11412, 11490, 11597, 12146, 12354, 12475, 12542, 12597, 12738, 12850, 12860, 13219, 13362, 13374, 13851, 13988, 14103, 14156, 14190, 14588, 14603, 14726, 14870, 14996, 15286, 15544, 15550, 15616, 15900, 16093, 16212, 16306, 16350, 16633, 16745, 16848, 16940, 16994, 17018, 17080, 17309, 17315, 17386, 17445, 17570, 17988, 18147, 18309, 18335, 18519, 18574, 19078, 19083, 19417, 19488, 19587, 19595, 19598, 19641, 19689, 19993, 20050, 20160, 20480, 20772, 20931, 20954, 21192, 21264, 21785, 22174, 22441, 22551, 22628, 22637, 22852, 23024, 23155, 23169, 23225, 23509, 23572, 23972, 24214, 24432, 24462, 24528, 24538, 24755, 24764, 24848, 24924, 25111, 25167, 25187, 25309, 25651, 25703, 25795, 25944, 26055, 26067, 26298, 26446, 26596, 26714, 27038, 27054, 27055, 27096, 27121, 27123, 27536, 27604, 27737, 27864, 27869, 28535, 28601, 28609, 28782, 28884, 28995, 29026, 29161, 29212, 29259, 29630, 29662, 29825, 29844, 29994, 30061, 30271, 30632, 30675, 30690, 30919, 31316, 31344, 31364, 31517, 31598, 31667, 31700, 31878, 31984, 32095, 32115, 32216, 32249, 32251, 32265, 32364, 32417, 32605, 32640, 32662, 32669, 32740, 32816, 32947, 32992, 33167, 33179, 33467, 33645, 33670, 33875, 33897, 33940, 33954, 34215, 34229, 34434, 34442, 34484, 34567, 34637, 34774, 34814, 34952, 34977, 35011, 35069, 35185, 35380, 35448, 35598, 35739, 35750, 35877, 35936, 36017, 36020, 36072, 36242, 36393, 36958, 37114, 37127, 37191, 37303, 37419, 37486, 37492, 37623, 37670, 38180, 38387, 38689, 38740, 38886, 38918, 38999, 39142, 39381, 39486, 39598, 39728, 39790, 39818, 39834, 40010, 40059, 40104, 40276, 40510, 40631, 40810, 40813, 40953, 41081, 41387, 41433, 41464, 41534, 41891, 41996, 42144, 42264, 42293, 42546, 42659, 42671, 42861, 43122, 43193, 43195, 43215, 43242, 43336, 43376, 43645, 43660, 43725, 43857, 43902, 43962, 44297, 44482, 44797, 44848, 44903, 45313, 45403, 45696, 45727, 46479, 46649, 46704, 46952, 47058, 47366, 47486, 47531, 47543, 47630, 47767, 47778, 47873, 48040, 48124, 48214, 48239, 48251, 48406, 48410, 48480, 48606, 48711, 48719, 48760, 48816, 48836, 48904, 48999, 49153, 49785, 50024, 50692, 50845, 50935, 51101, 51402, 51462, 51492, 51685, 51998, 52094, 52349, 52408, 52417, 52553, 52575, 52633, 52780, 52855, 53086, 53095, 53123, 53398, 53414, 53421, 53816, 53884, 53992, 54139, 54159, 54279, 54569, 54877, 54928, 55146, 55509, 55646, 55686, 55867, 55985, 56134, 56292, 56461, 56722, 56789, 56940, 56967, 57052, 57175, 57197, 57222, 57429, 57560, 58029, 58281, 58454, 58650, 58827, 58998, 58999, 59067, 59800, 59826, 59837, 59920, 60118, 60462, 60508, 60511, 60547, 60582, 60686, 60769, 61072, 61092, 61133, 61259, 61355, 61372, 61393, 61478, 61627, 61761, 61805, 62067, 62136, 62162, 62349, 62504, 62595, 62817, 63020, 63135, 63293, 63357, 63362, 63417, 63490, 63583, 63623, 63673, 63834, 63908, 63940, 63954, 64028, 64768, 64785, 64842, 64963, 65085, 65153, 65296})

var bloodlineNodes = utils.ToSet([]int{699, 2643, 3872, 4245, 5220, 5452, 6344, 7403, 7442, 8066, 8138, 8243, 8638, 8982, 9437, 10538, 10539, 10916, 12354, 12542, 12860, 13362, 13988, 14190, 14588, 16350, 17080, 17309, 17570, 18519, 19078, 19689, 20772, 20931, 21785, 22174, 23155, 24462, 24764, 24924, 25187, 25703, 26596, 27121, 27737, 27869, 28601, 28609, 29212, 29259, 30061, 30271, 30632, 31878, 32216, 32265, 32605, 32740, 33670, 33897, 34229, 34442, 34637, 34814, 34952, 34977, 35011, 35069, 35380, 36020, 36393, 37670, 38740, 38886, 39142, 39381, 39486, 40953, 41387, 41464, 43376, 43660, 43902, 44848, 44903, 45727, 46649, 46704, 47543, 47767, 48251, 48406, 48606, 48711, 48816, 48836, 49785, 51685, 52349, 52408, 52417, 52553, 52780, 52855, 53398, 53414, 54139, 58281, 58999, 59826, 60118, 60511, 60686, 61092, 63020, 63362, 63623, 63834, 64963})

// phrecia nodes
// var ascendancyNodesPoE1 = utils.ToSet([]int{322, 363, 409, 531, 540, 848, 863, 1094, 1116, 1424, 1582, 1811, 1830, 2048, 2103, 2123, 2134, 2266, 2380, 2437, 2670, 2688, 2707, 2727, 2748, 3125, 3245, 3599, 3702, 3892, 3897, 3952, 4174, 4491, 4561, 4888, 5266, 5273, 5281, 5523, 5888, 5903, 5913, 6105, 6140, 6252, 6281, 6311, 6481, 6605, 6613, 6814, 7128, 7237, 7629, 7842, 8455, 8636, 8756, 8909, 9053, 9197, 9517, 9529, 9828, 9982, 10135, 10870, 11043, 11127, 11130, 11154, 11213, 11217, 11219, 11385, 11873, 12126, 12195, 12264, 12332, 12421, 12789, 12915, 13061, 13763, 13903, 13971, 14144, 14213, 14442, 14681, 14858, 15014, 15390, 15486, 15874, 16023, 16115, 16305, 16317, 16440, 16638, 16639, 16671, 16759, 17025, 17123, 17140, 17298, 17328, 17511, 17765, 17853, 17999, 18026, 18313, 18378, 18551, 18588, 18635, 18806, 18817, 18859, 18893, 19024, 19258, 19611, 19967, 20073, 20105, 20239, 20602, 20797, 21095, 21289, 21521, 21905, 21979, 22056, 22167, 22622, 22636, 22720, 23061, 23429, 23770, 23799, 23851, 23940, 24063, 24105, 24174, 24188, 24206, 24342, 24394, 24607, 24698, 24701, 24704, 24726, 24800, 24938, 24984, 25120, 25478, 25530, 26231, 26415, 26496, 26963, 27339, 27380, 27473, 27553, 27664, 28212, 28454, 28569, 28815, 29126, 29127, 29166, 29294, 29437, 29654, 29846, 30034, 30419, 30769, 30851, 30940, 31254, 31396, 32105, 32223, 32730, 32742, 33031, 33795, 33835, 34223, 34257, 34448, 34482, 34708, 34762, 34767, 34822, 35022, 35066, 35432, 35754, 36396, 36782, 36919, 36978, 37082, 37134, 37308, 37469, 37534, 37553, 37757, 37782, 37852, 38079, 38080, 38139, 38400, 38522, 38874, 39144, 39246, 39463, 39702, 39737, 39868, 40058, 40063, 40139, 40171, 40299, 40300, 40465, 40664, 40731, 40762, 41042, 41349, 41560, 41643, 41937, 41978, 42290, 42396, 42487, 42716, 42923, 43029, 43431, 43474, 43641, 43703, 43835, 44410, 44996, 45027, 45368, 45399, 45449, 46465, 46492, 46836, 47042, 47200, 47637, 48062, 48223, 48353, 48456, 48586, 48740, 48846, 48926, 49271, 49511, 49543, 49658, 49731, 49937, 50314, 50330, 50352, 50743, 50844, 50932, 51178, 51229, 51449, 52104, 52264, 52296, 52308, 52323, 52629, 52644, 52944, 53334, 53426, 53715, 53922, 54067, 54114, 54201, 54394, 54663, 55065, 55236, 55407, 55734, 56028, 56067, 56184, 56538, 56701, 56856, 56883, 57157, 57630, 58026, 58229, 58425, 58427, 58661, 58846, 59242, 59443, 59500, 59546, 59943, 59958, 60154, 60156, 60505, 60757, 60791, 60867, 61085, 61147, 61246, 61282, 61696, 61871, 61900, 62155, 62230, 62352, 62445, 62534, 62851, 62974, 63336, 63992, 64012, 64038, 64111, 64171, 64248, 64905, 65410})

var ascendancyNodesPoE2 = utils.ToSet([]int{16, 30, 40, 59, 74, 110, 528, 664, 762, 770, 1347, 1442, 1579, 1583, 1988, 1994, 2516, 2702, 2857, 2877, 2995, 3065, 3084, 3165, 3704, 3762, 3781, 3987, 4245, 4495, 4891, 5386, 5563, 5817, 5852, 6109, 6127, 6935, 7120, 7246, 7621, 7656, 7793, 7979, 7998, 8143, 8272, 8415, 8525, 8611, 8854, 8867, 9294, 9798, 9988, 9994, 9997, 10072, 10371, 10694, 10731, 10987, 11641, 11771, 11776, 12000, 12054, 12183, 12488, 12795, 12876, 12882, 13065, 13174, 13673, 13675, 13715, 13772, 14429, 14508, 14960, 15044, 16100, 16249, 16276, 16433, 17058, 17268, 17646, 17754, 17788, 17923, 18146, 18158, 18348, 18585, 18678, 18826, 18849, 19233, 19424, 19482, 20195, 20772, 20830, 20895, 22147, 22541, 22661, 22908, 23005, 23352, 23415, 23416, 23508, 23710, 23880, 24039, 24135, 24226, 24295, 24475, 24807, 24868, 25172, 25239, 25434, 25438, 25618, 25779, 25781, 25885, 25935, 26085, 26282, 26638, 27418, 27667, 27686, 27990, 28153, 28431, 29074, 29162, 29323, 29398, 29645, 29871, 30071, 30115, 30151, 30233, 30996, 31116, 31223, 32534, 32559, 32560, 32637, 32699, 32771, 32952, 33141, 33570, 33736, 33812, 34419, 34501, 34817, 34882, 35033, 35187, 35453, 35801, 36252, 36365, 36564, 36659, 36676, 36696, 36728, 36788, 36822, 37046, 37078, 37336, 37397, 37523, 38014, 38578, 38601, 38769, 39204, 39241, 39292, 39365, 39411, 39470, 39640, 39723, 40719, 40721, 40915, 41008, 41076, 41619, 41736, 42017, 42035, 42275, 42416, 42441, 42522, 42845, 43095, 43128, 43131, 44357, 44371, 44484, 44746, 45248, 46016, 46071, 46454, 46522, 46535, 46644, 46990, 47097, 47184, 47236, 47312, 47344, 47442, 48537, 48682, 49049, 49165, 49189, 49340, 49380, 49503, 49759, 50098, 50192, 50219, 51142, 51690, 51737, 52068, 52448, 53108, 53762, 54194, 54838, 54892, 55536, 55582, 55611, 55796, 56162, 56842, 57141, 57181, 57253, 57819, 57959, 58149, 58574, 58591, 58704, 58747, 58751, 58932, 59342, 59372, 59540, 59759, 59822, 59913, 60287, 60298, 60634, 60662, 60859, 60913, 61039, 61267, 61461, 61804, 61897, 61973, 61985, 61991, 62388, 62797, 62804, 63002, 63236, 63254, 63259, 63401, 63484, 63713, 63894, 64031, 64117, 64379, 64789, 64962, 65173, 65413, 65518})

func (c *Character) GetMainSkill() string {
	mainSkill := ""
	maxLinks := 0
	if c.Realm == Poe2 {
		for _, skillGem := range *c.Skills {
			links := 0
			if (skillGem.Support == nil || *skillGem.Support) || skillGem.Sockets == nil {
				continue
			}
			for _, socket := range *skillGem.Sockets {
				if socket.Item != nil && *socket.Item == ItemSocketItemSupportgem {
					links++
				}
			}
			if links > maxLinks {
				maxLinks = links
				mainSkill = skillGem.BaseType
			}
		}
		return mainSkill
	}

	for _, item := range *c.Equipment {
		if item.SocketedItems == nil || item.Sockets == nil {
			continue
		}
		socketedItems := *item.SocketedItems
		sockets := *item.Sockets
		for _, gem := range socketedItems {

			if gem.Support == nil || *gem.Support {
				continue
			}
			group := sockets[*gem.Socket].Group
			links := 1
			for socketId, socket := range sockets {
				if len(socketedItems) <= socketId || socket.Group != group {
					continue
				}
				support := socketedItems[socketId].Support
				if support != nil && *support {
					links++
				}
			}
			if links > maxLinks {
				maxLinks = links
				mainSkill = gem.BaseType
			}

		}
	}
	return mainSkill
}
func (c *Character) GetAscendancyPoints() int {
	if c.Passives == nil {
		return 0
	}
	if c.Realm == Poe2 {
		return len(ascendancyNodesPoE2.Intersection(utils.ToSet(c.Passives.Hashes)))
	}
	return len(ascendancyNodesPoE1.Intersection(utils.ToSet(c.Passives.Hashes)))
}

func (c *Character) GetBloodlinePoints() int {
	if c.Passives == nil {
		return 0
	}
	return len(bloodlineNodes.Intersection(utils.ToSet(c.Passives.Hashes)))
}

func (c *Character) HasPantheon() bool {
	if c.Realm != Poe2 && c.Passives != nil {
		return c.Passives.PantheonMajor != nil && c.Passives.PantheonMinor != nil
	}
	return false
}

type ItemValue []any

func (v ItemValue) Name() string {
	return v[0].(string)
}

func (v ItemValue) Id() int {
	return v[1].(int)
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

func (g *StashTab) FlatMap() []*StashTab {
	var result []*StashTab
	result = append(result, g)
	if g.Children != nil {
		for _, child := range *g.Children {
			result = append(result, child.FlatMap()...)
		}
	}
	return result
}

func (g *StashTab) GetHash() [32]byte {
	var idAggregate strings.Builder
	idAggregate.WriteString(g.Id)
	for _, item := range utils.Deref(g.Items) {
		idAggregate.WriteString(*item.Id)
		if item.StackSize != nil {
			idAggregate.WriteString(fmt.Sprint(*item.StackSize))
		}
	}
	return sha256.Sum256([]byte(idAggregate.String()))
}
func (c *Character) GetAllItems() []Item {
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

func (c *Character) GetVoidStones() utils.Set[string] {
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

func (c *Character) GetNumberOfHighIlvlFlasks() int {
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

func (c *Character) HasSameEquipment(other *Character) bool {
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

func (c *Character) HasSameGems(other *Character) bool {
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

func (i *Item) GetPositionIndex() string {
	return fmt.Sprintf("%s-%d-%d", utils.Deref(i.InventoryId), utils.Deref(i.X), utils.Deref(i.Y))
}

func (i *Item) Equals(other Item) bool {
	if i.Name != other.Name || i.TypeLine != other.TypeLine || i.BaseType != other.BaseType {
		return false
	}
	return modsEqual(i.ImplicitMods, other.ImplicitMods) &&
		modsEqual(i.ExplicitMods, other.ExplicitMods)
}

func modsEqual(a, b *[]ItemMod) bool {
	if (a != nil && b == nil) || (a == nil && b != nil) {
		return false
	}
	return (a == nil && b == nil) || utils.ArrayEqualsUnordered(*a, *b)
}

// UnmarshalJSON handles the PoE API quirk of sending "flags": [] instead of
// an object when a mod has no flags set.
func (f *ItemModFlags) UnmarshalJSON(data []byte) error {
	if string(data) == "[]" {
		*f = ItemModFlags{}
		return nil
	}
	type itemModFlagsAlias ItemModFlags
	return json.Unmarshal(data, (*itemModFlagsAlias)(f))
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
