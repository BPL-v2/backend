package cron

import (
	"bpl/client"
	"bpl/config"
	"bpl/metrics"
	"bpl/parser"
	"bpl/repository"
	"bpl/service"
	"bpl/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/segmentio/kafka-go"
)

type MatchingService struct {
	ctx                       context.Context
	objectiveMatchService     service.ObjectiveMatchService
	objectiveService          service.ObjectiveService
	userService               service.UserService
	uniqueItemTrackingService service.UniqueItemTrackingService
	lastTimestamp             *time.Time
	event                     *repository.Event
}

func NewMatchingService(ctx context.Context, poeClient *client.PoEClient, event *repository.Event) (*MatchingService, error) {
	objectiveMatchService := service.NewObjectiveMatchService()
	objectiveService := service.NewObjectiveService()
	userService := service.NewUserService()
	uniqueItemTrackingService := service.NewUniqueItemTrackingService()
	matchingService := &MatchingService{
		objectiveMatchService:     objectiveMatchService,
		objectiveService:          objectiveService,
		userService:               userService,
		uniqueItemTrackingService: uniqueItemTrackingService,
		event:                     event,
		ctx:                       ctx,
	}
	timestamp, err := service.NewStashChangeService().GetLatestTimestamp(event.Id)
	if err == nil {
		matchingService.lastTimestamp = &timestamp
	}
	return matchingService, nil
}

func (m *MatchingService) GetStashChange(reader *kafka.Reader) (stashChange repository.StashChangeMessage, err error) {
	msg, err := reader.ReadMessage(context.Background())
	if err != nil {
		return stashChange, err
	}
	if err := json.Unmarshal(msg.Value, &stashChange); err != nil {
		return stashChange, err
	}
	return stashChange, nil
}

func (m *MatchingService) getItemMatches(
	stashChange repository.StashChangeMessage,
	userMap map[string]*repository.TeamUserWithPoEToken,
	teamMap map[string]string,
	itemChecker *parser.ItemChecker,
	desyncedObjectiveIds []int,
) []*repository.ObjectiveMatch {
	matches := make([]*repository.ObjectiveMatch, 0)
	syncFinished := len(desyncedObjectiveIds) == 0

	stashes := stashChange.Stashes

	for _, stash := range stashes {
		var accountName string
		if stash.StashChange.AccountName != nil {
			accountName = *stash.StashChange.AccountName
		}

		var userId *int
		teamId := stash.TeamId

		if accountName != "" {
			if u, ok := userMap[accountName]; ok && u != nil {
				userId = &u.UserId
				teamId = u.TeamId
			}
		}

		if stash.StashChange.League == nil || *stash.StashChange.League != m.event.Name || teamId == 0 {
			continue
		}

		fmt.Printf("Processing stash %s for team %d\n", stash.StashChange.Id, teamId)
		completions := make(map[int]int)
		if stash.StashChange.Items != nil {
			if err := m.uniqueItemTrackingService.TrackUniqueItems(stash.StashChange.Items, teamId, userId, m.event.Id, stashChange.Source, stashChange.Timestamp); err != nil {
				log.Printf("Failed to track unique items for stash %s: %v", stash.StashChange.Id, err)
			}
			for _, item := range stash.StashChange.Items {
				for _, result := range itemChecker.CheckForCompletions(&item) {
					if syncFinished || slices.Contains(desyncedObjectiveIds, result.ObjectiveId) {
						completions[result.ObjectiveId] += result.Number
					}
				}
			}
		}

		teamLabel := "<unknown>"
		if accountName != "" {
			if lbl, ok := teamMap[accountName]; ok && lbl != "" {
				teamLabel = lbl
			}
		}
		metrics.TeamMatchesTotal.WithLabelValues(teamLabel).Add(float64(len(completions)))

		sc := &repository.StashChange{
			StashId:   stash.StashChange.Id,
			EventId:   m.event.Id,
			Timestamp: stashChange.Timestamp,
		}
		matches = append(matches, m.objectiveMatchService.CreateItemMatches(completions, userId, teamId, sc)...)
	}

	return matches
}

func (m *MatchingService) GetReader(desyncedObjectiveIds []int) (*kafka.Reader, error) {
	err := m.objectiveService.StartSync(desyncedObjectiveIds)
	if err != nil {
		return nil, err
	}

	consumer, err := m.objectiveMatchService.GetKafkaConsumer(m.event.Id)
	if err != nil {
		return nil, err
	}

	if len(desyncedObjectiveIds) > 0 {
		consumer.GroupId += 1
		err = m.objectiveMatchService.SaveKafkaConsumerId(consumer)
		if err != nil {
			log.Print(err)
		}
	}

	return config.GetReader(m.event.Id, consumer.GroupId)

}

func (m *MatchingService) ProcessStashChanges(itemChecker *parser.ItemChecker, objectives []*repository.Objective) {
	users, err := m.userService.GetUsersForEvent(m.event.Id)
	if err != nil {
		log.Printf("Failed to get users for event %d: %v", m.event.Id, err)
		return
	}
	userMap := make(map[string]*repository.TeamUserWithPoEToken)
	teamMap := make(map[string]string)
	teamNames := make(map[int]string)
	for _, team := range m.event.Teams {
		teamNames[team.Id] = team.Name
	}

	for _, user := range users {
		userMap[user.AccountName] = user
		teamMap[user.AccountName] = teamNames[user.TeamId]
	}
	desyncedObjectiveIds := make([]int, 0)
	for _, objective := range objectives {
		if (objective.SyncStatus == repository.SyncStatusDesynced || objective.SyncStatus == repository.SyncStatusSyncing) && objective.ObjectiveType == repository.ObjectiveTypeItem {
			desyncedObjectiveIds = append(desyncedObjectiveIds, objective.Id)
		}
	}
	reader, err := m.GetReader(desyncedObjectiveIds)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer utils.Closer(reader)()

	syncing := len(desyncedObjectiveIds) > 0
	matches := make([]*repository.ObjectiveMatch, 0)
	if syncing {
		err = m.objectiveService.StartSync(desyncedObjectiveIds)
		if err != nil {
			log.Printf("Failed to start sync for desynced objectives: %v", err)
			return
		}
	}
	if m.lastTimestamp == nil {
		log.Println("No last change id found")
		// this means we dont have any earlier changes, so we assume there are no desynced objectives
		err = m.objectiveService.SetSynced(desyncedObjectiveIds)
		if err != nil {
			log.Printf("Failed to set synced for desynced objectives: %v", err)
		}
		desyncedObjectiveIds = make([]int, 0)
	}
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			stashChange, err := m.GetStashChange(reader)
			if err != nil {
				fmt.Printf("Failed to get stash change: %v", err)
				continue
			}
			if m.lastTimestamp != nil && stashChange.Timestamp.Truncate(time.Millisecond).Equal(m.lastTimestamp.Truncate(time.Millisecond)) {
				log.Println("Sync finished")
				// once we reach the starting change id the sync is finished
				err = m.objectiveService.SetSynced(desyncedObjectiveIds)
				if err != nil {
					log.Print(err)
				}
				syncing = false
			}

			matches = append(matches, m.getItemMatches(stashChange, userMap, teamMap, itemChecker, desyncedObjectiveIds)...)
			if !syncing {
				err = m.objectiveMatchService.SaveMatches(matches, desyncedObjectiveIds)
				if err != nil {
					fmt.Printf("Failed to save matches: %v", err)
					continue
				}
				desyncedObjectiveIds = make([]int, 0)
				matches = make([]*repository.ObjectiveMatch, 0)
			}

		}
	}
}

func StashEvaluationLoop(ctx context.Context, poeClient *client.PoEClient, event *repository.Event) error {
	fmt.Println("Starting stash evaluation loop for event", event.Name)
	m, err := NewMatchingService(ctx, poeClient, event)
	if err != nil {
		log.Fatal("Failed to create matching service:", err)
		return err
	}
	objectives, err := m.objectiveService.GetObjectivesForEvent(event.Id)
	if err != nil {
		return err
	}
	itemChecker, err := parser.NewItemChecker(objectives, false)
	if err != nil {
		return err
	}
	fmt.Println("Item checker initialized with", len(objectives), "objectives")
	go m.ProcessStashChanges(itemChecker, objectives)
	return nil
}
