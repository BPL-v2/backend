package scoring

import (
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ========== Score methods ==========

func TestScoreFinished(t *testing.T) {
	t.Run("all presets finished", func(t *testing.T) {
		s := &Score{
			PresetCompletions: map[int]*PresetCompletion{
				1: {Finished: true},
				2: {Finished: true},
			},
		}
		assert.True(t, s.Finished())
	})

	t.Run("one preset not finished", func(t *testing.T) {
		s := &Score{
			PresetCompletions: map[int]*PresetCompletion{
				1: {Finished: true},
				2: {Finished: false},
			},
		}
		assert.False(t, s.Finished())
	})

	t.Run("empty presets is not finished", func(t *testing.T) {
		// A score with no PresetCompletions hasn't actually completed anything yet
		// (e.g. a CATEGORY objective with no scoring rules of its own and no children
		// populated yet). It must not be vacuously treated as finished, otherwise
		// handleChildBonus would award bonus points to it before anything happened.
		s := &Score{PresetCompletions: map[int]*PresetCompletion{}}
		assert.False(t, s.Finished())
	})
}

func TestScoreTimestamp(t *testing.T) {
	t.Run("returns latest timestamp when finished", func(t *testing.T) {
		t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
		t2 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		s := &Score{
			PresetCompletions: map[int]*PresetCompletion{
				1: {Finished: true, Timestamp: t1},
				2: {Finished: true, Timestamp: t2},
			},
		}
		assert.Equal(t, t2, s.Timestamp())
	})

	t.Run("returns zero time when not finished", func(t *testing.T) {
		s := &Score{
			PresetCompletions: map[int]*PresetCompletion{
				1: {Finished: true, Timestamp: time.Now()},
				2: {Finished: false},
			},
		}
		assert.True(t, s.Timestamp().IsZero())
	})

	t.Run("single preset", func(t *testing.T) {
		t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
		s := &Score{
			PresetCompletions: map[int]*PresetCompletion{
				1: {Finished: true, Timestamp: t1},
			},
		}
		assert.Equal(t, t1, s.Timestamp())
	})
}

func TestScorePoints(t *testing.T) {
	t.Run("sums all preset points plus bonus", func(t *testing.T) {
		s := &Score{
			BonusPoints: 5,
			PresetCompletions: map[int]*PresetCompletion{
				1: {Points: 10},
				2: {Points: 20},
			},
		}
		assert.Equal(t, 35, s.Points())
	})

	t.Run("bonus only no presets", func(t *testing.T) {
		s := &Score{
			BonusPoints:       7,
			PresetCompletions: map[int]*PresetCompletion{},
		}
		assert.Equal(t, 7, s.Points())
	})

	t.Run("zero points", func(t *testing.T) {
		s := &Score{
			PresetCompletions: map[int]*PresetCompletion{
				1: {Points: 0},
			},
		}
		assert.Equal(t, 0, s.Points())
	})
}

func TestScoreCanShowTo(t *testing.T) {
	t.Run("same team can always see", func(t *testing.T) {
		s := &Score{
			TeamId:       1,
			HideProgress: true,
			PresetCompletions: map[int]*PresetCompletion{
				1: {Finished: false},
			},
		}
		assert.True(t, s.CanShowTo(1))
	})

	t.Run("other team can see when finished", func(t *testing.T) {
		s := &Score{
			TeamId:       1,
			HideProgress: true,
			PresetCompletions: map[int]*PresetCompletion{
				1: {Finished: true},
			},
		}
		assert.True(t, s.CanShowTo(2))
	})

	t.Run("other team can see when progress not hidden", func(t *testing.T) {
		s := &Score{
			TeamId:       1,
			HideProgress: false,
			PresetCompletions: map[int]*PresetCompletion{
				1: {Finished: false},
			},
		}
		assert.True(t, s.CanShowTo(2))
	})

	t.Run("other team cannot see when hidden and not finished", func(t *testing.T) {
		s := &Score{
			TeamId:       1,
			HideProgress: true,
			PresetCompletions: map[int]*PresetCompletion{
				1: {Finished: false},
			},
		}
		assert.False(t, s.CanShowTo(2))
	})
}

func TestEvaluateAggregations(t *testing.T) {
	t.Run("evaluates leaf objective with presence scoring", func(t *testing.T) {
		presetId := 100
		objective := &repository.Objective{
			Id: 1,
			ScoringRules: []*repository.ScoringRule{
				{
					Id:       presetId,
					Points:   repository.ExtendingNumberSlice{10},
					RuleType: repository.FIXED_POINTS_ON_COMPLETION,
				},
			},
		}
		aggregations := make(ObjectiveTeamMatches)
		aggregations[1] = TeamMatches{
			1: &Match{TeamId: 1, UserId: 1, Finished: true, Timestamp: time.Now()},
		}
		scoreMap := map[int]map[int]*Score{
			1: {
				1: &Score{
					ObjectiveId: 1,
					TeamId:      1,
					PresetCompletions: map[int]*PresetCompletion{
						presetId: {ObjectiveId: 1},
					},
				},
			},
		}
		err := EvaluateAggregations(objective, aggregations, scoreMap)
		assert.NoError(t, err)
		assert.Equal(t, 10, scoreMap[1][1].PresetCompletions[presetId].Points)
		assert.True(t, scoreMap[1][1].PresetCompletions[presetId].Finished)
	})

	t.Run("evaluates children recursively", func(t *testing.T) {
		presetId := 100
		childPresetId := 200
		child := &repository.Objective{
			Id: 2,
			ScoringRules: []*repository.ScoringRule{
				{
					Id:       childPresetId,
					Points:   repository.ExtendingNumberSlice{5},
					RuleType: repository.FIXED_POINTS_ON_COMPLETION,
				},
			},
		}
		parent := &repository.Objective{
			Id:       1,
			Children: []*repository.Objective{child},
			ScoringRules: []*repository.ScoringRule{
				{
					Id:       presetId,
					Points:   repository.ExtendingNumberSlice{20},
					RuleType: repository.FIXED_POINTS_ON_COMPLETION,
				},
			},
		}
		aggregations := make(ObjectiveTeamMatches)
		aggregations[1] = TeamMatches{
			1: &Match{TeamId: 1, UserId: 1, Finished: true, Timestamp: time.Now()},
		}
		aggregations[2] = TeamMatches{
			1: &Match{TeamId: 1, UserId: 1, Finished: true, Timestamp: time.Now()},
		}
		scoreMap := map[int]map[int]*Score{
			1: {
				1: &Score{
					ObjectiveId: 1,
					TeamId:      1,
					PresetCompletions: map[int]*PresetCompletion{
						presetId: {ObjectiveId: 1},
					},
				},
				2: &Score{
					ObjectiveId: 2,
					TeamId:      1,
					PresetCompletions: map[int]*PresetCompletion{
						childPresetId: {ObjectiveId: 2},
					},
				},
			},
		}
		err := EvaluateAggregations(parent, aggregations, scoreMap)
		assert.NoError(t, err)
		assert.Equal(t, 5, scoreMap[1][2].PresetCompletions[childPresetId].Points)
		assert.Equal(t, 20, scoreMap[1][1].PresetCompletions[presetId].Points)
	})

	t.Run("no scoring presets is fine", func(t *testing.T) {
		objective := &repository.Objective{
			Id:           1,
			ScoringRules: []*repository.ScoringRule{},
		}
		err := EvaluateAggregations(objective, make(ObjectiveTeamMatches), make(map[int]map[int]*Score))
		assert.NoError(t, err)
	})

	t.Run("unknown scoring method is silently skipped", func(t *testing.T) {
		objective := &repository.Objective{
			Id: 1,
			ScoringRules: []*repository.ScoringRule{
				{
					Id:       1,
					Points:   repository.ExtendingNumberSlice{10},
					RuleType: "UNKNOWN_METHOD",
				},
			},
		}
		err := EvaluateAggregations(objective, make(ObjectiveTeamMatches), make(map[int]map[int]*Score))
		assert.NoError(t, err)
	})
}

func TestHandlePresence(t *testing.T) {
	// This tests FIXED_POINTS_ON_COMPLETION scoring where teams get points simply for completing an objective
	// Only teams with Finished=true should receive points
	presetId := 100
	objective := &repository.Objective{
		Id: 1,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{10},
			},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	match1 := Match{
		TeamId:    1,
		Number:    1,
		UserId:    1,
		Finished:  false,
		Timestamp: now.Add(-24 * time.Hour),
	}
	match2 := Match{
		TeamId:    2,
		Number:    2,
		UserId:    2,
		Finished:  true,
		Timestamp: now.Add(-24 * time.Hour),
	}

	aggregations[objective.Id][1] = &match1
	aggregations[objective.Id][2] = &match2

	scoreMap := make(map[int]map[int]*Score)
	for teamId := range aggregations[objective.Id] {
		scoreMap[teamId] = make(map[int]*Score)
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handlePresence(objective, objective.ScoringRules[0], aggregations, scoreMap)
	assert.NoError(t, err, "handlePresence should not return an error")

	// Verify only the finished team gets points
	assert.Equal(t, 0, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 (not finished) should have 0 points")
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 (finished) should have 10 points")
}

func TestHandlePointsFromValue(t *testing.T) {
	// This tests POINTS_BY_VALUE scoring where points are calculated by multiplying
	// the match Number (e.g., item count, completion %) by a point value
	// Also tests that PointCap limits the maximum points
	value := 10.0
	presetId := 100
	objective := &repository.Objective{
		Id: 1,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:       presetId,
				Points:   repository.ExtendingNumberSlice{value},
				PointCap: 500,
			},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	match1 := Match{
		TeamId:    1,
		Number:    1,
		UserId:    1,
		Finished:  false,
		Timestamp: now.Add(-24 * time.Hour),
	}
	match2 := Match{
		TeamId:    2,
		Number:    2,
		UserId:    2,
		Finished:  true,
		Timestamp: now.Add(-24 * time.Hour),
	}
	match3 := Match{
		TeamId:    3,
		Number:    100,
		UserId:    3,
		Finished:  true,
		Timestamp: now.Add(-24 * time.Hour),
	}

	aggregations[objective.Id][1] = &match1
	aggregations[objective.Id][2] = &match2
	aggregations[objective.Id][3] = &match3

	scoreMap := make(map[int]map[int]*Score)
	for teamId := range aggregations[objective.Id] {
		scoreMap[teamId] = make(map[int]*Score)
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handlePointsFromValue(objective, objective.ScoringRules[0], aggregations, scoreMap)
	assert.NoError(t, err, "handlePointsFromValue should not return an error")

	// Verify points are calculated correctly and capped when necessary
	assert.Equal(t, int(value*float64(match1.Number)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 points should be value * Number (10 * 1 = 10)")
	assert.Equal(t, int(value*float64(match2.Number)), scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 points should be value * Number (10 * 2 = 20)")
	assert.Equal(t, objective.ScoringRules[0].PointCap, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 points should be capped at PointCap (500)")
}

func TestHandleRankedTime(t *testing.T) {
	// This tests RANK_BY_COMPLETION_TIME scoring where teams are ranked by completion time
	// Earlier completion = better rank. Ties result in same rank. Unfinished teams get 0 points.
	presetId := 100
	objective := &repository.Objective{
		Id: 1,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{10, 5},
			},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	// Teams 1 & 2: Finished at -24h (tied) -> rank 1, 10 points each
	// Team 3: Finished at -23h -> rank 2, 5 points
	// Team 4: Finished at -22h -> rank 3, 5 points (extends from last)
	// Team 5: Not finished -> rank 0, 0 points
	match1 := Match{TeamId: 1, UserId: 1, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match2 := Match{TeamId: 2, UserId: 2, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match3 := Match{TeamId: 3, UserId: 3, Finished: true, Timestamp: now.Add(-23 * time.Hour)}
	match4 := Match{TeamId: 4, UserId: 4, Finished: true, Timestamp: now.Add(-22 * time.Hour)}
	match5 := Match{TeamId: 5, UserId: 5, Finished: false, Timestamp: now.Add(-21 * time.Hour)}

	aggregations[objective.Id][1] = &match1
	aggregations[objective.Id][2] = &match2
	aggregations[objective.Id][3] = &match3
	aggregations[objective.Id][4] = &match4
	aggregations[objective.Id][5] = &match5

	scoreMap := make(map[int]map[int]*Score)
	for teamId := range aggregations[objective.Id] {
		scoreMap[teamId] = make(map[int]*Score)
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleRankedTime(objective, objective.ScoringRules[0], aggregations, scoreMap)
	assert.NoError(t, err, "handleRankedTime should not return an error")

	// Verify ranks and points are assigned correctly based on completion time
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1 (finished at -24h, tied)")
	assert.Equal(t, 10, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 10 points (rank 1)")
	assert.Equal(t, 1, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 1 (finished at -24h, tied)")
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 10 points (rank 1)")
	assert.Equal(t, 2, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 2 (finished at -23h)")
	assert.Equal(t, 5, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 5 points (rank 2)")
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank, "Team 4 should have rank 3 (finished at -22h)")
	assert.Equal(t, 5, scoreMap[4][objective.Id].PresetCompletions[presetId].Points, "Team 4 should have 5 points (rank 3, extends)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank, "Team 5 should have rank 0 (not finished)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Points, "Team 5 should have 0 points (not finished)")
}
func TestHandleRankedValue(t *testing.T) {
	// This tests RANK_BY_HIGHEST_VALUE scoring where teams are ranked by their Number value (higher = better)
	// Used for objectives where a higher count/value is better (e.g., boss kills, items collected)
	presetId := 100
	objective := &repository.Objective{
		Id: 1,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{10, 5},
			},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	// Teams 1 & 2: Number=4 (tied) -> rank 1, 10 points each
	// Team 3: Number=3 -> rank 2, 5 points
	// Team 4: Number=2 -> rank 3, 5 points
	// Team 5: Not finished -> rank 0, 0 points
	match1 := Match{TeamId: 1, UserId: 1, Number: 4, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match2 := Match{TeamId: 2, UserId: 2, Number: 4, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match3 := Match{TeamId: 3, UserId: 3, Number: 3, Finished: true, Timestamp: now.Add(-23 * time.Hour)}
	match4 := Match{TeamId: 4, UserId: 4, Number: 2, Finished: true, Timestamp: now.Add(-22 * time.Hour)}
	match5 := Match{TeamId: 5, UserId: 5, Number: 1, Finished: false, Timestamp: now.Add(-21 * time.Hour)}

	aggregations[objective.Id][1] = &match1
	aggregations[objective.Id][2] = &match2
	aggregations[objective.Id][3] = &match3
	aggregations[objective.Id][4] = &match4
	aggregations[objective.Id][5] = &match5

	scoreMap := make(map[int]map[int]*Score)
	for teamId := range aggregations[objective.Id] {
		scoreMap[teamId] = make(map[int]*Score)
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleRankedValue(objective, objective.ScoringRules[0], aggregations, scoreMap)
	assert.NoError(t, err, "handleRankedValue should not return an error")

	// Verify ranks and points are assigned correctly based on Number value
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1 (Number=4, tied)")
	assert.Equal(t, 10, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 10 points (rank 1)")
	assert.Equal(t, 1, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 1 (Number=4, tied)")
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 10 points (rank 1)")
	assert.Equal(t, 2, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 2 (Number=3)")
	assert.Equal(t, 5, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 5 points (rank 2)")
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank, "Team 4 should have rank 3 (Number=2)")
	assert.Equal(t, 5, scoreMap[4][objective.Id].PresetCompletions[presetId].Points, "Team 4 should have 5 points (rank 3, extends)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank, "Team 5 should have rank 0 (not finished)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Points, "Team 5 should have 0 points (not finished)")
}
func TestHandleRankedReverse(t *testing.T) {
	// This tests RANK_BY_LOWEST_VALUE scoring where teams are ranked by Number value (lower = better)
	// Used for objectives where a lower value is better (e.g., fastest time, fewest deaths)
	presetId := 100
	objective := &repository.Objective{
		Id: 1,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{10, 5},
			},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	// Teams 1 & 2: Number=1 (lowest, tied) -> rank 1, 10 points each
	// Team 3: Number=2 -> rank 2, 5 points
	// Team 4: Number=3 -> rank 3, 5 points
	// Team 5: Not finished -> rank 0, 0 points
	match1 := Match{TeamId: 1, UserId: 1, Number: 1, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match2 := Match{TeamId: 2, UserId: 2, Number: 1, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match3 := Match{TeamId: 3, UserId: 3, Number: 2, Finished: true, Timestamp: now.Add(-23 * time.Hour)}
	match4 := Match{TeamId: 4, UserId: 4, Number: 3, Finished: true, Timestamp: now.Add(-22 * time.Hour)}
	match5 := Match{TeamId: 5, UserId: 5, Number: 4, Finished: false, Timestamp: now.Add(-21 * time.Hour)}

	aggregations[objective.Id][1] = &match1
	aggregations[objective.Id][2] = &match2
	aggregations[objective.Id][3] = &match3
	aggregations[objective.Id][4] = &match4
	aggregations[objective.Id][5] = &match5

	scoreMap := make(map[int]map[int]*Score)
	for teamId := range aggregations[objective.Id] {
		scoreMap[teamId] = make(map[int]*Score)
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleRankedReverse(objective, objective.ScoringRules[0], aggregations, scoreMap)
	assert.NoError(t, err, "handleRankedReverse should not return an error")

	// Verify ranks and points - lower Number values get better ranks
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1 (Number=1, lowest, tied)")
	assert.Equal(t, 10, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 10 points (rank 1)")
	assert.Equal(t, 1, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 1 (Number=1, lowest, tied)")
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 10 points (rank 1)")
	assert.Equal(t, 2, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 2 (Number=2)")
	assert.Equal(t, 5, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 5 points (rank 2)")
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank, "Team 4 should have rank 3 (Number=3)")
	assert.Equal(t, 5, scoreMap[4][objective.Id].PresetCompletions[presetId].Points, "Team 4 should have 5 points (rank 3, extends)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank, "Team 5 should have rank 0 (not finished)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Points, "Team 5 should have 0 points (not finished)")
}

func TestHandleChildBonus(t *testing.T) {
	// This tests BONUS_PER_CHILD_COMPLETION scoring where bonus points are awarded to child objectives
	// based on completion order. Earlier completions get higher bonuses.
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{10, 9, 5},
			},
		},
		Children: utils.Map([]int{1, 2, 3, 4, 5}, func(id int) *repository.Objective {
			return &repository.Objective{
				Id: id,
			}
		}),
	}
	now := time.Now()

	// Build scoreMap with parent and child scores
	// Team 1: Completes 5 children -> bonuses: 10, 9, 5, 5, 5 (by completion order)
	// Team 2: Completes 1 child -> bonus: 10
	scoreMap := make(map[int]map[int]*Score)
	childData := []struct {
		objId, teamId int
		timestamp     time.Time
		finished      bool
	}{
		{1, 1, now.Add(-24 * time.Hour), true},
		{2, 1, now.Add(-23 * time.Hour), true},
		{3, 1, now.Add(-22 * time.Hour), true},
		{4, 1, now.Add(-21 * time.Hour), true},
		{5, 1, now.Add(-20 * time.Hour), true},
		{1, 2, now.Add(-20 * time.Hour), true},
	}

	// Initialize all teams with all child objectives (even if not finished)
	for teamId := 1; teamId <= 2; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 5; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Now update with actual finished data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Add parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleChildBonus(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleChildBonus should not return an error")

	// Verify BonusPoints were added to child scores in order of completion
	assert.Equal(t, 10, scoreMap[1][1].BonusPoints, "Team 1, child 1 should have 10 bonus (1st completed)")
	assert.Equal(t, 9, scoreMap[1][2].BonusPoints, "Team 1, child 2 should have 9 bonus (2nd completed)")
	assert.Equal(t, 5, scoreMap[1][3].BonusPoints, "Team 1, child 3 should have 5 bonus (3rd completed)")
	assert.Equal(t, 5, scoreMap[1][4].BonusPoints, "Team 1, child 4 should have 5 bonus (4th completed, extends)")
	assert.Equal(t, 5, scoreMap[1][5].BonusPoints, "Team 1, child 5 should have 5 bonus (5th completed, extends)")
	assert.Equal(t, 10, scoreMap[2][1].BonusPoints, "Team 2, child 1 should have 10 bonus (1st completed)")
}

func TestHandleChildRanking(t *testing.T) {
	// This tests RANK_BY_CHILD_COMPLETION_TIME scoring where teams are ranked by completing ALL child objectives
	// Only teams that complete all children get points. Ranking is by completion time of last child.
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{20, 10},
			},
		},
		Children: utils.Map([]int{1, 2}, func(id int) *repository.Objective {
			return &repository.Objective{
				Id: id,
			}
		}),
	}
	now := time.Now()

	// Build scoreMap
	scoreMap := make(map[int]map[int]*Score)
	childData := []struct {
		objId, teamId int
		timestamp     time.Time
		finished      bool
	}{
		{1, 1, now.Add(-23 * time.Hour), true},
		{2, 1, now.Add(-23 * time.Hour), true},
		{1, 2, now.Add(-22 * time.Hour), true},
		{2, 2, now.Add(-24 * time.Hour), true},
		{1, 3, now.Add(-20 * time.Hour), true},
	}

	// Initialize all teams with all child objectives
	for teamId := 1; teamId <= 3; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 2; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Now update with actual finished data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Add parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleChildRankingByTime(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleChildRankingByTime should not return an error")

	// Verify only teams that completed all children get points
	assert.Equal(t, 20, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 20 points (completed all children, rank 1)")
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 10 points (completed all children, rank 2)")
	assert.Equal(t, 0, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 0 points (didn't complete all children)")

}

func TestHandleBingoBoardHorizontal(t *testing.T) {
	// This tests BINGO_BOARD_RANKING scoring with a horizontal line completion (row 0)
	// Team completes objectives 1, 2, 3 which form the first row of a 3x3 grid
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
	}
	children := []*repository.Objective{}
	for i := range 3 {
		for j := range 3 {
			children = append(children, &repository.Objective{Id: i*3 + j + 1, Extra: fmt.Sprintf("%d,%d", i, j)})
		}
	}
	objective.Children = children

	now := time.Now()
	childData := []struct {
		objId     int
		timestamp time.Time
	}{
		{1, now.Add(-24 * time.Hour)},
		{2, now.Add(-23 * time.Hour)},
		{3, now.Add(-22 * time.Hour)},
	}

	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)

	for _, data := range childData {
		scoreMap[1][data.objId] = &Score{
			ObjectiveId: data.objId,
			TeamId:      1,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: data.objId,
					Finished:    true,
					Timestamp:   data.timestamp,
				},
			},
		}
	}

	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	assert.Equal(t, int(objective.ScoringRules[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should receive first place points for completing horizontal bingo")
	assert.Equal(t, childData[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Timestamp should match the last completed objective in the bingo line")
}

func TestGetBingoVertical(t *testing.T) {
	// This tests BINGO_BOARD_RANKING scoring with a vertical line completion (column 0)
	// Team completes objectives 1, 4, 7 which form the first column of a 3x3 grid
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
	}
	children := []*repository.Objective{}
	for i := range 3 {
		for j := range 3 {
			children = append(children, &repository.Objective{Id: i*3 + j + 1, Extra: fmt.Sprintf("%d,%d", i, j)})
		}
	}
	objective.Children = children

	now := time.Now()
	childData := []struct {
		objId     int
		timestamp time.Time
	}{
		{1, now.Add(-24 * time.Hour)},
		{4, now.Add(-23 * time.Hour)},
		{7, now.Add(-22 * time.Hour)},
	}

	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)

	for _, data := range childData {
		scoreMap[1][data.objId] = &Score{
			ObjectiveId: data.objId,
			TeamId:      1,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: data.objId,
					Finished:    true,
					Timestamp:   data.timestamp,
				},
			},
		}
	}

	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	// Verify vertical bingo completion is detected correctly
	assert.Equal(t, int(objective.ScoringRules[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should receive first place points for completing vertical bingo")
	assert.Equal(t, childData[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Timestamp should match the last completed objective in the bingo line")
}

func TestHandleBingoBoardDiagonal(t *testing.T) {
	// This tests BINGO_BOARD_RANKING scoring with a diagonal line completion
	// Team completes objectives 1, 5, 9 which form the main diagonal of a 3x3 grid
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
	}
	children := []*repository.Objective{}
	for i := range 3 {
		for j := range 3 {
			children = append(children, &repository.Objective{Id: i*3 + j + 1, Extra: fmt.Sprintf("%d,%d", i, j)})
		}
	}
	objective.Children = children

	now := time.Now()
	childData := []struct {
		objId     int
		timestamp time.Time
	}{
		{1, now.Add(-24 * time.Hour)},
		{5, now.Add(-23 * time.Hour)},
		{9, now.Add(-22 * time.Hour)},
	}

	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)

	for _, data := range childData {
		scoreMap[1][data.objId] = &Score{
			ObjectiveId: data.objId,
			TeamId:      1,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: data.objId,
					Finished:    true,
					Timestamp:   data.timestamp,
				},
			},
		}
	}

	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	// Verify diagonal bingo completion is detected correctly
	assert.Equal(t, int(objective.ScoringRules[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should receive first place points for completing diagonal bingo")
	assert.Equal(t, childData[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Timestamp should match the last completed objective in the bingo line")
}

func TestHandleBingoBoardCorrectTime(t *testing.T) {
	// This tests that BINGO_BOARD_RANKING correctly identifies the completion timestamp
	// The timestamp should be the LATEST child completion in the bingo line
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
	}
	children := []*repository.Objective{}
	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)

	var expectedTimestamp time.Time
	for i := range 3 {
		for j := range 3 {
			id := i*3 + j + 1
			children = append(children, &repository.Objective{Id: id, Extra: fmt.Sprintf("%d,%d", i, j)})
			timestamp := time.Now().Add(time.Duration(-id) * time.Hour)
			if id == 7 {
				expectedTimestamp = timestamp
			}
			scoreMap[1][id] = &Score{
				ObjectiveId: id,
				TeamId:      1,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: id,
						Finished:    true,
						Timestamp:   timestamp,
					},
				},
			}
		}
	}
	objective.Children = children

	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	// Verify the bingo timestamp matches the last completed child in the line (objective 7)
	assert.Equal(t, int(objective.ScoringRules[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should receive first place points for bingo")
	assert.Equal(t, expectedTimestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Timestamp should be the latest timestamp in the bingo line (objective 7)")
}

func TestHandleBingoBoardCorrectRanking(t *testing.T) {
	// This tests that BINGO_BOARD_RANKING correctly ranks teams based on completion time
	// Team with earlier completion of their bingo line should rank higher
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
	}
	// Team 1: Completes row with timestamps[1,2,3] -> last at timestamps[3]
	// Team 2: Completes row with timestamps[0,1,2] -> last at timestamps[2] (earlier)
	// Team 2 should rank first
	timestamps := utils.Map([]int{1, 2, 3, 4, 5, 6, 7, 8, 9}, func(i int) time.Time {
		return time.Now().Add(time.Duration(-i) * time.Hour)
	})
	children := []*repository.Objective{}
	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)
	scoreMap[2] = make(map[int]*Score)

	for i := range 3 {
		objId := i + 1
		scoreMap[1][objId] = &Score{
			ObjectiveId: objId,
			TeamId:      1,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: objId,
					Finished:    true,
					Timestamp:   timestamps[i+1],
				},
			},
		}
		scoreMap[2][objId] = &Score{
			ObjectiveId: objId,
			TeamId:      2,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: objId,
					Finished:    true,
					Timestamp:   timestamps[i],
				},
			},
		}
		for j := range 3 {
			id := i*3 + j + 1
			children = append(children, &repository.Objective{Id: id, Extra: fmt.Sprintf("%d,%d", i, j)})
		}
	}
	objective.Children = children

	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}
	scoreMap[2][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      2,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	// Verify teams are ranked correctly by their bingo completion times
	assert.Equal(t, int(objective.ScoringRules[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should receive first place points")
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1 (earlier bingo completion)")
	assert.Equal(t, int(objective.ScoringRules[0].Points.Get(1)), scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should receive second place points")
	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 2 (later bingo completion)")

	assert.Equal(t, timestamps[1].Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Team 1 timestamp should be timestamps[1] (latest in their bingo line)")
	assert.Equal(t, timestamps[0].Unix(), scoreMap[2][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Team 2 timestamp should be timestamps[0] (latest in their bingo line)")
}

func TestMultipleScoringRulesOnUmbrellaObjective(t *testing.T) {
	// This tests that an umbrella objective can have multiple scoring methods applied:
	// 1. RANK_BY_COMPLETION_TIME - ranks teams based on completion time of all children
	// 2. BONUS_PER_CHILD_COMPLETION - gives bonus points for each completed child

	rankedPresetId := 100
	bonusPresetId := 200

	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:       rankedPresetId,
				RuleType: repository.RANK_BY_CHILD_COMPLETION_TIME,
				Points:   repository.ExtendingNumberSlice{50, 30, 10}, // Points for ranking 1st, 2nd, 3rd
			},
			{
				Id:       bonusPresetId,
				RuleType: repository.BONUS_PER_CHILD_COMPLETION,
				Points:   repository.ExtendingNumberSlice{15, 10, 5}, // Bonus for 1st, 2nd, 3rd+ child completed
			},
		},
		Children: utils.Map([]int{1, 2, 3, 4}, func(id int) *repository.Objective {
			return &repository.Objective{Id: id}
		}),
	}

	now := time.Now()

	// Team 1: Completes all 4 children, finishes FIRST (earliest latest timestamp at -21h)
	// Team 2: Completes all 4 children, finishes SECOND (latest timestamp at -15h)
	// Team 3: Completes all 4 children, finishes THIRD (latest timestamp at -10h)
	childData := []struct {
		objId, teamId int
		timestamp     time.Time
		finished      bool
	}{
		// Team 1 - completes all 4, last one finishes at -21h
		{1, 1, now.Add(-24 * time.Hour), true},
		{2, 1, now.Add(-23 * time.Hour), true},
		{3, 1, now.Add(-22 * time.Hour), true},
		{4, 1, now.Add(-21 * time.Hour), true}, // Latest timestamp for team 1
		// Team 2 - completes all 4, last one finishes at -15h
		{1, 2, now.Add(-20 * time.Hour), true},
		{2, 2, now.Add(-18 * time.Hour), true},
		{3, 2, now.Add(-16 * time.Hour), true},
		{4, 2, now.Add(-15 * time.Hour), true}, // Latest timestamp for team 2
		// Team 3 - completes all 4, last one finishes at -10h
		{1, 3, now.Add(-14 * time.Hour), true},
		{2, 3, now.Add(-12 * time.Hour), true},
		{3, 3, now.Add(-11 * time.Hour), true},
		{4, 3, now.Add(-10 * time.Hour), true}, // Latest timestamp for team 3
	}

	// Initialize scoreMap for all teams and all child objectives
	scoreMap := make(map[int]map[int]*Score)
	for teamId := 1; teamId <= 3; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)

		// Initialize all child objectives for this team (even unfinished ones)
		for childId := 1; childId <= 4; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					rankedPresetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
					bonusPresetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Apply actual completion data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[rankedPresetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[rankedPresetId].Timestamp = data.timestamp
		scoreMap[data.teamId][data.objId].PresetCompletions[bonusPresetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[bonusPresetId].Timestamp = data.timestamp
	}

	// Initialize parent objective scores with both presets
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				rankedPresetId: {ObjectiveId: objective.Id},
				bonusPresetId:  {ObjectiveId: objective.Id},
			},
		}
	}

	// Apply RANK_BY_CHILD_COMPLETION_TIME scoring
	err := handleChildRankingByTime(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)

	// Apply BONUS_PER_CHILD_COMPLETION scoring
	err = handleChildBonus(objective, objective.ScoringRules[1], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)

	// Verify RANK_BY_CHILD_COMPLETION_TIME results
	// Team 1 completes all children last at -21h -> rank 1 (all children done)
	// Team 2 completes 2 children last at -19h -> rank 2
	// Team 3 completes 1 child last at -18h -> rank 3
	assert.Equal(t, 50, scoreMap[1][objective.Id].PresetCompletions[rankedPresetId].Points, "Team 1 should rank 1st with 50 points")
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[rankedPresetId].Rank, "Team 1 should have rank 1")

	assert.Equal(t, 30, scoreMap[2][objective.Id].PresetCompletions[rankedPresetId].Points, "Team 2 should rank 2nd with 30 points")
	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[rankedPresetId].Rank, "Team 2 should have rank 2")

	assert.Equal(t, 10, scoreMap[3][objective.Id].PresetCompletions[rankedPresetId].Points, "Team 3 should rank 3rd with 10 points")
	assert.Equal(t, 3, scoreMap[3][objective.Id].PresetCompletions[rankedPresetId].Rank, "Team 3 should have rank 3")

	// Verify BONUS_PER_CHILD_COMPLETION results (applied to child objectives)
	// All teams complete 4 children, so bonus is distributed by completion order
	// Team 1: completes children at -24, -23, -22, -21 hours
	assert.Equal(t, 15, scoreMap[1][1].BonusPoints, "Team 1, child 1 should have 15 bonus (completed first)")
	assert.Equal(t, 10, scoreMap[1][2].BonusPoints, "Team 1, child 2 should have 10 bonus (completed second)")
	assert.Equal(t, 5, scoreMap[1][3].BonusPoints, "Team 1, child 3 should have 5 bonus (completed third)")
	assert.Equal(t, 5, scoreMap[1][4].BonusPoints, "Team 1, child 4 should have 5 bonus (completed fourth)")

	// Team 2: completes children at -20, -18, -16, -15 hours
	assert.Equal(t, 15, scoreMap[2][1].BonusPoints, "Team 2, child 1 should have 15 bonus (completed first)")
	assert.Equal(t, 10, scoreMap[2][2].BonusPoints, "Team 2, child 2 should have 10 bonus (completed second)")
	assert.Equal(t, 5, scoreMap[2][3].BonusPoints, "Team 2, child 3 should have 5 bonus (completed third)")
	assert.Equal(t, 5, scoreMap[2][4].BonusPoints, "Team 2, child 4 should have 5 bonus (completed fourth)")

	// Team 3: completes children at -14, -12, -11, -10 hours
	assert.Equal(t, 15, scoreMap[3][1].BonusPoints, "Team 3, child 1 should have 15 bonus (completed first)")
	assert.Equal(t, 10, scoreMap[3][2].BonusPoints, "Team 3, child 2 should have 10 bonus (completed second)")
	assert.Equal(t, 5, scoreMap[3][3].BonusPoints, "Team 3, child 3 should have 5 bonus (completed third)")
	assert.Equal(t, 5, scoreMap[3][4].BonusPoints, "Team 3, child 4 should have 5 bonus (completed fourth)")
	// Verify that both scoring methods are independent and don't interfere
	// The parent objective should have separate PresetCompletion entries for each preset
	assert.NotNil(t, scoreMap[1][objective.Id].PresetCompletions[rankedPresetId], "Ranked preset should exist")
	assert.NotNil(t, scoreMap[1][objective.Id].PresetCompletions[bonusPresetId], "Bonus preset should exist")

	// Verify the presets are truly separate (bonus preset doesn't set Points/Rank on parent)
	assert.Equal(t, 0, scoreMap[1][objective.Id].PresetCompletions[bonusPresetId].Points, "Bonus preset shouldn't set points on parent objective")
	assert.Equal(t, 0, scoreMap[1][objective.Id].PresetCompletions[bonusPresetId].Rank, "Bonus preset shouldn't set rank on parent objective")
}

func TestHandleChildRankingByNumber(t *testing.T) {
	// This tests RANK_BY_CHILD_VALUE_SUM scoring where teams are ranked by the sum of Number values
	// from child objectives (e.g., total atlas completion percentage, total boss kills, etc.)
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{100, 75, 50, 25},
			},
		},
		Children: utils.Map([]int{1, 2, 3}, func(id int) *repository.Objective {
			return &repository.Objective{Id: id}
		}),
	}

	// Test data:
	// Team 1: child numbers are 10, 20, 30 = 60 total (rank 1)
	// Team 2: child numbers are 15, 15, 15 = 45 total (rank 2)
	// Team 3: child numbers are 5, 10, 15 = 30 total (rank 3, tied with team 4)
	// Team 4: child numbers are 10, 10, 10 = 30 total (rank 3, tied with team 3)
	// Team 5: child numbers are 5, 5, 5 = 15 total (rank 4)
	childData := []struct {
		objId, teamId int
		number        int
	}{
		{1, 1, 10}, {2, 1, 20}, {3, 1, 30},
		{1, 2, 15}, {2, 2, 15}, {3, 2, 15},
		{1, 3, 5}, {2, 3, 10}, {3, 3, 15},
		{1, 4, 10}, {2, 4, 10}, {3, 4, 10},
		{1, 5, 5}, {2, 5, 5}, {3, 5, 5},
	}

	// Initialize scoreMap for all teams and all child objectives
	scoreMap := make(map[int]map[int]*Score)
	for teamId := 1; teamId <= 5; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 3; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: childId,
						Number:      0,
					},
				},
			}
		}
	}

	// Apply the number data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Number = data.number
	}

	// Initialize parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	// Apply the ranking
	err := handleChildRankingByNumber(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)

	// Verify results - teams ranked by sum of child Number values
	// Team 1: 60 total -> rank 1, 100 points
	assert.Equal(t, 60, scoreMap[1][objective.Id].PresetCompletions[presetId].Number, "Team 1 should have 60 total")
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1")
	assert.Equal(t, 100, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 100 points")

	// Team 2: 45 total -> rank 2, 75 points
	assert.Equal(t, 45, scoreMap[2][objective.Id].PresetCompletions[presetId].Number, "Team 2 should have 45 total")
	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 2")
	assert.Equal(t, 75, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 75 points")

	// Team 3 & 4: 30 total (tied) -> both rank 3, both get same points
	assert.Equal(t, 30, scoreMap[3][objective.Id].PresetCompletions[presetId].Number, "Team 3 should have 30 total")
	assert.Equal(t, 3, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 3 (tied)")
	assert.Equal(t, 50, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 50 points (rank 3)")

	assert.Equal(t, 30, scoreMap[4][objective.Id].PresetCompletions[presetId].Number, "Team 4 should have 30 total")
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank, "Team 4 should have rank 3 (tied)")
	assert.Equal(t, 50, scoreMap[4][objective.Id].PresetCompletions[presetId].Points, "Team 4 should have 50 points (rank 3, same as team 3)")

	// Team 5: 15 total -> rank 4 (after tied pair), points advance to position 4
	assert.Equal(t, 15, scoreMap[5][objective.Id].PresetCompletions[presetId].Number, "Team 5 should have 15 total")
	assert.Equal(t, 4, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank, "Team 5 should have rank 4")
	assert.Equal(t, 25, scoreMap[5][objective.Id].PresetCompletions[presetId].Points, "Team 5 should have 25 points (rank 4)")
}

func TestHandleChildRankingByTimeWithRequiredChildCompletions(t *testing.T) {
	// This tests RANK_BY_CHILD_COMPLETION_TIME with Extra["required_completed_children"]
	// Teams score if they complete at least the specified number of children
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{100, 75, 50},
				Extra: map[string]string{
					"required_completed_children": "2", // Need at least 2 out of 4 children
				},
			},
		},
		Children: utils.Map([]int{1, 2, 3, 4}, func(id int) *repository.Objective {
			return &repository.Objective{Id: id}
		}),
	}

	now := time.Now()
	// Team 1: Completes 2 children, finishes at -20h -> rank 1 (earliest among teams meeting threshold)
	// Team 2: Completes 2 children, finishes at -15h -> rank 2
	// Team 3: Completes 4 children, finishes at -10h -> rank 3 (more completions don't beat faster threshold completion)
	// Team 4: Completes 1 child -> doesn't score (not enough completions)
	childData := []struct {
		objId     int
		teamId    int
		timestamp time.Time
		finished  bool
	}{
		{1, 1, now.Add(-24 * time.Hour), true},
		{2, 1, now.Add(-20 * time.Hour), true}, // Team 1's latest
		{3, 1, now.Add(-1 * time.Hour), true},  // later, but already has 2 completions so doesn't affect ranking
		{1, 2, now.Add(-18 * time.Hour), true},
		{2, 2, now.Add(-15 * time.Hour), true}, // Team 2's latest
		{1, 3, now.Add(-14 * time.Hour), true},
		{2, 3, now.Add(-12 * time.Hour), true}, // Team 3's latest
		{3, 3, now.Add(-11 * time.Hour), true},
		{4, 3, now.Add(-10 * time.Hour), true},
		{1, 4, now.Add(-22 * time.Hour), true}, // Team 4 only completes 1
	}

	// Initialize scoreMap
	scoreMap := make(map[int]map[int]*Score)
	for teamId := 1; teamId <= 4; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 4; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Apply actual completion data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Initialize parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleChildRankingByTime(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleChildRankingByTime should not return an error")

	// Verify results - teams with at least 2 completions score
	// Among finished teams, ranking is based on earlier completion time (latest child timestamp), not on total completion count.
	assert.Equal(t, 4, scoreMap[3][objective.Id].PresetCompletions[presetId].Number, "Team 3 should have 4 completions")
	assert.Equal(t, 50, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 50 points (rank 3)")
	assert.Equal(t, 3, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 3")
	assert.True(t, scoreMap[3][objective.Id].PresetCompletions[presetId].Finished, "Team 3 should be finished (4 >= 2)")

	assert.Equal(t, 3, scoreMap[1][objective.Id].PresetCompletions[presetId].Number, "Team 1 should have 3 completions")
	assert.Equal(t, 100, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 100 points (rank 1)")
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1")
	assert.True(t, scoreMap[1][objective.Id].PresetCompletions[presetId].Finished, "Team 1 should be marked finished")

	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Number, "Team 2 should have 2 completions")
	assert.Equal(t, 75, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 75 points (rank 2)")
	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 2")
	assert.True(t, scoreMap[2][objective.Id].PresetCompletions[presetId].Finished, "Team 2 should be marked finished")

	assert.Equal(t, 1, scoreMap[4][objective.Id].PresetCompletions[presetId].Number, "Team 4 should have 1 completion")
	assert.Equal(t, 0, scoreMap[4][objective.Id].PresetCompletions[presetId].Points, "Team 4 should have 0 points (not enough completions)")
	assert.Equal(t, 0, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank, "Team 4 should have rank 0")
	assert.False(t, scoreMap[4][objective.Id].PresetCompletions[presetId].Finished, "Team 4 shouldn't be finished")
}

func TestHandleChildRankingByTimeWithRequiredChildCompletionsPercent(t *testing.T) {
	// This tests RANK_BY_CHILD_COMPLETION_TIME with Extra["required_completed_children_percent"]
	// Teams score if they complete at least the specified percentage of children
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{100, 75, 50},
				Extra: map[string]string{
					"required_completed_children_percent": "50", // Need at least 50% of 4 children (>=2)
				},
			},
		},
		Children: utils.Map([]int{1, 2, 3, 4}, func(id int) *repository.Objective {
			return &repository.Objective{Id: id}
		}),
	}

	now := time.Now()
	// Team 1: Completes 3 children (75%) -> should score (75% >= 50%)
	// Team 2: Completes 2 children (50% exactly) -> should score (50% >= 50%)
	// Team 3: Completes 1 child (25%) -> should not score (25% < 50%)
	childData := []struct {
		objId, teamId int
		timestamp     time.Time
		finished      bool
	}{
		{1, 1, now.Add(-24 * time.Hour), true},
		{2, 1, now.Add(-22 * time.Hour), true},
		{3, 1, now.Add(-20 * time.Hour), true}, // Team 1's latest (3 completions)
		{1, 2, now.Add(-18 * time.Hour), true},
		{2, 2, now.Add(-15 * time.Hour), true}, // Team 2's latest (2 completions)
		{1, 3, now.Add(-10 * time.Hour), true}, // Team 3 only 1 completion
	}

	// Initialize scoreMap
	scoreMap := make(map[int]map[int]*Score)
	for teamId := 1; teamId <= 3; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 4; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Apply actual completion data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Initialize parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleChildRankingByTime(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleChildRankingByTime should not return an error")

	// Verify results - teams with at least 50% completions should be finished
	// Among finished teams, ranking is based on completion time (latest child timestamp), not highest completion count.
	assert.Equal(t, 3, scoreMap[1][objective.Id].PresetCompletions[presetId].Number, "Team 1 should have 3 completions")
	assert.Equal(t, 100, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 100 points (position 1 in sorted array)")
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1")
	assert.True(t, scoreMap[1][objective.Id].PresetCompletions[presetId].Finished, "Team 1 should be finished (75% >= 50%)")

	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Number, "Team 2 should have 2 completions")
	assert.Equal(t, 75, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 75 points (position 2 in sorted array)")
	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 2")
	assert.True(t, scoreMap[2][objective.Id].PresetCompletions[presetId].Finished, "Team 2 should be finished (50% >= 50%)")

	assert.Equal(t, 1, scoreMap[3][objective.Id].PresetCompletions[presetId].Number, "Team 3 should have 1 completion")
	assert.Equal(t, 0, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 0 points (25% < 50%)")
	assert.Equal(t, 0, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 0")
	assert.False(t, scoreMap[3][objective.Id].PresetCompletions[presetId].Finished, "Team 3 shouldn't be finished")
}

func TestHandleBingoBoardWithRequiredNumberOfBingos(t *testing.T) {
	// This tests BINGO_BOARD_RANKING with Extra["required_bingo_count"]
	// Teams must complete multiple bingo lines to score
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{100, 75, 50},
				Extra: map[string]string{
					"required_bingo_count": "2", // Need 2 bingos to score
				},
			},
		},
	}
	// Create 3x3 grid
	children := []*repository.Objective{}
	for i := range 3 {
		for j := range 3 {
			children = append(children, &repository.Objective{Id: i*3 + j + 1, Extra: fmt.Sprintf("%d,%d", i, j)})
		}
	}
	objective.Children = children

	now := time.Now()
	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)
	scoreMap[2] = make(map[int]*Score)

	// Team 1: Completes first row (1,2,3) and first column (1,4,7) = 2 bingos
	// Row completion time: max(1,2,3) = objective 3 at -26h
	// Column completion time: max(1,4,7) = objective 7 at -20h
	// Bingo timestamp: min(-26h, -20h) = -26h (earliest bingo completion)
	team1Completions := []struct {
		objId     int
		timestamp time.Time
	}{
		{1, now.Add(-30 * time.Hour)},
		{2, now.Add(-28 * time.Hour)},
		{3, now.Add(-26 * time.Hour)}, // Row completes here (earliest of two bingos)
		{4, now.Add(-24 * time.Hour)},
		{7, now.Add(-20 * time.Hour)}, // Column completes here
	}

	// Team 2: Completes first row (1,2,3) only = 1 bingo (shouldn't score)
	team2Completions := []struct {
		objId     int
		timestamp time.Time
	}{
		{1, now.Add(-25 * time.Hour)},
		{2, now.Add(-23 * time.Hour)},
		{3, now.Add(-21 * time.Hour)}, // Latest for team 2
	}

	// Initialize all objectives for both teams
	for teamId := 1; teamId <= 2; teamId++ {
		for i := 1; i <= 9; i++ {
			scoreMap[teamId][i] = &Score{
				ObjectiveId: i,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: i,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Apply team 1 completions
	for _, data := range team1Completions {
		scoreMap[1][data.objId].PresetCompletions[presetId].Finished = true
		scoreMap[1][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Apply team 2 completions
	for _, data := range team2Completions {
		scoreMap[2][data.objId].PresetCompletions[presetId].Finished = true
		scoreMap[2][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Initialize parent objective scores
	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}
	scoreMap[2][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      2,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	// Verify team 1 scores (has 2 bingos)
	assert.True(t, scoreMap[1][objective.Id].PresetCompletions[presetId].Finished, "Team 1 should be finished (has 2 bingos)")
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1")
	assert.Equal(t, 100, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 100 points")
	assert.Equal(t, team1Completions[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Team 1 timestamp should be earliest bingo completion (row completes at objective 3)")

	// Verify team 2 doesn't score (only has 1 bingo)
	assert.False(t, scoreMap[2][objective.Id].PresetCompletions[presetId].Finished, "Team 2 shouldn't be finished (only 1 bingo)")
	assert.Equal(t, 0, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 0")
	assert.Equal(t, 0, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 0 points")
}

// -- Edge case tests for tie-breaking and sorting logic --
func TestHandlePointsFromValue_MissingScoreMapEntry(t *testing.T) {
	presetId := 1
	objective := &repository.Objective{Id: 10}
	preset := &repository.ScoringRule{
		Id:     presetId,
		Points: repository.ExtendingNumberSlice{2},
	}
	aggregations := ObjectiveTeamMatches{
		10: TeamMatches{
			1:  {TeamId: 1, Number: 5, Finished: true, Timestamp: time.Now()},
			99: {TeamId: 99, Number: 3, Finished: true, Timestamp: time.Now()}, // no scoreMap entry
		},
	}
	scoreMap := map[int]map[int]*Score{
		1: {
			10: {ObjectiveId: 10, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {},
			}},
		},
		// team 99 missing from scoreMap
	}
	err := handlePointsFromValue(objective, preset, aggregations, scoreMap)
	assert.NoError(t, err)
	assert.Equal(t, 10, scoreMap[1][10].PresetCompletions[presetId].Points) // 2*5
}

func TestHandlePointsFromValue_NoCap(t *testing.T) {
	presetId := 1
	objective := &repository.Objective{Id: 10}
	preset := &repository.ScoringRule{
		Id:       presetId,
		Points:   repository.ExtendingNumberSlice{10},
		PointCap: 0, // no cap
	}
	aggregations := ObjectiveTeamMatches{
		10: TeamMatches{
			1: {TeamId: 1, Number: 100, Finished: true, Timestamp: time.Now()},
		},
	}
	scoreMap := map[int]map[int]*Score{
		1: {10: {ObjectiveId: 10, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
			presetId: {},
		}}},
	}
	err := handlePointsFromValue(objective, preset, aggregations, scoreMap)
	assert.NoError(t, err)
	assert.Equal(t, 1000, scoreMap[1][10].PresetCompletions[presetId].Points) // uncapped
}

func TestHandlePointsFromValue_WithCap(t *testing.T) {
	presetId := 1
	objective := &repository.Objective{Id: 10}
	preset := &repository.ScoringRule{
		Id:       presetId,
		Points:   repository.ExtendingNumberSlice{10},
		PointCap: 50,
	}
	aggregations := ObjectiveTeamMatches{
		10: TeamMatches{
			1: {TeamId: 1, Number: 100, Finished: true, Timestamp: time.Now()},
		},
	}
	scoreMap := map[int]map[int]*Score{
		1: {10: {ObjectiveId: 10, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
			presetId: {},
		}}},
	}
	err := handlePointsFromValue(objective, preset, aggregations, scoreMap)
	assert.NoError(t, err)
	assert.Equal(t, 50, scoreMap[1][10].PresetCompletions[presetId].Points) // capped
}

func TestHandlePointsFromValue_NilObjectiveScore(t *testing.T) {
	presetId := 1
	objective := &repository.Objective{Id: 10}
	preset := &repository.ScoringRule{
		Id:     presetId,
		Points: repository.ExtendingNumberSlice{10},
	}
	aggregations := ObjectiveTeamMatches{
		10: TeamMatches{
			1: {TeamId: 1, Number: 5, Finished: true, Timestamp: time.Now()},
		},
	}
	// scoreMap has team but no objective entry
	scoreMap := map[int]map[int]*Score{
		1: {},
	}
	err := handlePointsFromValue(objective, preset, aggregations, scoreMap)
	assert.NoError(t, err) // should not panic
}

// ========== handlePresence edge cases ==========

func TestHandlePresence_MissingScoreMapEntry(t *testing.T) {
	presetId := 1
	objective := &repository.Objective{Id: 10}
	preset := &repository.ScoringRule{
		Id:     presetId,
		Points: repository.ExtendingNumberSlice{25},
	}
	aggregations := ObjectiveTeamMatches{
		10: TeamMatches{
			1:  {TeamId: 1, Finished: true, Timestamp: time.Now()},
			99: {TeamId: 99, Finished: true, Timestamp: time.Now()}, // no scoreMap
		},
	}
	scoreMap := map[int]map[int]*Score{
		1: {10: {ObjectiveId: 10, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
			presetId: {},
		}}},
	}
	err := handlePresence(objective, preset, aggregations, scoreMap)
	assert.NoError(t, err)
	assert.Equal(t, 25, scoreMap[1][10].PresetCompletions[presetId].Points)
}

func TestHandlePresence_NotFinished(t *testing.T) {
	presetId := 1
	objective := &repository.Objective{Id: 10}
	preset := &repository.ScoringRule{
		Id:     presetId,
		Points: repository.ExtendingNumberSlice{25},
	}
	aggregations := ObjectiveTeamMatches{
		10: TeamMatches{
			1: {TeamId: 1, Finished: false, Number: 3, Timestamp: time.Now()},
		},
	}
	scoreMap := map[int]map[int]*Score{
		1: {10: {ObjectiveId: 10, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
			presetId: {},
		}}},
	}
	err := handlePresence(objective, preset, aggregations, scoreMap)
	assert.NoError(t, err)
	assert.Equal(t, 0, scoreMap[1][10].PresetCompletions[presetId].Points) // no points for unfinished
	assert.Equal(t, 3, scoreMap[1][10].PresetCompletions[presetId].Number)
}

// ========== handleRanked edge cases ==========

func TestHandleRanked_MissingScoreMapEntry(t *testing.T) {
	presetId := 1
	objective := &repository.Objective{Id: 10}
	preset := &repository.ScoringRule{
		Id:     presetId,
		Points: repository.ExtendingNumberSlice{100, 50},
	}
	now := time.Now()
	aggregations := ObjectiveTeamMatches{
		10: TeamMatches{
			1:  {TeamId: 1, Finished: true, Timestamp: now.Add(-time.Hour)},
			2:  {TeamId: 2, Finished: true, Timestamp: now},
			99: {TeamId: 99, Finished: true, Timestamp: now.Add(-2 * time.Hour)}, // no scoreMap
		},
	}
	scoreMap := map[int]map[int]*Score{
		1: {10: {ObjectiveId: 10, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}}},
		2: {10: {ObjectiveId: 10, TeamId: 2, PresetCompletions: map[int]*PresetCompletion{presetId: {}}}},
	}
	rankFun := func(a, b *Match) bool { return a.Timestamp.Before(b.Timestamp) }
	err := handleRanked(objective, preset, aggregations, rankFun, scoreMap)
	assert.NoError(t, err)
	// Team 99 (earliest) is skipped; team 1 and 2 should still get ranked
	comp1 := scoreMap[1][10].PresetCompletions[presetId]
	comp2 := scoreMap[2][10].PresetCompletions[presetId]
	assert.True(t, comp1.Points > 0 || comp2.Points > 0)
}

func TestHandleRanked_UnfinishedTeam(t *testing.T) {
	presetId := 1
	objective := &repository.Objective{Id: 10}
	preset := &repository.ScoringRule{
		Id:     presetId,
		Points: repository.ExtendingNumberSlice{100, 50},
	}
	now := time.Now()
	aggregations := ObjectiveTeamMatches{
		10: TeamMatches{
			1: {TeamId: 1, Finished: true, Timestamp: now},
			2: {TeamId: 2, Finished: false, Number: 5, Timestamp: now},
		},
	}
	scoreMap := map[int]map[int]*Score{
		1: {10: {ObjectiveId: 10, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}}},
		2: {10: {ObjectiveId: 10, TeamId: 2, PresetCompletions: map[int]*PresetCompletion{presetId: {}}}},
	}
	rankFun := func(a, b *Match) bool { return a.Timestamp.Before(b.Timestamp) }
	err := handleRanked(objective, preset, aggregations, rankFun, scoreMap)
	assert.NoError(t, err)
	comp1 := scoreMap[1][10].PresetCompletions[presetId]
	comp2 := scoreMap[2][10].PresetCompletions[presetId]
	assert.Equal(t, 100, comp1.Points)
	assert.Equal(t, 0, comp2.Points) // not finished
	assert.Equal(t, 0, comp2.Rank)   // no rank
	assert.Equal(t, 5, comp2.Number) // Number still set
}

// ========== handleChildBonus edge cases ==========

func TestHandleChildBonus_NoFinishedChildren(t *testing.T) {
	presetId := 1
	child1 := &repository.Objective{Id: 2}
	child2 := &repository.Objective{Id: 3}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child1, child2},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{10, 5},
		}},
	}
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: false},
			}},
			3: {ObjectiveId: 3, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: false},
			}},
		},
	}
	err := handleChildBonus(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)
	// teamChildScores is empty for team 1, so the loop body doesn't run
	// and scoreMap[1][1].PresetCompletions[presetId] is not updated
}

func TestHandleChildBonus_MissingParentScore(t *testing.T) {
	presetId := 1
	child := &repository.Objective{Id: 2}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{10},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			// no entry for parent objective 1
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
		},
	}
	err := handleChildBonus(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err) // should not panic
}

func TestHandleChildBonus_Grandchildren(t *testing.T) {
	presetId := 1
	// parent
	//   child1 (id=2)
	//     grandchild1 (id=4)
	//     grandchild2 (id=5)
	//   child2 (id=3)
	//     grandchild3 (id=6)
	parent := &repository.Objective{
		Id: 1,
		Children: []*repository.Objective{
			{Id: 2, Children: []*repository.Objective{
				{Id: 4},
				{Id: 5},
			}},
			{Id: 3, Children: []*repository.Objective{
				{Id: 6},
			}},
		},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{10, 7, 5},
		}},
	}

	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			4: {ObjectiveId: 4, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {Finished: true, Timestamp: now.Add(-3 * time.Hour)}}},
			5: {ObjectiveId: 5, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {Finished: true, Timestamp: now.Add(-2 * time.Hour)}}},
			6: {ObjectiveId: 6, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {Finished: true, Timestamp: now.Add(-1 * time.Hour)}}},
		},
	}

	err := handleChildBonus(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)

	assert.Equal(t, 10, scoreMap[1][4].BonusPoints, "grandchild1 (1st finished) should get 10 bonus")
	assert.Equal(t, 7, scoreMap[1][5].BonusPoints, "grandchild2 (2nd finished) should get 7 bonus")
	assert.Equal(t, 5, scoreMap[1][6].BonusPoints, "grandchild3 (3rd finished) should get 5 bonus")
}

func TestHandleChildBonus_DoesNotAwardUnpopulatedCategoryLeaf(t *testing.T) {
	// Regression test: a CATEGORY child that has no scoring rules of its own and no
	// children populated yet (e.g. event setup isn't finished) has an empty
	// PresetCompletions map. getLeafIds treats it as a leaf since it has no children,
	// and it must NOT be mistaken for an already-finished child by handleChildBonus.
	presetId := 1
	finishedChild := &repository.Objective{Id: 2}
	unpopulatedCategoryChild := &repository.Objective{Id: 3}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{finishedChild, unpopulatedCategoryChild},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{10, 5},
		}},
	}

	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
			// unpopulated CATEGORY objective: no scoring rules of its own, so its
			// PresetCompletions map is empty even though nothing has happened.
			3: {ObjectiveId: 3, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{}},
		},
	}

	err := handleChildBonus(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)

	assert.Equal(t, 10, scoreMap[1][2].BonusPoints, "the actually finished child should get the bonus")
	assert.Equal(t, 0, scoreMap[1][3].BonusPoints, "unpopulated category leaf must not be awarded a bonus")
	assert.Equal(t, 1, scoreMap[1][1].PresetCompletions[presetId].Number, "only 1 of 2 children should count as finished")
	assert.False(t, scoreMap[1][1].PresetCompletions[presetId].Finished, "parent should not be finished since only 1 of 2 children is done")
}

// ========== handleBingoBoard edge cases ==========

func TestHandleBingoBoard_InvalidRequiredBingos(t *testing.T) {
	presetId := 1
	child1 := &repository.Objective{Id: 2, Extra: "0,0"}
	child2 := &repository.Objective{Id: 3, Extra: "0,1"}
	child3 := &repository.Objective{Id: 4, Extra: "1,0"}
	child4 := &repository.Objective{Id: 5, Extra: "1,1"}
	objective := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child1, child2, child3, child4},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
			Extra:  map[string]string{"required_bingo_count": "not_a_number"},
		}},
	}
	now := time.Now()
	// Team 1 completes top row (0,0) and (0,1) → 1 bingo
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
			3: {ObjectiveId: 3, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
			4: {ObjectiveId: 4, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: false},
			}},
			5: {ObjectiveId: 5, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: false},
			}},
		},
	}
	// invalid parse → falls back to 1 required bingo. Top row complete → should finish.
	err := handleBingoBoard(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)
	assert.True(t, scoreMap[1][1].PresetCompletions[presetId].Finished)
}

func TestHandleBingoBoard_ChildWithoutGridCoordinates(t *testing.T) {
	presetId := 1
	child1 := &repository.Objective{Id: 2, Extra: "0,0"}
	child2 := &repository.Objective{Id: 3, Extra: "invalid"}
	objective := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child1, child2},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
			3: {ObjectiveId: 3, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
		},
	}
	// child2 has no valid grid coords → gets gridCellMap entry {0,0} by default (zero value)
	// The function should not panic.
	err := handleBingoBoard(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)
}

func TestHandleBingoBoard_NoBingoCompletion(t *testing.T) {
	presetId := 1
	child1 := &repository.Objective{Id: 2, Extra: "0,0"}
	child2 := &repository.Objective{Id: 3, Extra: "0,1"}
	child3 := &repository.Objective{Id: 4, Extra: "1,0"}
	child4 := &repository.Objective{Id: 5, Extra: "1,1"}
	objective := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child1, child2, child3, child4},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
		}},
	}
	now := time.Now()
	// Team 1 only has diagonal (0,0) and (1,1) — not a complete row/col/diagonal in 2x2
	// Wait, (0,0) and (1,1) IS a diagonal in 2x2. Let me only give (0,0) and (1,0).
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
			3: {ObjectiveId: 3, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: false},
			}},
			4: {ObjectiveId: 4, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
			5: {ObjectiveId: 5, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: false},
			}},
		},
	}
	// (0,0) and (1,0) done → col 0 complete! Let me change to (0,0) only.
	// Actually, let's just give (0,0) completed and nothing else:
	scoreMap[1][3].PresetCompletions[presetId].Finished = false
	scoreMap[1][4].PresetCompletions[presetId].Finished = false

	err := handleBingoBoard(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)
	assert.False(t, scoreMap[1][1].PresetCompletions[presetId].Finished)
	assert.Equal(t, 0, scoreMap[1][1].PresetCompletions[presetId].Points)
}

func TestHandleBingoBoard_MissingScoreMapForParent(t *testing.T) {
	presetId := 1
	child1 := &repository.Objective{Id: 2, Extra: "0,0"}
	child2 := &repository.Objective{Id: 3, Extra: "1,0"}
	objective := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child1, child2},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			// no entry for parent objective 1
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
			3: {ObjectiveId: 3, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
		},
	}
	err := handleBingoBoard(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err) // should not panic
}

// ========== handleChildRankingByTime edge cases ==========

func TestHandleChildRankingByTime_InvalidExtraParse(t *testing.T) {
	presetId := 1
	child := &repository.Objective{Id: 2}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
			Extra:  map[string]string{"required_completed_children": "abc"},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
		},
	}
	// invalid parse → falls back to len(children) = 1
	err := handleChildRankingByTime(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)
	assert.True(t, scoreMap[1][1].PresetCompletions[presetId].Finished)
	assert.Equal(t, 100, scoreMap[1][1].PresetCompletions[presetId].Points)
}

func TestHandleChildRankingByTime_MissingScoreMapForParent(t *testing.T) {
	presetId := 1
	child := &repository.Objective{Id: 2}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			// no entry for parent objective 1
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
		},
	}
	err := handleChildRankingByTime(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)
}

func TestHandleChildRankingByTime_ZeroRequiredCompletions(t *testing.T) {
	presetId := 1
	child := &repository.Objective{Id: 2}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
			Extra:  map[string]string{"required_completed_children": "0"},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: true, Timestamp: now},
			}},
		},
	}
	// requiredChildCompletions=0 → the guard skips timestamp correction,
	// but ObjectivesCompleted (1) >= 0, so team IS finished with default timestamp
	err := handleChildRankingByTime(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)
	assert.True(t, scoreMap[1][1].PresetCompletions[presetId].Finished)
	assert.Equal(t, 100, scoreMap[1][1].PresetCompletions[presetId].Points)
}

func TestHandleChildRankingByTime_PercentOverwritesAbsolute(t *testing.T) {
	presetId := 1
	children := make([]*repository.Objective, 10)
	for i := range children {
		children[i] = &repository.Objective{Id: i + 2}
	}
	parent := &repository.Objective{
		Id:       1,
		Children: children,
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
			Extra: map[string]string{
				"required_completed_children":         "1",  // would be easy
				"required_completed_children_percent": "50", // 50% of 10 = 5
			},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
		},
	}
	// Only complete 3 of 10 children → below 50%
	for i := 2; i <= 11; i++ {
		finished := i <= 4 // 3 finished
		scoreMap[1][i] = &Score{
			ObjectiveId: i,
			TeamId:      1,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {Finished: finished, Timestamp: now},
			},
		}
	}
	err := handleChildRankingByTime(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)
	// 3 < 5 (50%), so team shouldn't finish despite 3 >= 1 (absolute)
	assert.False(t, scoreMap[1][1].PresetCompletions[presetId].Finished)
}

// ========== handleChildRankingByNumber edge cases ==========

func TestHandleChildRankingByNumber_ZeroNumbers(t *testing.T) {
	presetId := 1
	child := &repository.Objective{Id: 2}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
		}},
	}
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Number: 0},
			}},
		},
	}
	err := handleChildRankingByNumber(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)
	comp := scoreMap[1][1].PresetCompletions[presetId]
	assert.Equal(t, 0, comp.Number) // sum of zeros
	assert.Equal(t, 0, comp.Points) // skipped due to Number==0
}

func TestHandleChildRankingByNumber_MissingChildScore(t *testing.T) {
	presetId := 1
	child1 := &repository.Objective{Id: 2}
	child2 := &repository.Objective{Id: 3}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child1, child2},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Number: 5, Timestamp: now},
			}},
			// child 3 missing from scoreMap
		},
	}
	err := handleChildRankingByNumber(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)
	comp := scoreMap[1][1].PresetCompletions[presetId]
	assert.Equal(t, 5, comp.Number) // only counts existing child
	assert.Equal(t, 100, comp.Points)
}

func TestHandleChildRankingByNumber_MissingParentScore(t *testing.T) {
	presetId := 1
	child := &repository.Objective{Id: 2}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			// no parent entry
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{
				presetId: {Number: 5, Timestamp: now},
			}},
		},
	}
	err := handleChildRankingByNumber(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err) // should not panic
}

// ========== EvaluateAggregations edge cases ==========

func TestEvaluateAggregations_UnknownScoringRule(t *testing.T) {
	objective := &repository.Objective{
		Id: 1,
		ScoringRules: []*repository.ScoringRule{{
			Id:       1,
			RuleType: "NONEXISTENT_METHOD",
			Points:   repository.ExtendingNumberSlice{10},
		}},
	}
	scoreMap := map[int]map[int]*Score{}
	err := EvaluateAggregations(objective, make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err) // unknown method is silently skipped
}

func TestEvaluateAggregations_ChildError(t *testing.T) {
	// This tests that if a child evaluation fails, the error propagates.
	// All scoring functions return nil normally, so we just test the recursion works.
	child := &repository.Objective{
		Id:           2,
		ScoringRules: []*repository.ScoringRule{},
	}
	parent := &repository.Objective{
		Id:           1,
		Children:     []*repository.Objective{child},
		ScoringRules: []*repository.ScoringRule{},
	}
	scoreMap := map[int]map[int]*Score{}
	err := EvaluateAggregations(parent, make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)
}

// ========== getBingoCompletionTime edge cases ==========

func TestGetBingoCompletionTime_EmptyGrid(t *testing.T) {
	result := getBingoCompletionTime(1, map[int]map[int]time.Time{}, 3)
	assert.True(t, result.IsZero())
}

func TestGetBingoCompletionTime_InsufficientBingos(t *testing.T) {
	now := time.Now()
	grid := map[int]map[int]time.Time{
		0: {0: now, 1: now}, // row 0 complete in 2x2
	}
	// Require 2 bingos but only 1 is complete
	result := getBingoCompletionTime(2, grid, 2)
	assert.True(t, result.IsZero())
}

func TestGetBingoCompletionTime_ExactBingos(t *testing.T) {
	now := time.Now()
	grid := map[int]map[int]time.Time{
		0: {0: now, 1: now.Add(time.Hour)},     // row 0 complete
		1: {0: now, 1: now.Add(2 * time.Hour)}, // row 1 complete
	}
	// Require 2 bingos, have 2 rows complete
	result := getBingoCompletionTime(2, grid, 2)
	assert.False(t, result.IsZero())
}

// ========== handleChildRankingByTime: uncovered branches ==========

func TestHandleChildRankingByTime_InvalidPercentParse(t *testing.T) {
	presetId := 1
	child := &repository.Objective{Id: 2}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
			Extra: map[string]string{
				"required_completed_children_percent": "not-a-number",
			},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		1: {
			1: {ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {}}},
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {Finished: true, Timestamp: now}}},
		},
	}
	err := handleChildRankingByTime(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)
	// Invalid parse falls back to len(children)=1, so completing 1 child should finish
	assert.True(t, scoreMap[1][1].PresetCompletions[presetId].Finished)
}

func TestHandleChildRankingByTime_NilScoreMapEntry(t *testing.T) {
	presetId := 1
	child := &repository.Objective{Id: 2}
	parent := &repository.Objective{
		Id:       1,
		Children: []*repository.Objective{child},
		ScoringRules: []*repository.ScoringRule{{
			Id:     presetId,
			Points: repository.ExtendingNumberSlice{100},
		}},
	}
	now := time.Now()
	scoreMap := map[int]map[int]*Score{
		// Team 1 has no entry for the parent objective → nil guard hit
		1: {
			2: {ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*PresetCompletion{presetId: {Finished: true, Timestamp: now}}},
		},
	}
	err := handleChildRankingByTime(parent, parent.ScoringRules[0], nil, scoreMap)
	assert.NoError(t, err)
}

// ========== getExtremeQuery: uncovered default branch ==========

func TestGetExtremeQuery_InvalidType(t *testing.T) {
	_, err := getExtremeQuery("INVALID_TYPE")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid aggregation type")
}

func TestGetExtremeQuery_Maximum(t *testing.T) {
	query, err := getExtremeQuery(repository.CountingMethodHighestValue)
	assert.NoError(t, err)
	assert.Contains(t, query, "DESC")
}

func TestGetExtremeQuery_Minimum(t *testing.T) {
	query, err := getExtremeQuery(repository.CountingMethodLowestValue)
	assert.NoError(t, err)
	assert.Contains(t, query, "ASC")
}

// ========== handleChildRankingByTime sort: multiple unfinished teams ==========

func TestHandleChildRankingByTime_MultipleUnfinishedTeams(t *testing.T) {
	// Two teams below required_completed_children with different completion counts.
	// This exercises the sort branches for unfinished teams.
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringRules: []*repository.ScoringRule{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{100, 75, 50, 25},
				Extra: map[string]string{
					"required_completed_children": "3",
				},
			},
		},
		Children: []*repository.Objective{
			{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
		},
	}

	now := time.Now()
	scoreMap := make(map[int]map[int]*Score)
	for teamId := 1; teamId <= 4; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 4; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {ObjectiveId: childId},
				},
			}
		}
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	// Team 1: 3 completions (meets threshold) — finished
	for _, childId := range []int{1, 2, 3} {
		scoreMap[1][childId].PresetCompletions[presetId].Finished = true
		scoreMap[1][childId].PresetCompletions[presetId].Timestamp = now.Add(-time.Duration(childId) * time.Hour)
	}
	// Team 2: 2 completions (below threshold) — unfinished
	for _, childId := range []int{1, 2} {
		scoreMap[2][childId].PresetCompletions[presetId].Finished = true
		scoreMap[2][childId].PresetCompletions[presetId].Timestamp = now.Add(-time.Duration(childId) * time.Hour)
	}
	// Team 3: 1 completion (below threshold) — unfinished, fewer than team 2
	scoreMap[3][1].PresetCompletions[presetId].Finished = true
	scoreMap[3][1].PresetCompletions[presetId].Timestamp = now.Add(-time.Hour)
	// Team 4: 2 completions (below threshold) — same count as team 2, exercises equal-completions branch
	for _, childId := range []int{1, 2} {
		scoreMap[4][childId].PresetCompletions[presetId].Finished = true
		scoreMap[4][childId].PresetCompletions[presetId].Timestamp = now.Add(-time.Duration(childId+5) * time.Hour)
	}

	err := handleChildRankingByTime(objective, objective.ScoringRules[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)

	// Team 1 should be finished and ranked
	assert.True(t, scoreMap[1][objective.Id].PresetCompletions[presetId].Finished)
	assert.Equal(t, 100, scoreMap[1][objective.Id].PresetCompletions[presetId].Points)

	// Teams 2, 3, 4 unfinished — no points
	assert.False(t, scoreMap[2][objective.Id].PresetCompletions[presetId].Finished)
	assert.False(t, scoreMap[3][objective.Id].PresetCompletions[presetId].Finished)
	assert.False(t, scoreMap[4][objective.Id].PresetCompletions[presetId].Finished)
}
