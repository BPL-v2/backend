package scoring

import (
	"bpl/metrics"
	"bpl/repository"
	"bpl/utils"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type ScoreType string

type PresetCompletion struct {
	Finished    bool
	Timestamp   time.Time
	Rank        int
	UserId      int
	Points      int
	Number      int
	ObjectiveId int
}

type Score struct {
	ObjectiveId       int
	TeamId            int
	PresetCompletions map[int]*PresetCompletion
	HideProgress      bool
	BonusPoints       int
}

func (s *Score) Finished() bool {
	if len(s.PresetCompletions) == 0 {
		return false
	}
	for _, pc := range s.PresetCompletions {
		if !pc.Finished {
			return false
		}
	}
	return true
}

func (s *Score) Timestamp() time.Time {
	latest := time.Time{}
	if !s.Finished() {
		return latest
	}
	for _, pc := range s.PresetCompletions {
		if pc.Timestamp.After(latest) {
			latest = pc.Timestamp
		}
	}
	return latest
}

func (s *Score) Points() int {
	total := s.BonusPoints
	for _, pc := range s.PresetCompletions {
		total += pc.Points
	}
	return total
}

func (s *Score) CanShowTo(teamId int) bool {
	return (s.TeamId == teamId) || s.Finished() || !s.HideProgress
}

func EvaluateAggregations(objective *repository.Objective, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	timer := prometheus.NewTimer(metrics.ScoreEvaluationDuration)
	defer timer.ObserveDuration()
	for _, childObjective := range objective.Children {
		err := EvaluateAggregations(childObjective, aggregations, scoreMap)
		if err != nil {
			return err
		}
	}
	for _, preset := range objective.ScoringRules {
		if fun, ok := scoringFunctions[preset.RuleType]; ok {
			err := fun(objective, preset, aggregations, scoreMap)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type TeamCompletion struct {
	TeamId              int
	ObjectivesCompleted int
	LatestTimestamp     int64
}

var scoringFunctions = map[repository.ScoringRuleType]func(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error{
	repository.FIXED_POINTS_ON_COMPLETION:    handlePresence,
	repository.RANK_BY_COMPLETION_TIME:       handleRankedTime,
	repository.RANK_BY_HIGHEST_VALUE:         handleRankedValue,
	repository.RANK_BY_LOWEST_VALUE:          handleRankedReverse,
	repository.POINTS_BY_VALUE:               handlePointsFromValue,
	repository.RANK_BY_CHILD_COMPLETION_TIME: handleChildRankingByTime,
	repository.BONUS_PER_CHILD_COMPLETION:    handleChildBonus,
	repository.BINGO_BOARD_RANKING:           handleBingoBoard,
	repository.RANK_BY_CHILD_VALUE_SUM:       handleChildRankingByNumber,
}

func handlePointsFromValue(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	for teamId, match := range aggregations[objective.Id] {
		if scoreMap[teamId] == nil || scoreMap[teamId][objective.Id] == nil || scoreMap[teamId][objective.Id].PresetCompletions[scoringRule.Id] == nil {
			continue
		}
		completion := scoreMap[teamId][objective.Id].PresetCompletions[scoringRule.Id]
		completion.Number = match.Number
		completion.UserId = match.UserId
		completion.Finished = match.Finished
		completion.Timestamp = match.Timestamp
		completion.Points = int(scoringRule.Points.GetScoreFromNumber(match.Number))
		if scoringRule.PointCap != 0 && completion.Points > scoringRule.PointCap {
			completion.Points = scoringRule.PointCap
		}
	}
	return nil
}

func handlePresence(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	for teamId, match := range aggregations[objective.Id] {
		if scoreMap[teamId] == nil || scoreMap[teamId][objective.Id] == nil || scoreMap[teamId][objective.Id].PresetCompletions[scoringRule.Id] == nil {
			continue
		}
		completion := scoreMap[teamId][objective.Id].PresetCompletions[scoringRule.Id]
		completion.Number = match.Number
		completion.UserId = match.UserId
		completion.Finished = match.Finished
		completion.Timestamp = match.Timestamp
		if match.Finished {
			completion.Points = int(scoringRule.Points.Get(0))
		}
	}
	return nil
}
func handleRankedTime(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	rankFun := func(a, b *Match) bool {
		if a.Finished && b.Finished {
			return a.Timestamp.Before(b.Timestamp)
		}
		return a.Finished
	}
	return handleRanked(objective, scoringRule, aggregations, rankFun, scoreMap)
}

func handleRankedValue(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	rankFun := func(a, b *Match) bool {
		if a.Number == b.Number {
			return a.Timestamp.Before(b.Timestamp)
		}
		return a.Number > b.Number
	}
	return handleRanked(objective, scoringRule, aggregations, rankFun, scoreMap)
}

func handleRankedReverse(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	rankFun := func(a, b *Match) bool {
		if a.Number == b.Number {
			return a.Timestamp.Before(b.Timestamp)
		}
		return a.Number < b.Number
	}
	return handleRanked(objective, scoringRule, aggregations, rankFun, scoreMap)
}

func isTiedWithNext(index int, matches []*Match, rankFun func(a, b *Match) bool) bool {
	if index >= len(matches)-1 {
		return false
	}
	return rankFun(matches[index], matches[index+1]) == rankFun(matches[index+1], matches[index])
}

func handleRanked(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, rankFun func(a, b *Match) bool, scoreMap map[int]map[int]*Score) error {
	matches := make([]*Match, 0)
	for _, match := range aggregations[objective.Id] {
		matches = append(matches, match)
	}
	sort.Slice(matches, func(i, j int) bool { return rankFun(matches[i], matches[j]) })
	i := 0
	for j, match := range matches {
		if scoreMap[match.TeamId] == nil || scoreMap[match.TeamId][objective.Id] == nil || scoreMap[match.TeamId][objective.Id].PresetCompletions[scoringRule.Id] == nil {
			continue
		}
		completion := scoreMap[match.TeamId][objective.Id].PresetCompletions[scoringRule.Id]
		completion.UserId = match.UserId
		completion.Timestamp = match.Timestamp
		completion.Number = match.Number
		completion.Finished = match.Finished

		if match.Finished {
			completion.Rank = i + 1
			completion.Points = int(scoringRule.Points.Get(i))
		}
		if !isTiedWithNext(j, matches, rankFun) {
			i++
		}
	}
	return nil
}

type Tuple struct {
	X int
	Y int
}

type GridFinish struct {
	Grid Tuple
	Time time.Time
}

func handleBingoBoard(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	numberOfBingosRequired := 1
	if val, ok := scoringRule.Extra["required_bingo_count"]; ok {
		parsed, err := strconv.Atoi(val)
		if err == nil {
			numberOfBingosRequired = parsed
		}
	}
	objectiveMap := make(map[int]*repository.Objective)
	for _, child := range objective.Children {
		objectiveMap[child.Id] = child
	}
	gridCellMap := make(map[int]Tuple)
	regex := regexp.MustCompile(`(\d+),(\d+)`)
	gridSize := 0
	for _, child := range objective.Children {
		matches := regex.FindStringSubmatch(child.Extra)
		if len(matches) == 3 {
			x, _ := strconv.Atoi(matches[1])
			y, _ := strconv.Atoi(matches[2])
			gridCellMap[child.Id] = Tuple{X: x, Y: y}
			gridSize = max(gridSize, x+1, y+1)
		}
	}

	teamChildFinishes := make(map[int][]GridFinish)

	for teamId, teamScores := range scoreMap {
		for childId := range objectiveMap {
			childScore := teamScores[childId]
			if childScore != nil && childScore.Finished() {
				teamChildFinishes[teamId] = append(teamChildFinishes[teamId], GridFinish{Grid: gridCellMap[childId], Time: childScore.Timestamp()})
			}
		}
	}

	bingoScores := []*PresetCompletion{}
	for teamId, finishedGridCells := range teamChildFinishes {
		gridTimestamps := make(map[int]map[int]time.Time)
		for _, completion := range finishedGridCells {
			cellPos := completion.Grid
			if _, exists := gridTimestamps[cellPos.X]; !exists {
				gridTimestamps[cellPos.X] = make(map[int]time.Time)
			}
			gridTimestamps[cellPos.X][cellPos.Y] = completion.Time
		}
		if scoreMap[teamId] == nil || scoreMap[teamId][objective.Id] == nil || scoreMap[teamId][objective.Id].PresetCompletions[scoringRule.Id] == nil {
			continue
		}
		completion := scoreMap[teamId][objective.Id].PresetCompletions[scoringRule.Id]
		finishTime := getBingoCompletionTime(numberOfBingosRequired, gridTimestamps, gridSize)
		if !finishTime.IsZero() {
			completion.Finished = true
			completion.Timestamp = finishTime
		}
		bingoScores = append(bingoScores, completion)
	}

	sort.Slice(bingoScores, func(i, j int) bool {
		if bingoScores[i].Finished != bingoScores[j].Finished {
			return bingoScores[i].Finished
		}
		return bingoScores[i].Timestamp.Before(bingoScores[j].Timestamp)
	})
	rank := 1
	for i, score := range bingoScores {
		if score.Finished {
			if i > 0 && (bingoScores[i-1].Finished && bingoScores[i-1].Timestamp.Before(score.Timestamp)) {
				rank = i + 1
			}
			score.Rank = rank
			score.Points = int(scoringRule.Points.Get(rank - 1))
		}
	}
	return nil
}

func getBingoCompletionTime(numberOfBingosRequired int, girdTimestamps map[int]map[int]time.Time, gridSize int) time.Time {
	finishTime := int64(math.MaxInt64)
	rowTimes := map[int][]int64{}
	colTimes := map[int][]int64{}
	diag1Times := []int64{}
	diag2Times := []int64{}
	for x, row := range girdTimestamps {
		for y := range row {
			gridSize = max(gridSize, x, y)
		}
	}

	for x, row := range girdTimestamps {
		for y, timestamp := range row {
			rowTimes[x] = append(rowTimes[x], timestamp.UnixNano())
			colTimes[y] = append(colTimes[y], timestamp.UnixNano())
			if x == y {
				diag1Times = append(diag1Times, timestamp.UnixNano())
			}
			if x+y == gridSize-1 {
				diag2Times = append(diag2Times, timestamp.UnixNano())
			}
		}
	}
	bingoCount := 0
	for i := 0; i < gridSize; i++ {
		if len(rowTimes[i]) == gridSize {
			finishTime = min(utils.Max(rowTimes[i]...), finishTime)
			bingoCount++
		}
		if len(colTimes[i]) == gridSize {
			finishTime = min(utils.Max(colTimes[i]...), finishTime)
			bingoCount++
		}
	}
	if len(diag1Times) == gridSize {
		finishTime = min(utils.Max(diag1Times...), finishTime)
		bingoCount++
	}
	if len(diag2Times) == gridSize {
		finishTime = min(utils.Max(diag2Times...), finishTime)
		bingoCount++
	}
	if finishTime == int64(math.MaxInt64) || bingoCount < numberOfBingosRequired {
		return time.Time{}
	}
	return time.Unix(0, finishTime)
}

func getLeafIds(objective *repository.Objective) []int {
	if len(objective.Children) == 0 {
		return []int{objective.Id}
	}
	leafIds := []int{}
	for _, child := range objective.Children {
		leafIds = append(leafIds, getLeafIds(child)...)
	}
	return leafIds
}

func handleChildBonus(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	finishCounts := make(map[int]int)
	teamIds := make(map[int]bool)
	childIds := getLeafIds(objective)

	teamChildScores := map[int][]*Score{}
	for teamId, objectiveScores := range scoreMap {
		teamIds[teamId] = true
		for _, id := range childIds {
			score := objectiveScores[id]
			if score != nil && score.Finished() {
				finishCounts[teamId]++
				teamChildScores[teamId] = append(teamChildScores[teamId], score)
			}
		}
		sort.Slice(teamChildScores[teamId], func(i, j int) bool {
			return teamChildScores[teamId][i].Timestamp().Before(teamChildScores[teamId][j].Timestamp())
		})
	}

	for teamId, childScores := range teamChildScores {
		latestTimestamp := time.Time{}
		for i, childScore := range childScores {
			childScore.BonusPoints += int(scoringRule.Points.Get(i))
			currentTimestamp := childScore.Timestamp()
			if currentTimestamp.After(latestTimestamp) {
				latestTimestamp = currentTimestamp
			}
		}
		if scoreMap[teamId] == nil || scoreMap[teamId][objective.Id] == nil || scoreMap[teamId][objective.Id].PresetCompletions[scoringRule.Id] == nil {
			continue
		}
		completion := scoreMap[teamId][objective.Id].PresetCompletions[scoringRule.Id]
		completion.Finished = finishCounts[teamId] == len(objective.Children)
		completion.Number = finishCounts[teamId]
		completion.Timestamp = latestTimestamp
	}
	return nil
}

func handleChildRankingByTime(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	requiredChildCompletions := len(objective.Children)
	if val, ok := scoringRule.Extra["required_completed_children"]; ok {
		parsed, err := strconv.Atoi(val)
		if err == nil {
			requiredChildCompletions = parsed
		}
	}
	if val, ok := scoringRule.Extra["required_completed_children_percent"]; ok {
		parsed, err := strconv.Atoi(val)
		if err == nil {
			requiredChildCompletions = (len(objective.Children) * parsed) / 100
		}
	}
	teamCompletions := make(map[int]*TeamCompletion)
	teamCompletionTimestamps := make(map[int][]int64)
	for teamId := range scoreMap {
		teamCompletions[teamId] = &TeamCompletion{TeamId: teamId}
	}
	childIds := map[int]bool{}
	for _, child := range objective.Children {
		childIds[child.Id] = true
		for teamId, objectiveScores := range scoreMap {
			childScore := objectiveScores[child.Id]
			if childScore != nil && childScore.Finished() {
				timestamp := childScore.Timestamp().UnixNano()
				teamCompletionTimestamps[teamId] = append(teamCompletionTimestamps[teamId], timestamp)
				teamCompletions[teamId].ObjectivesCompleted++
				teamCompletions[teamId].LatestTimestamp = max(teamCompletions[teamId].LatestTimestamp, timestamp)
			}
		}
	}
	for teamId, completion := range teamCompletions {
		if completion.ObjectivesCompleted < requiredChildCompletions || requiredChildCompletions <= 0 {
			continue
		}
		timestamps := teamCompletionTimestamps[teamId]
		slices.Sort(timestamps)
		completion.LatestTimestamp = timestamps[requiredChildCompletions-1]
	}
	rankedTeams := utils.Values(teamCompletions)
	sort.Slice(rankedTeams, func(i, j int) bool {
		ifinished := rankedTeams[i].ObjectivesCompleted >= requiredChildCompletions
		jfinished := rankedTeams[j].ObjectivesCompleted >= requiredChildCompletions
		if ifinished && jfinished {
			return rankedTeams[i].LatestTimestamp < rankedTeams[j].LatestTimestamp
		}
		if ifinished && !jfinished {
			return true
		}
		if !ifinished && jfinished {
			return false
		}
		if rankedTeams[i].ObjectivesCompleted == rankedTeams[j].ObjectivesCompleted {
			return rankedTeams[i].LatestTimestamp < rankedTeams[j].LatestTimestamp
		}
		return rankedTeams[i].ObjectivesCompleted > rankedTeams[j].ObjectivesCompleted
	})
	for i, completion := range rankedTeams {
		if scoreMap[completion.TeamId] == nil || scoreMap[completion.TeamId][objective.Id] == nil || scoreMap[completion.TeamId][objective.Id].PresetCompletions[scoringRule.Id] == nil {
			continue
		}
		comp := scoreMap[completion.TeamId][objective.Id].PresetCompletions[scoringRule.Id]
		comp.Number = completion.ObjectivesCompleted
		if completion.ObjectivesCompleted >= requiredChildCompletions {
			comp.Finished = true
			comp.Points = int(scoringRule.Points.Get(i))
			comp.Rank = i + 1
			comp.Timestamp = time.Unix(0, completion.LatestTimestamp)
		}
	}
	return nil
}

func handleChildRankingByNumber(objective *repository.Objective, scoringRule *repository.ScoringRule, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	teamCompletions := make(map[int]*TeamCompletion)
	for teamId := range scoreMap {
		teamCompletions[teamId] = &TeamCompletion{TeamId: teamId}
	}
	for _, child := range objective.Children {
		for teamId, objectiveScores := range scoreMap {
			childScore := objectiveScores[child.Id]
			if childScore != nil {
				maxNum := 0
				latest := time.Time{}
				for _, comp := range childScore.PresetCompletions {
					if comp.Number > maxNum {
						maxNum = comp.Number
						if comp.Timestamp.After(latest) {
							latest = comp.Timestamp
						}
					}
				}
				teamCompletions[teamId].ObjectivesCompleted += maxNum
				teamCompletions[teamId].LatestTimestamp = max(teamCompletions[teamId].LatestTimestamp, latest.UnixNano())
			}
		}
	}
	rankedTeams := utils.Values(teamCompletions)
	rankfun := func(i, j int) bool {
		if rankedTeams[i].ObjectivesCompleted == rankedTeams[j].ObjectivesCompleted {
			return rankedTeams[i].LatestTimestamp < rankedTeams[j].LatestTimestamp
		}
		return rankedTeams[i].ObjectivesCompleted > rankedTeams[j].ObjectivesCompleted
	}
	sort.Slice(rankedTeams, rankfun)
	rank := 1
	for i, completion := range rankedTeams {
		if scoreMap[completion.TeamId] == nil || scoreMap[completion.TeamId][objective.Id] == nil || scoreMap[completion.TeamId][objective.Id].PresetCompletions[scoringRule.Id] == nil {
			continue
		}
		comp := scoreMap[completion.TeamId][objective.Id].PresetCompletions[scoringRule.Id]
		comp.Number = completion.ObjectivesCompleted
		if comp.Number == 0 {
			continue
		}
		comp.Points = int(scoringRule.Points.Get(rank - 1))
		comp.Rank = rank
		if i+1 < len(rankedTeams) && (rankfun(i, i+1) != rankfun(i+1, i)) {
			rank++
		}
	}
	return nil
}
