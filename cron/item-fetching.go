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
	"strconv"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

type FetchingService struct {
	ctx                  context.Context
	event                *repository.Event
	poeClient            *client.PoEClient
	stashChangeService   service.StashChangeService
	stashChannel         chan repository.StashChangeMessage
	oauthService         service.OauthService
	userRepository       repository.UserRepository
	guildStashRepository repository.GuildStashRepository
	activityRepository   repository.ActivityRepository
	timingRepository     repository.TimingRepository
}

var (
	fetchingService *FetchingService
	once            sync.Once
)

func NewFetchingService(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) *FetchingService {
	once.Do(func() {
		stashChangeService := service.NewStashChangeService()
		fetchingService = &FetchingService{
			ctx:                  ctx,
			event:                event,
			poeClient:            poeClient,
			stashChangeService:   stashChangeService,
			oauthService:         service.NewOauthService(),
			stashChannel:         make(chan repository.StashChangeMessage),
			userRepository:       repository.NewUserRepository(),
			guildStashRepository: repository.NewGuildStashRepository(),
			activityRepository:   repository.NewActivityRepository(),
			timingRepository:     repository.NewTimingRepository(),
		}
	})
	return fetchingService
}

func (f *FetchingService) GetTimings() (map[repository.TimingKey]time.Duration, error) {
	return f.timingRepository.GetTimings()
}

func (f *FetchingService) FetchStashChanges() error {
	fmt.Println("Starting stash change fetch loop")
	token, err := f.oauthService.GetApplicationToken(repository.ProviderPoE)
	if err != nil {
		log.Printf("Failed to get PoE token: %v", err)
		return fmt.Errorf("failed to get PoE token: %w", err)
	}
	initialStashChange, err := f.stashChangeService.GetInitialChangeId(f.event)
	fmt.Printf("Initial stash change ID: %s\n", initialStashChange)
	if err != nil {
		log.Print(err)
		return nil
	}

	changeId := initialStashChange
	count := 0
	consecutiveErrors := 0
	for {
		select {
		case <-f.ctx.Done():
			return nil
		default:
			response, clientErr := f.poeClient.GetPublicStashes(token, "pc", changeId)
			if clientErr != nil {
				consecutiveErrors++
				if consecutiveErrors > 5 {
					log.Print("Too many consecutive errors, exiting")
					return fmt.Errorf("too many consecutive errors")
				}
				if clientErr.StatusCode == 429 {
					log.Print(clientErr.ResponseHeaders)
					retryAfter, err := strconv.Atoi(clientErr.ResponseHeaders.Get("Retry-After"))
					if err != nil {
						retryAfter = 60
					}
					<-time.After((time.Duration(retryAfter) + 1) * time.Second)
				} else {
					log.Print(clientErr)
					<-time.After(60 * time.Second)
				}
				continue
			}
			consecutiveErrors = 0
			select {
			case f.stashChannel <- repository.StashChangeMessage{ChangeId: changeId, NextChangeId: response.NextChangeId, Stashes: repository.ToTeamStashChanges(response.Stashes, 0)}:
			case <-f.ctx.Done():
				return nil
			}
			changeId = response.NextChangeId
			metrics.ChangeIdGauge.Set(float64(service.ChangeIdToInt(changeId)))
			if count%20 == 0 {
				ninjaId, err := service.GetNinjaChangeId()
				if err == nil {
					metrics.NinjaChangeIdGauge.Set(float64(service.ChangeIdToInt(ninjaId)))
				}
			}
			count++
		}
	}
}

type GuildStashFetcher struct {
	UserId      int
	Token       string
	TokenExpiry time.Time
	LastUse     time.Time
}

type GuildStashFetchers struct {
	Fetchers map[int]*GuildStashFetcher // key is team ID
	mu       sync.Mutex
}

func (f *GuildStashFetchers) GetToken(stash *repository.GuildStashTab) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	expiryThreshold := time.Now().Add(1 * time.Minute)
	chosenFetcher := (*GuildStashFetcher)(nil)
	for _, userId := range stash.UserIds {
		fetcher, exists := f.Fetchers[int(userId)]
		if exists && fetcher.TokenExpiry.After(expiryThreshold) && (chosenFetcher == nil || fetcher.LastUse.Before(chosenFetcher.LastUse)) {
			chosenFetcher = fetcher
		}
	}
	if chosenFetcher == nil {
		return "", fmt.Errorf("no valid fetcher found for stash %s", stash.Id)
	}
	chosenFetcher.LastUse = time.Now()
	return chosenFetcher.Token, nil
}

func InitFetchers(users []*repository.TeamUserWithPoEToken, stashes []*repository.GuildStashTab) *GuildStashFetchers {
	fetcherMap := make(map[int]*GuildStashFetcher)
	for _, user := range users {
		fetcher := &GuildStashFetcher{
			UserId:      user.UserId,
			Token:       user.Token,
			TokenExpiry: user.TokenExpiry,
			LastUse:     time.Now(),
		}
		fetcherMap[user.UserId] = fetcher
	}
	return &GuildStashFetchers{
		Fetchers: fetcherMap,
		mu:       sync.Mutex{},
	}
}

func (f *FetchingService) FilterStashChanges() error {
	err := config.CreateTopic(f.event.Id)
	if err != nil {
		return fmt.Errorf("failed to create kafka topic: %w", err)
	}
	users, err := f.userRepository.GetUsersForEvent(f.event.Id)
	if err != nil {
		return fmt.Errorf("failed to get users for event %d: %w", f.event.Id, err)
	}
	userMap := make(map[string]int)
	for _, user := range users {
		userMap[user.AccountName] = user.UserId
	}

	writer, err := config.GetWriter(f.event.Id)
	if err != nil {
		return fmt.Errorf("failed to get kafka writer: %w", err)
	}
	defer utils.Closer(writer)()

	for stashChange := range f.stashChannel {
		select {
		case <-f.ctx.Done():
			return fmt.Errorf("context canceled")
		default:
			stashes := make([]repository.TeamStashChange, 0)
			now := time.Now()
			for _, stash := range stashChange.Stashes {
				metrics.StashCounterTotal.Inc()
				if stash.StashChange.League != nil && *stash.StashChange.League == f.event.Name {
					stashes = append(stashes, stash)
					metrics.StashCounterFiltered.Inc()
					if stash.StashChange.AccountName != nil && userMap[*stash.StashChange.AccountName] != 0 {
						err = f.activityRepository.SaveActivity(&repository.Activity{
							Time:    now.Add(-5 * time.Minute),
							UserId:  userMap[*stash.StashChange.AccountName],
							EventId: f.event.Id,
						})
						if err != nil {
							log.Printf("Failed to save activity for user %s: %v", *stash.StashChange.AccountName, err)
						}
					}
				}
			}
			if len(stashes) > 0 {
				log.Printf("Found %d stashes for change ID: %s\n", len(stashes), stashChange.ChangeId)
			}
			message := repository.StashChangeMessage{
				ChangeId:     stashChange.ChangeId,
				NextChangeId: stashChange.NextChangeId,
				Stashes:      stashes,
				Timestamp:    now,
			}
			// make sure that stash changes are only saved if the messages are successfully written to kafka
			err = f.stashChangeService.SaveStashChangesConditionally(message, f.event.Id,
				func(data []byte) error {
					return writer.WriteMessages(context.Background(),
						kafka.Message{
							Value: data,
						},
					)
				})
			if err != nil {
				log.Printf("Failed to save stash changes conditionally: %v", err)
			}
		}
	}
	return nil
}

func (f *FetchingService) InitGuildStashFetching() (kafkaWriter *kafka.Writer, fetchers *GuildStashFetchers, err error) {
	users, err := f.userRepository.GetUsersForEvent(f.event.Id)
	if err != nil {
		log.Printf("Failed to get users for event %d: %v", f.event.Id, err)
		return
	}
	// todo: move this redundant db call
	stashes, err := f.guildStashRepository.GetByEvent(f.event.Id)
	if err != nil {
		log.Printf("Failed to get stashes for event %d: %v", f.event.Id, err)
		return
	}
	fetchers = InitFetchers(users, stashes)
	err = config.CreateTopic(f.event.Id)
	if err != nil {
		log.Print(err)
		return
	}
	kafkaWriter, err = config.GetWriter(f.event.Id)
	if err != nil {
		log.Print(err)
		return nil, nil, fmt.Errorf("failed to get kafka writer: %w", err)
	}
	return kafkaWriter, fetchers, nil
}

func (f *FetchingService) FetchGuildStashTab(tab *repository.GuildStashTab) error {
	kafkaWriter, fetchers, err := f.InitGuildStashFetching()
	if err != nil {
		fmt.Printf("Failed to initialize guild stash fetching: %v\n", err)
		return err
	}
	wg := sync.WaitGroup{}
	wg.Go(func() {
		err := f.updateGuildStash(tab, fetchers, kafkaWriter)
		if err != nil {
			fmt.Printf("Failed to fetch stash %s for team %d: %v\n", tab.Id, tab.TeamId, err)
		}
	})
	for _, child := range tab.Children {
		wg.Go(func() {
			err := f.updateGuildStash(child, fetchers, kafkaWriter)
			if err != nil {
				fmt.Printf("Failed to fetch stash %s for team %d: %v\n", child.Id, child.TeamId, err)
			}
		})
	}
	wg.Wait()
	return nil
}

func (f *FetchingService) AccessDeterminationLoop() {
	for {
		if time.Now().Before(f.event.EventStartTime.Add(5 * time.Minute)) {
			time.Sleep(10 * time.Second)
			continue
		}
		err := f.DetermineStashAccess()
		if err != nil {
			fmt.Printf("Failed to determine stash access: %v\n", err)
		}
		select {
		case <-f.ctx.Done():
			return
		case <-time.After(30 * time.Minute):
		}
	}
}

func (f *FetchingService) DetermineStashAccess() error {
	users, err := f.userRepository.GetUsersForEvent(f.event.Id)
	if err != nil {
		return err
	}
	stashToUsers := make(map[string]pq.Int32Array)
	stashMap := make(map[string]client.StashTab)
	userMap := make(map[int]*repository.TeamUserWithPoEToken)
	for _, user := range users {
		userMap[user.UserId] = user
	}

	type stashResult struct {
		userId  int
		stashes []client.StashTab
	}
	ch := make(chan stashResult, len(users))
	wg := sync.WaitGroup{}
	for _, user := range users {
		wg.Go(func() {
			stashes, err := f.GetAvailableStashes(user)
			if err != nil {
				return
			}
			ch <- stashResult{userId: user.UserId, stashes: *stashes}
		})
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	for result := range ch {
		for _, stash := range result.stashes {
			stashToUsers[stash.Id] = append(stashToUsers[stash.Id], int32(result.userId))
			stashMap[stash.Id] = stash
			if stash.Children != nil {
				for _, stashChild := range *stash.Children {
					stashToUsers[stashChild.Id] = append(stashToUsers[stashChild.Id], int32(result.userId))
					stashMap[stashChild.Id] = stashChild
				}
			}
		}
	}

	existingStashes, err := f.guildStashRepository.GetByEvent(f.event.Id)
	if err != nil {
		return err
	}
	existingStashMap := make(map[string]*repository.GuildStashTab)
	for _, stash := range existingStashes {
		existingStashMap[stash.Id] = stash
	}

	persistedStashes := make([]*repository.GuildStashTab, 0)
	for _, stash := range stashMap {
		users := stashToUsers[stash.Id]
		existingStash, exists := existingStashMap[stash.Id]
		if !exists {
			user := userMap[int(users[0])]
			persistedStashes = append(persistedStashes, &repository.GuildStashTab{
				Id:           stash.Id,
				EventId:      f.event.Id,
				TeamId:       user.TeamId,
				OwnerId:      user.UserId,
				Name:         stash.Name,
				Type:         stash.Type,
				Index:        stash.Index,
				Color:        stash.Metadata.Colour,
				ParentId:     stash.Parent,
				FetchEnabled: false,
				LastFetch:    time.Now(),
				UserIds:      stashToUsers[stash.Id],
				Raw:          "{}",
			})
		} else {
			existingStash.UserIds = stashToUsers[stash.Id]
			existingStash.Color = stash.Metadata.Colour
			existingStash.Name = stash.Name
			existingStash.Type = stash.Type
			existingStash.Index = stash.Index
			persistedStashes = append(persistedStashes, existingStash)
		}
	}
	err = f.guildStashRepository.SaveAll(persistedStashes)
	if err != nil {
		return err
	}
	return nil
}

func (f *FetchingService) GetAvailableStashes(user *repository.TeamUserWithPoEToken) (*[]client.StashTab, error) {
	if user.TokenExpiry.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}
	response, err := f.poeClient.ListGuildStashes(user.Token, f.event.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to list guild stashes for user %d: %v", user.UserId, err)
	}
	return &response.Stashes, nil
}

func (f *FetchingService) FetchGuildStashes() error {
	kafkaWriter, fetchers, err := f.InitGuildStashFetching()
	if err != nil {
		return fmt.Errorf("failed to initialize guild stash fetching: %w", err)
	}
	defer utils.Closer(kafkaWriter)()

	for {
		if time.Now().Before(f.event.EventStartTime.Add(5 * time.Minute)) {
			time.Sleep(10 * time.Second)
			continue
		}
		timings, err := f.GetTimings()
		if err != nil {
			return err
		}
		guildStashes, err := f.guildStashRepository.GetActiveByEvent(f.event.Id)
		if err != nil {
			return fmt.Errorf("failed to get guild stashes for event %d: %w", f.event.Id, err)
		}
		wg := sync.WaitGroup{}
		for _, stash := range guildStashes {
			if !stash.ShouldUpdate(timings) {
				continue
			}
			fmt.Printf("Fetching guild stash %s for team %d\n", stash.Id, stash.TeamId)
			wg.Go(func() {
				err := f.updateGuildStash(stash, fetchers, kafkaWriter)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
			})
		}
		wg.Wait()
		select {
		case <-f.ctx.Done():
			return fmt.Errorf("context canceled")
		case <-time.After(1 * time.Second):
		}
	}
}

func (f *FetchingService) updateGuildStash(stash *repository.GuildStashTab, fetchers *GuildStashFetchers, kafkaWriter *kafka.Writer) error {
	token, err := fetchers.GetToken(stash)
	if err != nil {
		return fmt.Errorf("no token found for team %d: %w", stash.TeamId, err)
	}
	response, httpError := f.poeClient.GetGuildStash(token, f.event.Name, stash.Id, stash.ParentId)
	if httpError != nil {
		if httpError.StatusCode == 404 {
			fmt.Printf("Stash %s not found (404), deleting from database\n", stash.Id)
			if err := f.guildStashRepository.Delete(stash.Id, stash.EventId); err != nil {
				return fmt.Errorf("failed to delete stash %s after 404: %w", stash.Id, err)
			}
			return nil
		}
		return fmt.Errorf("failed to fetch guild stash %s for team %d: %d - %s", stash.Id, stash.TeamId, httpError.StatusCode, httpError.Description)
	}
	if response.Stash == nil {
		return fmt.Errorf("empty stash response for stash %s for team %d", stash.Id, stash.TeamId)
	}
	stash.LastFetch = time.Now()
	stash.Index = response.Stash.Index
	stash.Name = response.Stash.Name
	stash.Type = response.Stash.Type
	stash.Color = response.Stash.Metadata.Colour
	if response.Stash.Items != nil && response.Stash.Type == "UniqueStash" && len(*response.Stash.Items) > 0 {
		stash.Name = parser.ItemClasses[(*response.Stash.Items)[0].BaseType]
	}
	err = f.updateStashItems(stash, response, kafkaWriter)
	if err != nil {
		return err
	}
	if response.Stash.Children != nil {
		stash.AddChildren(*response.Stash.Children)
	}
	return f.guildStashRepository.Save(stash)
}

func (f *FetchingService) updateStashItems(stash *repository.GuildStashTab, response *client.GetStashResponse, kafkaWriter *kafka.Writer) error {
	if stash.Raw != "" && stash.Raw != "{}" {
		var existingStash client.StashTab
		err := json.Unmarshal([]byte(stash.Raw), &existingStash)
		if err != nil {
			return fmt.Errorf("failed to unmarshal existing stash data for stash %s: %v", stash.Id, err)
		}
		if existingStash.GetHash() == response.Stash.GetHash() {
			fmt.Printf("No changes for stash %s, skipping\n", stash.Id)
			return nil
		}
	}
	raw, err := json.Marshal(response.Stash)
	if err != nil {
		return fmt.Errorf("failed to marshal new stash data for stash %s: %v", stash.Id, err)
	}
	stash.Raw = string(raw)
	items := []client.Item{}
	if response.Stash.Items != nil {
		items = *response.Stash.Items
	}
	newStashChange := &client.PublicStashChange{
		Id:        stash.Id,
		Public:    true,
		League:    &f.event.Name,
		Items:     items,
		StashType: stash.Type,
	}
	err = addGuildStashesToQueue(kafkaWriter, newStashChange, stash.TeamId)
	if err != nil {
		return fmt.Errorf("failed to add stash change to queue for stash %s: %w", stash.Id, err)
	}
	return nil
}

func addGuildStashesToQueue(kafkaWriter *kafka.Writer, change *client.PublicStashChange, teamId int) error {
	message, err := json.Marshal(repository.StashChangeMessage{
		ChangeId:     "",
		NextChangeId: "",
		Stashes:      []repository.TeamStashChange{{StashChange: *change, TeamId: teamId}},
		Timestamp:    time.Now(),
	})
	if err != nil {
		return err
	}
	return kafkaWriter.WriteMessages(
		context.Background(),
		kafka.Message{Value: message},
	)
}

func GuildStashFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	fetchingService := NewFetchingService(ctx, event, poeClient)
	go func() {
		err := fetchingService.FetchGuildStashes()
		if err != nil {
			fmt.Printf("Failed to fetch guild stashes: %v\n", err)
		}
	}()
	go fetchingService.AccessDeterminationLoop()
}

func ItemFetchLoop(ctx context.Context, event *repository.Event, poeClient *client.PoEClient) {
	fmt.Println("Starting item fetch loop")
	fetchingService := NewFetchingService(ctx, event, poeClient)
	go func() {
		err := fetchingService.FetchStashChanges()
		if err != nil {
			fmt.Printf("Failed to fetch stash changes: %v\n", err)
		}
	}()
	go func() {
		err := fetchingService.FilterStashChanges()
		if err != nil {
			fmt.Printf("Failed to filter stash changes: %v\n", err)
		}
	}()

}
