package scoring

import (
	"bpl/metrics"
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"log"
	"sort"
	"time"

	"gorm.io/gorm"
)

type ObjectiveIdTeamId struct {
	ObjectiveId int
	TeamId      int
}

type FreshMatches map[ObjectiveIdTeamId]bool

func (f FreshMatches) contains(match *Match) bool {
	return f[ObjectiveIdTeamId{ObjectiveId: match.ObjectiveId, TeamId: match.TeamId}]
}

var cacheDuration = 1 * time.Minute
var earliestMatchesCache = make(map[int][]*Match)
var nextCacheInvalidation = time.Now().Add(cacheDuration)

type Match struct {
	ObjectiveId int
	Number      int
	Timestamp   time.Time
	UserId      int
	TeamId      int
	Finished    bool
}

type TeamMatches = map[int]*Match

type ObjectiveTeamMatches = map[int]TeamMatches

type AggregationHandler func(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error)

var aggregationMap = map[repository.CountingMethod]AggregationHandler{
	repository.CountingMethodFirstFreshCompletion: handleEarliestFreshItem,
	repository.CountingMethodFirstCompletion:      handleEarliest,
	repository.CountingMethodLatestValue:          handleLatest,
	repository.CountingMethodHighestValue:         handleMaximum,
	repository.CountingMethodLowestValue:          handleMinimum,
	repository.CountingMethodValueChangeInWindow:  handleDifferenceBetween,
}

func AggregateMatches(db *gorm.DB, event *repository.Event, objectives []*repository.Objective) ObjectiveTeamMatches {
	// err := calculateDerivedMatches(db, objectives)
	// if err != nil {
	// 	log.Print("Error calculating derived matches: ", err)
	// }
	totalTime := time.Now()
	aggregations := make(ObjectiveTeamMatches)
	teamIds := utils.Map(event.Teams, func(team *repository.Team) int {
		return team.Id
	})
	objectiveMap := make(map[int]repository.Objective)
	objectivesByAggregation := make(map[repository.CountingMethod][]*repository.Objective)
	for _, objective := range objectives {
		objectivesByAggregation[objective.CountingMethod] = append(objectivesByAggregation[objective.CountingMethod], objective)
		objectiveMap[objective.Id] = *objective
		aggregations[objective.Id] = make(TeamMatches)
	}
	for _, aggregation := range []repository.CountingMethod{
		repository.CountingMethodFirstFreshCompletion,
		repository.CountingMethodFirstCompletion,
		repository.CountingMethodHighestValue,
		repository.CountingMethodLowestValue,
		repository.CountingMethodLatestValue,
		repository.CountingMethodValueChangeInWindow,
	} {
		if handler, ok := aggregationMap[aggregation]; ok {
			t := time.Now()
			matches, err := handler(db, objectivesByAggregation[aggregation], teamIds, event.Id)
			if err != nil {
				log.Print(err)
				continue
			}
			for _, match := range matches {
				// todo: maybe move this into the aggregation steps
				if aggregation != repository.CountingMethodValueChangeInWindow {
					match.Finished = objectiveMap[match.ObjectiveId].RequiredAmount <= match.Number
				}
				aggregations[match.ObjectiveId][match.TeamId] = match
			}
			metrics.ScoreAggregationDuration.WithLabelValues(string(aggregation)).Set(time.Since(t).Seconds())
		}
	}
	metrics.ScoreAggregationDuration.WithLabelValues("total").Set(time.Since(totalTime).Seconds())
	return aggregations
}

func calculateDerivedMatches(db *gorm.DB, objectives []*repository.Objective) error {
	matches := []*repository.ObjectiveMatch{}
	childSumObjectives := utils.Filter(objectives, func(objective *repository.Objective) bool {
		return objective.TrackedValue == repository.TrackedValueChildNumerValueSum
	})
	m, err := calculateLatestChildNumerValueSum(db, childSumObjectives)
	if err != nil {
		return err
	}
	matches = append(matches, m...)

	childCompletionObjectives := utils.Filter(objectives, func(objective *repository.Objective) bool {
		return objective.TrackedValue == repository.TrackedValueCompletedChildObjectiveCount
	})
	m, err = calculateLatestChildCompletionNumber(db, childCompletionObjectives)
	if err != nil {
		return err
	}
	matches = append(matches, m...)
	if len(matches) == 0 {
		return nil
	}
	return db.CreateInBatches(&matches, 1000).Error
}

func buildChildMaps(objectives []*repository.Objective) ([]int, map[int]int) {
	childIds := make([]int, 0)
	childToParent := make(map[int]int)
	for _, objective := range objectives {
		for _, child := range objective.Children {
			childIds = append(childIds, child.Id)
			childToParent[child.Id] = objective.Id
		}
	}
	return childIds, childToParent
}

func getOrCreateMatch(matches map[ObjectiveIdTeamId]*repository.ObjectiveMatch, parentId int, teamId int, timestamp time.Time) *repository.ObjectiveMatch {
	key := ObjectiveIdTeamId{ObjectiveId: parentId, TeamId: teamId}
	if matches[key] == nil {
		matches[key] = &repository.ObjectiveMatch{
			ObjectiveId: parentId,
			TeamId:      teamId,
			Timestamp:   timestamp,
		}
	}
	return matches[key]
}

func flattenMatches(matches map[ObjectiveIdTeamId]*repository.ObjectiveMatch) []*repository.ObjectiveMatch {
	flatMatches := make([]*repository.ObjectiveMatch, 0, len(matches))
	for _, match := range matches {
		flatMatches = append(flatMatches, match)
	}
	return flatMatches
}

func calculateLatestChildCompletionNumber(db *gorm.DB, objectives []*repository.Objective) ([]*repository.ObjectiveMatch, error) {
	childIds, childToParent := buildChildMaps(objectives)
	if len(childIds) == 0 {
		return make([]*repository.ObjectiveMatch, 0), nil
	}
	query := `
	SELECT
		objective_id,
		team_id,
		objectives.required_amount <= MAX(number) AS finished
	FROM objective_matches AS match
	JOIN objectives ON objectives.id = match.objective_id
	WHERE
		match.objective_id IN @objectiveIds
	GROUP BY
		match.objective_id, match.team_id, objectives.required_amount
	`
	result := make([]struct {
		ObjectiveId int
		TeamId      int
		Finished    bool
	}, 0)
	err := db.Raw(query, map[string]any{"objectiveIds": childIds}).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	matches := make(map[ObjectiveIdTeamId]*repository.ObjectiveMatch)
	for _, row := range result {
		parentId, ok := childToParent[row.ObjectiveId]
		if !ok {
			continue
		}
		match := getOrCreateMatch(matches, parentId, row.TeamId, time.Now())
		if row.Finished {
			match.Number += 1
		}
	}
	return flattenMatches(matches), nil
}

func calculateLatestChildNumerValueSum(db *gorm.DB, objectives []*repository.Objective) ([]*repository.ObjectiveMatch, error) {
	childIds, childToParent := buildChildMaps(objectives)
	if len(childIds) == 0 {
		return make([]*repository.ObjectiveMatch, 0), nil
	}
	query := `
	SELECT
		objective_id,
		team_id,
		MAX(number) AS number
	FROM
		objective_matches AS match
	WHERE
		match.objective_id IN @objectiveIds
	GROUP BY
		match.objective_id, match.team_id
	`
	result := make([]struct {
		ObjectiveId int
		TeamId      int
		Number      int
	}, 0)
	err := db.Raw(query, map[string]any{"objectiveIds": childIds}).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	matches := make(map[ObjectiveIdTeamId]*repository.ObjectiveMatch)
	for _, row := range result {
		parentId, ok := childToParent[row.ObjectiveId]
		if !ok {
			continue
		}
		match := getOrCreateMatch(matches, parentId, row.TeamId, time.Now())
		match.Number += row.Number
	}
	return flattenMatches(matches), nil
}

func getObjectiveIds(objectives []*repository.Objective) []int {
	return utils.Map(objectives, func(objective *repository.Objective) int {
		return objective.Id
	})
}

func handleEarliest(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	if nextCacheInvalidation.Before(time.Now()) {
		earliestMatchesCache = make(map[int][]*Match)
		nextCacheInvalidation = time.Now().Add(cacheDuration)
	}
	unfinishedObjectiveIds := make([]int, 0)
	existingMatches := make([]*Match, 0)
	objectiveMap := make(map[int]repository.Objective)
	for _, objective := range objectives {
		objectiveMap[objective.Id] = *objective
		existing, ok := earliestMatchesCache[objective.Id]
		if ok {
			existingMatches = append(existingMatches, existing...)
		} else {
			unfinishedObjectiveIds = append(unfinishedObjectiveIds, objective.Id)
		}
	}
	query := `
	WITH ranked_matches AS (
		SELECT 
			match.objective_id,
			match.number,
			match.timestamp,
			match.user_id, 
			match.team_id,
			match.number >= objectives.required_amount AS finished,
			RANK() OVER (
				PARTITION BY match.objective_id, match.team_id
				ORDER BY
					CASE 
						WHEN match.number >= objectives.required_amount THEN 1000000
						ELSE match.number
					END DESC,
					match.timestamp ASC,
					match.number DESC,
					match.user_id ASC
			) AS rank
		FROM 
			objective_matches as match
		JOIN 
			objectives ON objectives.id = match.objective_id AND objectives.id IN @objectiveIds
		WHERE 
			match.objective_id IN @objectiveIds
	)
	SELECT 
		*
	FROM 
		ranked_matches
	WHERE 
		rank = 1;
	`
	matches := make([]*Match, 0)
	err := db.Raw(query, map[string]any{"objectiveIds": unfinishedObjectiveIds}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	matches = append(matches, existingMatches...)
	newCache := make(map[int][]*Match)
	for _, match := range matches {
		if match.Finished {
			match.Number = objectiveMap[match.ObjectiveId].RequiredAmount
			newCache[match.ObjectiveId] = append(newCache[match.ObjectiveId], match)
		}
	}
	// for id, objectives := range newCache {
	// 	if len(objectives) == len(teamIds) {
	// 		earliestMatchesCache[id] = objectives
	// 	}
	// }
	return matches, nil
}

func handleEarliestFreshItem(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	freshMatches, err := getFreshMatches(db, objectives, teamIds, eventId)
	if err != nil {
		return nil, err
	}
	firstMatches, err := handleEarliest(db, objectives, teamIds, eventId)
	if err != nil {
		return nil, err
	}
	matches := make([]*Match, 0)
	for _, match := range firstMatches {
		if freshMatches.contains(match) {
			matches = append(matches, match)
		}
	}
	return matches, nil
}

func getExtremeQuery(aggregationType repository.CountingMethod) (string, error) {
	var order string
	switch aggregationType {
	case repository.CountingMethodHighestValue:
		order = "DESC"
	case repository.CountingMethodLowestValue:
		order = "ASC"
	default:
		return "", fmt.Errorf("invalid aggregation type")
	}
	return fmt.Sprintf(`
    SELECT DISTINCT ON (objective_id, team_id)
        objective_id,
        team_id,
        user_id,
        number,
        timestamp
    FROM objective_matches
    WHERE objective_id IN @objectiveIds
    ORDER BY objective_id, team_id, number %s, timestamp ASC
	`, order), nil

}

func handleMaximum(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	t := time.Now()
	query, err := getExtremeQuery(repository.CountingMethodHighestValue)
	if err != nil {
		return nil, err
	}
	matches := make([]*Match, 0)
	err = db.Raw(query, map[string]any{"objectiveIds": getObjectiveIds(objectives)}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	metrics.ScoreAggregationDuration.WithLabelValues("handleMaximum").Set(time.Since(t).Seconds())
	return matches, nil
}

func handleMinimum(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	query, err := getExtremeQuery(repository.CountingMethodLowestValue)
	if err != nil {
		return nil, err
	}
	matches := make([]*Match, 0)
	err = db.Raw(query,
		map[string]any{"objectiveIds": getObjectiveIds(objectives)}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func handleLatestSum(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	query := `
    WITH latest AS (
        SELECT
            match.objective_id,
            match.user_id,
            MAX(timestamp) AS timestamp
        FROM
            objective_matches AS match
        WHERE
            match.objective_id IN @objectiveIds
        GROUP BY
            match.objective_id, match.user_id 
    )		
    SELECT
        match.objective_id,
        match.team_id,
        SUM(match.number) AS number,
        MAX(match.timestamp) AS timestamp
    FROM
        objective_matches AS match
    JOIN
        latest ON latest.objective_id = match.objective_id 
        AND (latest.user_id = match.user_id OR (latest.user_id IS NULL AND match.user_id IS NULL))
        AND latest.timestamp = match.timestamp
    GROUP BY
        match.objective_id, match.team_id
    `
	matches := make([]*Match, 0)
	err := db.Raw(query, map[string]any{"objectiveIds": getObjectiveIds(objectives)}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func handleLatest(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	query := `
	WITH latest AS (
		SELECT
			match.objective_id,
			match.team_id,
			MAX(timestamp) AS timestamp
		FROM
			objective_matches AS match
		WHERE
			match.objective_id IN @objectiveIds
		GROUP BY
			match.objective_id, match.team_id 
	)		
	SELECT
		match.objective_id,
		match.team_id,
		match.number,
		match.timestamp,
		match.user_id
	FROM
		objective_matches AS match
	JOIN
		latest ON latest.objective_id = match.objective_id AND latest.team_id = match.team_id
	`
	matches := make([]*Match, 0)
	err := db.Raw(query, map[string]any{"objectiveIds": getObjectiveIds(objectives)}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func getFreshMatches(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) (FreshMatches, error) {
	// todo: might want to also check if the match finishes the objective
	t := time.Now()
	query := `
    WITH latest AS (
        SELECT DISTINCT ON (stash_id) id
        FROM stash_changes
        WHERE event_id = @eventId
        ORDER BY stash_id, id DESC
    )
    SELECT
        objective_matches.objective_id,
        objective_matches.team_id
    FROM objective_matches
	JOIN latest ON objective_matches.stash_change_id = latest.id AND
     	objective_matches.objective_id IN @objectiveIds
    GROUP BY
        objective_matches.objective_id,
        objective_matches.team_id
    `
	matchList := make([]ObjectiveIdTeamId, 0)
	result := db.Raw(query, map[string]any{"objectiveIds": getObjectiveIds(objectives), "eventId": eventId}).Scan(&matchList)
	if result.Error != nil {
		return nil, result.Error
	}
	freshMatches := make(FreshMatches)
	for _, id := range matchList {
		freshMatches[id] = true
	}
	metrics.ScoreAggregationDuration.WithLabelValues("getFreshMatches").Set(time.Since(t).Seconds())
	return freshMatches, nil
}

func handleDifferenceBetween(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	query := `
	SELECT
		match.objective_id,
		match.team_id,
		match.user_id,
		match.number,
		match.timestamp
	FROM
		objective_matches AS match
	WHERE
		match.objective_id IN @objectiveIds
	ORDER BY
		match.objective_id, match.timestamp
	`
	objectiveMap := make(map[int]repository.Objective)
	for _, objective := range objectives {
		objectiveMap[objective.Id] = *objective
	}
	preMatches := make([]*Match, 0)
	err := db.Raw(query, map[string]any{"objectiveIds": getObjectiveIds(objectives)}).Scan(&preMatches).Error
	if err != nil {
		return nil, err
	}
	matches := make([]*Match, 0)
	for _, objective := range objectives {
		if objective.ValidFrom == nil || objective.ValidTo == nil {
			fmt.Printf("VALUE_CHANGE_IN_WINDOW objective %d does not have timestamps set\n", objective.Id)
			continue
		}

		matches = append(matches, getDifferencesBetweenTimestamps(objective, preMatches, teamIds)...)
	}
	return matches, nil

}

func getDifferencesBetweenTimestamps(objective *repository.Objective, preMatches []*Match, teamIds []int) []*Match {
	matches := []*Match{}
	for _, teamId := range teamIds {
		objectiveMatches := utils.Filter(preMatches, func(match *Match) bool {
			return match.ObjectiveId == objective.Id && match.TeamId == teamId
		})
		sort.Slice(objectiveMatches, func(i, j int) bool {
			return objectiveMatches[i].Timestamp.Before(objectiveMatches[j].Timestamp)
		})
		if len(objectiveMatches) == 0 {
			continue
		}
		minMatch := &Match{
			Timestamp: objectiveMatches[0].Timestamp.Add(-time.Hour),
			Number:    0,
		}
		maxMatch := objectiveMatches[0]
		for _, match := range objectiveMatches {
			if match.Timestamp.Before(*objective.ValidFrom) && minMatch.Timestamp.Before(match.Timestamp) {
				minMatch = match
			}
			if match.Timestamp.Before(*objective.ValidTo) && maxMatch.Timestamp.Before(match.Timestamp) {
				maxMatch = match
			}
		}
		matches = append(matches, &Match{
			ObjectiveId: objective.Id,
			Number:      maxMatch.Number - minMatch.Number,
			Timestamp:   maxMatch.Timestamp,
			UserId:      0,
			TeamId:      maxMatch.TeamId,
			Finished:    time.Now().After(*objective.ValidTo),
		})
	}
	return matches
}
