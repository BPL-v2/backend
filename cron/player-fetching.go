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
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

var (
	pobQueue       = make(chan *repository.CharacterPob, 2000)
	activeServices sync.Map // eventId int -> *PlayerFetchingService

	pobWriterOnce sync.Once
	pobWriter     *kafka.Writer
)

// pobRequestMessage is published to the pob-requests Kafka topic.
type pobRequestMessage struct {
	CharacterID string          `json:"character_id"`
	Game        string          `json:"game"`
	Character   json.RawMessage `json:"character"`
	QueuedAt    time.Time       `json:"queued_at"`
}

// pobResultMessage is consumed from the pob-results Kafka topic.
type pobResultMessage struct {
	CharacterID string          `json:"character_id"`
	Character   json.RawMessage `json:"character"`
	Export      string          `json:"export"`
	Error       string          `json:"error,omitempty"`
	QueuedAt    time.Time       `json:"queued_at"`
}

func getPoBRequestWriter() *kafka.Writer {
	pobWriterOnce.Do(func() {
		pobWriter = config.GetPoBRequestWriter()
	})
	return pobWriter
}

// publishPoBRequest serialises a Character and publishes it to the
// pob-requests Kafka topic for async PoB export processing.
func publishPoBRequest(character *client.Character) {
	game := "poe1"
	if character.Realm == client.Poe2 {
		game = "poe2"
	}
	charJSON, err := json.Marshal(character)
	if err != nil {
		log.Printf("failed to marshal character %s for PoB request: %v", character.Id, err)
		return
	}
	msg := pobRequestMessage{
		CharacterID: character.Id,
		Game:        game,
		Character:   charJSON,
		QueuedAt:    time.Now(),
	}
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal PoB request for character %s: %v", character.Id, err)
		return
	}
	publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := getPoBRequestWriter().WriteMessages(publishCtx, kafka.Message{
		Key:   []byte(character.Id),
		Value: msgJSON,
	}); err != nil {
		log.Printf("failed to publish PoB request for character %s: %v", character.Id, err)
	}
}

func GetActiveServiceForEvent(eventId int) (*PlayerFetchingService, bool) {
	v, ok := activeServices.Load(eventId)
	if !ok {
		return nil, false
	}
	return v.(*PlayerFetchingService), true
}

type PlayerFetchingService struct {
	userRepository            repository.UserRepository
	objectiveMatchService     service.ObjectiveMatchService
	objectiveService          service.ObjectiveService
	characterService          service.CharacterService
	ladderService             service.LadderService
	atlasService              service.AtlasService
	oauthService              service.OauthService
	timingRepository          repository.TimingRepository
	characterRepository       repository.CharacterRepository
	activityRepository        repository.ActivityRepository
	itemWishService           service.ItemWishService
	timings                   map[repository.TimingKey]time.Duration

	lastLadderUpdate time.Time
	poeClient        *client.PoEClient
	playersByUserId  map[int]*parser.PlayerUpdate
}

func (s *PlayerFetchingService) GetPlayerByUserId(userId int) (*parser.PlayerUpdate, bool) {
	p, ok := s.playersByUserId[userId]
	return p, ok
}

func (s *PlayerFetchingService) ReloadTimings() error {
	timings, err := s.timingRepository.GetTimings()
	if err != nil {
		return err
	}
	s.timings = timings
	return nil
}

func NewPlayerFetchingService(poeClient *client.PoEClient) *PlayerFetchingService {
	return &PlayerFetchingService{
		userRepository:            repository.NewUserRepository(),
		objectiveMatchService:     service.NewObjectiveMatchService(),
		objectiveService:          service.NewObjectiveService(),
		ladderService:             service.NewLadderService(),
		characterService:          service.NewCharacterService(poeClient),
		atlasService:              service.NewAtlasService(),
		oauthService:              service.NewOauthService(),
		itemWishService:           service.NewItemWishService(),
		timingRepository:          repository.NewTimingRepository(),
		characterRepository:       repository.NewCharacterRepository(),
		activityRepository:        repository.NewActivityRepository(),
		lastLadderUpdate:          time.Now().Add(-1 * time.Hour),
		poeClient:                 poeClient,
	}
}

func (s *PlayerFetchingService) shouldUpdateLadder(timings map[repository.TimingKey]time.Duration) bool {
	return time.Since(s.lastLadderUpdate) > timings[repository.LadderUpdateInterval]
}

func (s *PlayerFetchingService) UpdateCharacterName(player *parser.PlayerUpdate, event *repository.Event) {
	charactersResponse, err := s.poeClient.ListCharacters(player.Token, event.GetRealm())
	player.Mu.Lock()
	defer player.Mu.Unlock()
	player.LastUpdateTimes.CharacterName = time.Now()
	if err != nil {
		player.SuccessiveErrors++
		if err.StatusCode == 401 || err.StatusCode == 403 {
			player.TokenExpiry = time.Now()
			return
		}
		log.Printf("Error fetching characters for player %d: %v", player.UserId, err)
		return
	}
	player.SuccessiveErrors = 0
	for _, char := range charactersResponse.Characters {
		if char.League != nil && *char.League == event.Name && char.Level > player.New.Character.Level {
			player.New.Character.Name = char.Name
			player.New.Character.Level = char.Level
			player.New.Character.Experience = char.Experience
			player.New.Character.Class = char.Class
		}
	}
}

func (s *PlayerFetchingService) UpdateCharacter(player *parser.PlayerUpdate, event *repository.Event) (*client.Character, error) {
	fmt.Println("Updating character", player.New.Character.Name)
	characterResponse, clientError := s.poeClient.GetCharacter(player.Token, player.New.Character.Name, event.GetRealm())
	player.Mu.Lock()
	defer player.Mu.Unlock()
	player.LastUpdateTimes.Character = time.Now()
	if clientError != nil {
		player.SuccessiveErrors++
		if clientError.StatusCode == 401 || clientError.StatusCode == 403 {
			player.TokenExpiry = time.Now()
			return nil, fmt.Errorf("error fetching character for player %d: %d", player.UserId, clientError.StatusCode)
		}
		if clientError.StatusCode == 404 {
			player.New.Character.Name = ""
			return nil, fmt.Errorf("character not found for player %d: %s", player.UserId, player.New.Character.Name)
		}
		return nil, fmt.Errorf("error fetching character for player %d: %v", player.UserId, clientError)
	}
	err := s.itemWishService.UpdateItemWishFulfillment(player.TeamId, player.UserId, characterResponse.Character)
	if err != nil {
		log.Printf("Error updating item wish fulfillment for player %d: %v", player.UserId, err)
	}
	player.SuccessiveErrors = 0
	player.New.Character = characterResponse.Character
	if !player.New.Character.HasSameEquipment(player.Old.Character) {
		log.Printf("Character equipment changed for player %d, queuing for PoB processing", player.UserId)
		publishPoBRequest(characterResponse.Character)
		player.LastUpdateTimes.PoB = time.Now()
	}
	player.New.VoidStones = player.Old.VoidStones.Union(player.New.Character.GetVoidStones())
	player.Old.VoidStones = player.New.VoidStones
	character := &repository.Character{
		Id:               player.New.Character.Id,
		UserId:           &player.UserId,
		EventId:          event.Id,
		Name:             player.New.Character.Name,
		Level:            player.New.Character.Level,
		MainSkill:        player.New.Character.GetMainSkill(),
		Ascendancy:       player.New.Character.Class,
		AscendancyPoints: player.New.Character.GetAscendancyPoints(),
		AtlasPoints:      player.New.MaxAtlasTreeNodes(),
		VoidStones:       pq.StringArray(player.New.VoidStones.ToSlice()),
	}
	err = s.characterRepository.Save(character)
	if err != nil {
		return nil, fmt.Errorf("error saving character %s (%s) for user %d: %v", character.Name, character.Id, player.UserId, err)
	}
	return characterResponse.Character, nil
}

func (s *PlayerFetchingService) UpdateLeagueAccount(player *parser.PlayerUpdate, event *repository.Event) {
	if event.GameVersion == repository.PoE2 {
		return
	}
	leagueAccount, err := s.poeClient.GetLeagueAccount(player.Token, event.Name)
	player.Mu.Lock()
	defer player.Mu.Unlock()
	player.LastUpdateTimes.LeagueAccount = time.Now()
	if err != nil {
		player.SuccessiveErrors++
		if err.StatusCode == 401 || err.StatusCode == 403 {
			fmt.Printf("Error fetching league account for player %d: %d", player.UserId, err.StatusCode)
			player.TokenExpiry = time.Now()
			return
		}
		log.Print(err)
		return
	}
	player.SuccessiveErrors = 0
	player.New.AtlasPassiveTrees = leagueAccount.LeagueAccount.AtlasPassiveTrees
	if len(player.New.AtlasPassiveTrees) > 0 {
		err := s.atlasService.SaveAtlasTrees(player.UserId, event.Id, player.New.AtlasPassiveTrees)
		if err != nil {
			fmt.Printf("Error saving atlas trees %d: %v\n", player.UserId, err)
		}
	}

}

func (s *PlayerFetchingService) UpdateLadder(players []*parser.PlayerUpdate, event *repository.Event) {
	if !s.shouldUpdateLadder(s.timings) {
		return
	}
	s.lastLadderUpdate = time.Now()
	var resp *client.GetLeagueLadderResponse
	var clientError *client.ClientError

	token, err := s.oauthService.GetApplicationToken(repository.ProviderPoE)
	if err != nil {
		log.Printf("Error fetching application token: %v", err)
		return
	}

	resp, clientError = s.poeClient.GetFullLadder(token, event.Name, event.GetRealm())
	if clientError != nil {
		log.Printf("Error fetching ladder: %v", clientError)
		return
	}

	charToUpdate := map[string]*parser.PlayerUpdate{}
	foundInLadder := make(map[string]bool)
	charToUserId := map[string]int{}
	for _, player := range players {
		charToUpdate[player.New.Character.Name] = player
		charToUserId[player.New.Character.Name] = player.UserId
	}

	delveHistoryEntries := make([]*repository.CharacterDelveHistory, 0)
	entriesToPersist := make([]*client.LadderEntry, 0, len(resp.Ladder.Entries))
	for _, entry := range resp.Ladder.Entries {
		entriesToPersist = append(entriesToPersist, &entry)
		if player, ok := charToUpdate[entry.Character.Name]; ok {
			player.Mu.Lock()
			foundInLadder[entry.Character.Name] = true
			player.New.Character.Level = entry.Character.Level
			if entry.Character.Depth != nil && entry.Character.Depth.Default != nil {
				player.New.DelveDepth = *entry.Character.Depth.Default
				if player.New.Character.Id != "" && player.New.DelveDepth != player.Old.DelveDepth {
					delveHistoryEntries = append(delveHistoryEntries, &repository.CharacterDelveHistory{
						CharacterId: player.New.Character.Id,
						Delve:       player.New.DelveDepth,
						RecordedAt:  time.Now(),
					})
				}
			}
			if entry.Character.Experience != nil {
				player.New.Character.Experience = *entry.Character.Experience
			}
			player.Mu.Unlock()
		}
	}
	err = s.ladderService.UpsertLadder(entriesToPersist, event.Id, charToUserId)
	if err != nil {
		log.Print(clientError)
	}
	err = s.characterService.SaveDelveHistoryEntries(delveHistoryEntries)
	if err != nil {
		log.Printf("Error saving delve history entries: %v", err)
	}
}

func (service *PlayerFetchingService) initPlayerUpdates(event *repository.Event) ([]*parser.PlayerUpdate, error) {
	users, err := service.userRepository.GetUsersForEvent(event.Id)
	if err != nil {
		return nil, err
	}
	players := utils.Map(users, func(user *repository.TeamUserWithPoEToken) *parser.PlayerUpdate {
		return &parser.PlayerUpdate{
			UserId:           user.UserId,
			TeamId:           user.TeamId,
			AccountName:      user.AccountName,
			Token:            user.Token,
			TokenExpiry:      user.TokenExpiry,
			SuccessiveErrors: 0,
			New: parser.Player{
				Character: &client.Character{},
				PoB:       &repository.CharacterPob{},
			},
			Old: parser.Player{
				Character: &client.Character{},
				PoB:       &repository.CharacterPob{},
			},
			Mu: sync.Mutex{},
			LastUpdateTimes: struct {
				CharacterName time.Time
				Character     time.Time
				LeagueAccount time.Time
				PoB           time.Time
			}{},
		}
	})
	latestCharacters, err := service.characterService.GetCharactersForEvent(event.Id)
	if err != nil {
		return nil, err
	}
	latestPobs, err := service.characterService.GetLatestPoBsForEvent(event.Id)
	if err != nil {
		return nil, err
	}
	pobMap := make(map[string]*repository.CharacterPob, len(latestPobs))
	for _, pob := range latestPobs {
		pobMap[pob.CharacterId] = pob
	}
	characterMap := make(map[int]*repository.Character, len(latestCharacters))
	voidStonesMap := make(map[int]utils.Set[string], len(latestCharacters))
	for _, character := range latestCharacters {
		existing := characterMap[*character.UserId]
		if existing == nil || character.Level > existing.Level {
			characterMap[*character.UserId] = character
		}
		voidStonesMap[*character.UserId] = voidStonesMap[*character.UserId].Union(utils.ToSet([]string(character.VoidStones)))
	}
	latestDelveDepths, err := service.characterService.GetLatestDelveDepthsForEvent(event.Id)
	if err != nil {
		return nil, err
	}

	for _, player := range players {
		if character, ok := characterMap[player.UserId]; ok {
			player.New.Character.Name = character.Name
			player.Old.Character.Name = character.Name
			player.New.Character.Id = character.Id
			player.Old.Character.Id = character.Id
			player.New.Character.Level = character.Level
			player.Old.Character.Level = character.Level
			player.New.Character.Class = character.Ascendancy
			player.Old.Character.Class = character.Ascendancy
			savedVoidStones := voidStonesMap[player.UserId]
			player.New.VoidStones = savedVoidStones
			player.Old.VoidStones = savedVoidStones
			player.LastUpdateTimes.CharacterName = time.Now()
			if pob, ok := pobMap[character.Id]; ok {
				player.New.PoB = pob
				player.Old.PoB = pob
			}
			if depth, ok := latestDelveDepths[character.Id]; ok {
				player.New.DelveDepth = depth
				player.Old.DelveDepth = depth
			}
		}
	}
	service.playersByUserId = make(map[int]*parser.PlayerUpdate, len(players))
	for _, p := range players {
		service.playersByUserId[p.UserId] = p
	}
	return players, nil
}

func (service *PlayerFetchingService) UpdatePlayerTokens(players []*parser.PlayerUpdate, event *repository.Event) []*parser.PlayerUpdate {
	users, err := service.userRepository.GetUsersForEvent(event.Id)
	usermap := make(map[int]*repository.TeamUserWithPoEToken, len(users))
	if err != nil {
		fmt.Printf("Error fetching users for event %d: %v", event.Id, err)
		return players
	}
	for _, player := range players {
		if user, ok := usermap[player.UserId]; ok {
			player.Token = user.Token
			player.TokenExpiry = user.TokenExpiry
		}
	}
	return players
}

// processPoBResult handles a PoB export result: decodes the export, builds the
// CharacterPob entity, feeds it into pobQueue for the player loop, and persists
// it if the stats changed.
func processPoBResult(character *client.Character, export string, queuedAt time.Time, characterRepo repository.CharacterRepository, itemService service.ItemService) {
	pob, err := client.DecodePoBExport(export)
	if err != nil {
		metrics.PobsCalculatedErrorCounter.Inc()
		log.Printf("Error decoding PoB export for character %s: %v", character.Name, err)
		return
	}
	metrics.PobsCalculatedCounter.Inc()
	p := repository.PoBExport{}
	if err = p.FromString(export); err != nil {
		log.Printf("Error parsing PoB export for character %s: %v", character.Name, err)
		return
	}
	itemIds, err := itemService.GetItemIds(character)
	if err != nil {
		log.Printf("Error getting item ids for character %s: %v", character.Name, err)
	}
	pobEntity := &repository.CharacterPob{
		CreatedAt:        queuedAt,
		UpdatedAt:        queuedAt,
		CharacterId:      character.Id,
		Level:            character.Level,
		Ascendancy:       character.Class,
		Export:           p,
		XP:               int64(character.Experience),
		Items:            itemIds,
		HighIlevelFlasks: int8(character.GetNumberOfHighIlvlFlasks()),
	}
	pobEntity.UpdateStats(pob)
	pobQueue <- pobEntity
	oldPob, _ := characterRepo.GetLatestCharacterPoB(character.Id)
	if pobEntity.HasEqualStats(oldPob) {
		log.Printf("No changes in stats for character %s, skipping save", character.Name)
		return
	}
	if oldPob != nil {
		if oldPobDecoded, decodeErr := oldPob.Export.Decode(); decodeErr == nil {
			if pob.NumEquipmentPieces() < oldPobDecoded.NumEquipmentPieces() {
				log.Printf("PoB for character %s has fewer equipped items than previous, skipping save", character.Name)
				return
			}
			if pob.SwappedWeaponsAndLostDps(oldPobDecoded) {
				log.Printf("PoB for character %s swapped weapons and lost dps, skipping save", character.Name)
				return
			}
		}
	}
	metrics.PobsSavedCounter.Inc()
	if err = characterRepo.SavePoB(pobEntity); err != nil {
		log.Printf("Error saving character stats for %s: %v", character.Name, err)
	}
}

// PlayerStatsLoop consumes PoB export results from the pob-results Kafka topic
// and persists them to the database, feeding the pobQueue used by PlayerFetchLoop.
func PlayerStatsLoop(ctx context.Context) {
	characterRepo := repository.NewCharacterRepository()
	itemService := service.NewItemService()

	reader := config.GetPoBResultReader()
	defer reader.Close()

	log.Print("PoB result consumer started")
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("PoB result consumer fetch error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		metrics.PobQueueGauge.Set(float64(reader.Stats().Lag))

		var result pobResultMessage
		if err := json.Unmarshal(msg.Value, &result); err != nil {
			log.Printf("Failed to unmarshal PoB result: %v", err)
			reader.CommitMessages(ctx, msg)
			continue
		}
		if result.Error != "" {
			metrics.PobsCalculatedErrorCounter.Inc()
			log.Printf("PoB error for character %s: %s", result.CharacterID, result.Error)
			reader.CommitMessages(ctx, msg)
			continue
		}
		var character client.Character
		if err := json.Unmarshal(result.Character, &character); err != nil {
			log.Printf("Failed to unmarshal character from PoB result %s: %v", result.CharacterID, err)
			reader.CommitMessages(ctx, msg)
			continue
		}
		processPoBResult(&character, result.Export, result.QueuedAt, characterRepo, itemService)
		reader.CommitMessages(ctx, msg)
	}
}
func drainStatQueue() map[string]*repository.CharacterPob {
	pobMap := make(map[string]*repository.CharacterPob)
	for {
		select {
		case pob := <-pobQueue:
			pobMap[pob.CharacterId] = pob
		default:
			return pobMap
		}
	}
}

func PlayerFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	service := NewPlayerFetchingService(poeClient)
	players, err := service.initPlayerUpdates(event)
	if err != nil {
		log.Print(err)
		return
	}
	objectives, err := service.objectiveService.GetObjectivesForEvent(event.Id)
	if err != nil {
		log.Print(err)
		return
	}
	playerChecker, err := parser.NewPlayerChecker(objectives)
	if err != nil {
		log.Print(err)
		return
	}
	teamChecker, err := parser.NewTeamChecker(objectives)
	if err != nil {
		log.Print(err)
		return
	}
	activeServices.Store(event.Id, service)
	defer activeServices.Delete(event.Id)
	fmt.Printf("Starting PlayerFetchLoop for event: %s with %d players\n", event.Name, len(players))
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if time.Now().Before(event.EventStartTime.Add(5 * time.Minute)) {
				fmt.Printf("Event %s has not started yet, waiting...\n", event.Name)
				time.Sleep(10 * time.Second)
				continue
			}
			err := service.ReloadTimings()
			if err != nil {
				log.Print(err)
				return
			}

			wg := sync.WaitGroup{}
			for _, player := range players {
				if player.ShouldUpdateCharacterName(service.timings) {
					wg.Go(func() {
						service.UpdateCharacterName(player, event)
					})
				}
				if player.ShouldUpdateCharacter(service.timings) {
					wg.Go(func() {
						_, err := service.UpdateCharacter(player, event)
						if err != nil {
							fmt.Printf("Error updating character for player %d: %v\n", player.UserId, err)
						}
					})
				}
				if player.ShouldUpdateLeagueAccount(service.timings) {
					wg.Go(func() {
						service.UpdateLeagueAccount(player, event)
					})
				}
			}
			if service.shouldUpdateLadder(service.timings) {
				wg.Go(func() {
					service.UpdateLadder(players, event)
				})
			}
			wg.Wait()

			pobMap := drainStatQueue()
			for _, player := range players {
				player.Mu.Lock()
				if pob, ok := pobMap[player.New.Character.Id]; ok {
					player.New.PoB = pob
				}
				if player.New.Character.Experience != player.Old.Character.Experience {
					player.LastActive = time.Now()
					err = service.activityRepository.SaveActivity(&repository.Activity{
						Time:    time.Now(),
						UserId:  player.UserId,
						EventId: event.Id,
					})
					if err != nil {
						fmt.Printf("Error saving activity for player %d: %v\n", player.UserId, err)
					}
				}
				player.Mu.Unlock()
			}

			matches := utils.FlatMap(players, func(player *parser.PlayerUpdate) []*repository.ObjectiveMatch {
				return service.GetPlayerMatches(player, playerChecker)
			})
			for _, team := range event.Teams {
				teamPlayers := utils.Filter(players, func(player *parser.PlayerUpdate) bool {
					return player.TeamId == team.Id
				})
				teamMatches := service.GetTeamMatches(teamPlayers, teamChecker)
				matches = append(matches, teamMatches...)
			}
			err = service.objectiveMatchService.SaveMatches(matches, []int{})
			if err != nil {
				log.Print(err)
			}
			for _, player := range players {
				player.Mu.Lock()
				player.Old = player.New
				player.Mu.Unlock()
			}
			players = service.UpdatePlayerTokens(players, event)
			time.Sleep(1 * time.Second)
		}
	}
}

func (m *PlayerFetchingService) GetPlayerMatches(player *parser.PlayerUpdate, playerChecker *parser.PlayerChecker) []*repository.ObjectiveMatch {
	return utils.Map(playerChecker.CheckForCompletions(player), func(result *parser.CheckResult) *repository.ObjectiveMatch {
		return &repository.ObjectiveMatch{
			ObjectiveId: result.ObjectiveId,
			UserId:      &player.UserId,
			Number:      result.Number,
			Timestamp:   time.Now(),
			TeamId:      player.TeamId,
		}
	})
}

func (m *PlayerFetchingService) GetTeamMatches(players []*parser.PlayerUpdate, teamChecker *parser.TeamChecker) []*repository.ObjectiveMatch {
	matches := []*repository.ObjectiveMatch{}
	if len(players) == 0 {
		return matches
	}
	now := time.Now()
	for _, result := range teamChecker.CheckForCompletions(players) {
		if result.Number > 0 {
			matches = append(matches, &repository.ObjectiveMatch{
				ObjectiveId: result.ObjectiveId,
				Number:      result.Number,
				Timestamp:   now,
				TeamId:      players[0].TeamId,
			})
		}
	}
	return matches
}
