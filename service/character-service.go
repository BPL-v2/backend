package service

import (
	"bpl/client"
	"bpl/metrics"
	"bpl/parser"
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

type CharacterService interface {
	TrackActivity(eventId int, update *parser.PlayerUpdate) error
	GetCharactersForUser(user *repository.User) ([]*repository.Character, error)
	GetCharactersForEvent(eventId int) ([]*repository.Character, error)
	GetLatestPoBsForEvent(eventId int) ([]*repository.CharacterPob, error)
	GetCharacterById(characterId string) (*repository.Character, error)
	GetCharacterHistory(characterId string) ([]*repository.CharacterPob, error)
	GetCharacterStatsForEvent(eventId int, cutoff time.Time) (map[string]*repository.CharacterPob, error)
	GetTeamAtlasesForEvent(eventId int, teamId int) ([]*repository.AtlasTree, error)
	GetPobForIdBeforeTimestamp(characterId string, timestamp time.Time) (*repository.CharacterPob, error)
	GetPobs(characterId string) ([]*repository.CharacterPob, error)
	UpdateCharacter(characterId string) (*client.GGGCharacter, error)
	GetInfoForCharacter(characterId string) (*CharacterInfo, error)
	UpdatePoB(pob *repository.CharacterPob) error
	UpdateLatestPoBs() error
	UpdatePoBStats() error
	GetPoBById(pobId int) (*repository.CharacterPob, error)
	DeletePoB(pobId int) error
}

type CharacterServiceImpl struct {
	characterRepository repository.CharacterRepository
	eventRepository     repository.EventRepository
	teamRepository      repository.TeamRepository
	userRepository      repository.UserRepository
	activityRepository  repository.ActivityRepository
	atlasService        AtlasService
	itemService         ItemService
	poeClient           *client.PoEClient
}

func NewCharacterService(poeClient *client.PoEClient) CharacterService {
	return &CharacterServiceImpl{
		characterRepository: repository.NewCharacterRepository(),
		eventRepository:     repository.NewEventRepository(),
		teamRepository:      repository.NewTeamRepository(),
		userRepository:      repository.NewUserRepository(),
		activityRepository:  repository.NewActivityRepository(),
		atlasService:        NewAtlasService(),
		itemService:         NewItemService(),
		poeClient:           poeClient,
	}
}

func (c *CharacterServiceImpl) TrackActivity(eventId int, update *parser.PlayerUpdate) error {
	if update.New.Character.Experience != update.Old.Character.Experience {
		err := c.activityRepository.SaveActivity(&repository.Activity{
			Time:    time.Now(),
			UserId:  update.UserId,
			EventId: eventId,
		})
		if err != nil {
			fmt.Println("Error saving activity")
		}
	}

	return nil
}

func (c *CharacterServiceImpl) GetCharactersForUser(user *repository.User) ([]*repository.Character, error) {
	return c.characterRepository.GetCharactersForUser(user)
}

func (c *CharacterServiceImpl) GetCharactersForEvent(eventId int) ([]*repository.Character, error) {
	return c.characterRepository.GetCharactersForEvent(eventId)
}

func (c *CharacterServiceImpl) GetLatestPoBsForEvent(eventId int) ([]*repository.CharacterPob, error) {
	return c.characterRepository.GetLatestPoBsForEvent(eventId)
}

func (c *CharacterServiceImpl) GetCharacterById(characterId string) (*repository.Character, error) {
	return c.characterRepository.GetCharacterById(characterId)
}

func (c *CharacterServiceImpl) GetCharacterHistory(characterId string) ([]*repository.CharacterPob, error) {
	return c.characterRepository.GetCharacterHistory(characterId)
}
func (c *CharacterServiceImpl) GetCharacterStatsForEvent(eventId int, cutoff time.Time) (map[string]*repository.CharacterPob, error) {
	return c.characterRepository.GetCharacterStatsForEvent(eventId, cutoff)
}

func (c *CharacterServiceImpl) GetTeamAtlasesForEvent(eventId int, teamId int) ([]*repository.AtlasTree, error) {
	return c.atlasService.GetLatestAtlasesForEventAndTeam(eventId, teamId)
}

func (c *CharacterServiceImpl) GetPobForIdBeforeTimestamp(characterId string, timestamp time.Time) (*repository.CharacterPob, error) {
	pob, err := c.characterRepository.GetPobByCharacterIdBeforeTimestamp(characterId, timestamp)
	if err != nil {
		return nil, err
	}
	return pob, nil
}

func (c *CharacterServiceImpl) GetPobs(characterId string) ([]*repository.CharacterPob, error) {
	pob, err := c.characterRepository.GetPobs(characterId)
	if err != nil {
		return nil, err
	}
	return pob, nil
}

func (c *CharacterServiceImpl) UpdateCharacter(characterId string) (*client.GGGCharacter, error) {
	character, err := c.characterRepository.GetCharacterById(characterId)
	if err != nil {
		return nil, err
	}
	if character.UserId == nil {
		return nil, fmt.Errorf("character has no user")
	}
	user, err := c.userRepository.GetUserById(*character.UserId, "OauthAccounts")
	if err != nil {
		return nil, err
	}
	if user.GetPoEToken() == "" {
		return nil, fmt.Errorf("user has no poe token")
	}
	event, err := c.eventRepository.GetEventById(character.EventId)
	if err != nil {
		return nil, err
	}
	response, clientErr := c.poeClient.GetCharacter(user.GetPoEToken(), character.Name, event.GetRealm())
	if clientErr != nil {
		return nil, fmt.Errorf("%s", clientErr.Description)
	}
	return response.Character, nil
}

type CharacterInfo struct {
	User      *repository.User
	Event     *repository.Event
	Character *repository.Character
	TeamId    int
}

func (ci *CharacterInfo) ToPlayerUpdate() (*parser.PlayerUpdate, error) {
	for _, oauth := range ci.User.OauthAccounts {
		if oauth.Provider == repository.ProviderPoE && oauth.Expiry.After(time.Now()) {
			return &parser.PlayerUpdate{
				UserId:      ci.User.Id,
				TeamId:      ci.TeamId,
				AccountName: *ci.User.GetAccountName(repository.ProviderPoE),
				Token:       oauth.AccessToken,
				TokenExpiry: oauth.Expiry,
				Mu:          sync.Mutex{},
				New: parser.Player{
					Character: &client.GGGCharacter{
						Id:    ci.Character.Id,
						Name:  ci.Character.Name,
						Class: ci.Character.Ascendancy,
						Level: ci.Character.Level,
					},
					VoidStones: utils.ToSet(ci.Character.VoidStones),
				},
				Old: parser.Player{
					Character: &client.GGGCharacter{
						Id:    ci.Character.Id,
						Name:  ci.Character.Name,
						Class: ci.Character.Ascendancy,
						Level: ci.Character.Level,
					},
					VoidStones: utils.ToSet(ci.Character.VoidStones),
				},
				LastUpdateTimes: struct {
					CharacterName time.Time
					Character     time.Time
					LeagueAccount time.Time
					PoB           time.Time
				}{},
			}, nil
		}
	}
	return nil, fmt.Errorf("no valid PoE oauth token found for user")
}

func (c *CharacterServiceImpl) GetInfoForCharacter(characterId string) (*CharacterInfo, error) {
	character, err := c.characterRepository.GetCharacterById(characterId)
	if err != nil {
		return nil, err
	}
	if character.UserId == nil {
		return nil, fmt.Errorf("character has no associtated user")
	}
	user, err := c.userRepository.GetUserById(*character.UserId, "OauthAccounts")
	if err != nil {
		return nil, err
	}
	teamUser, err := c.teamRepository.GetTeamForUser(character.EventId, *character.UserId)
	if err != nil {
		return nil, err
	}
	event, err := c.eventRepository.GetEventById(character.EventId)
	if err != nil {
		return nil, err
	}
	return &CharacterInfo{
		Character: character,
		User:      user,
		Event:     event,
		TeamId:    teamUser.TeamId,
	}, nil

}

func (c *CharacterServiceImpl) UpdatePoB(pob *repository.CharacterPob) error {
	newExport, err := client.UpdatePoBExport(pob.Export.ToString())
	if err != nil {
		metrics.PobsCalculatedErrorCounter.Inc()
		return err
	}
	metrics.PobsCalculatedCounter.Inc()
	p := repository.PoBExport{}
	err = p.FromString(newExport)
	if err != nil {
		return err
	}
	pob.Export = p
	pob.UpdatedAt = time.Now()
	pobDecoded, err := pob.Export.Decode()
	if err == nil {
		pob.UpdateStats(pobDecoded)
		fmt.Printf("Updated PoB for character %s\n", pob.CharacterId)
	} else {
		fmt.Printf("Error decoding updated PoB for character %s: %v\n", pob.CharacterId, err)
	}
	return c.characterRepository.SavePoB(pob)
}

func (c *CharacterServiceImpl) UpdateLatestPoBs() error {
	semaphore := make(chan struct{}, 4)
	updateStart := time.Date(2026, 07, 21, 10, 0, 0, 0, time.Local)
	startId := 0
	mappy, err := c.itemService.GetItemMap()
	if err != nil {
		return err
	}
	doedres := mappy[repository.ItemTypeUnique]["Foulborn Doedre's Scorn"]
	if doedres == 0 {
		return fmt.Errorf("Foulborn Doedre's Scorn not found in item map")
	}

	for {
		fmt.Printf("Fetching PoBs starting from ID %d\n", startId)
		pobs, err := c.characterRepository.GetPobsFromIdWithLimit(startId+1, 100)

		if err != nil {
			fmt.Printf("Error getting PoBs from id %d: %v\n", startId, err)
			return err
		}
		if len(pobs) == 0 {
			break
		}
		for _, characterPob := range pobs {
			startId = characterPob.Id
			if !slices.Contains(characterPob.Items, int32(doedres)) {
				continue
			}
			fmt.Printf("Processing PoB ID %d\n", characterPob.Id)

			if characterPob.UpdatedAt.After(updateStart) {
				continue
			}
			semaphore <- struct{}{}
			go func(characterPob *repository.CharacterPob) {
				defer func() { <-semaphore }() // Release the slot when done
				err := c.UpdatePoB(characterPob)
				if err != nil {
					fmt.Printf("Error updating PoB for character %s: %v\n", characterPob.CharacterId, err)
				}
			}(characterPob)
		}
	}
	return nil
}

func (c *CharacterServiceImpl) UpdatePoBStats() error {
	startId := 0
	_, err := c.itemService.GetItemMap()
	if err != nil {
		return err
	}

	for {
		pobs, err := c.characterRepository.GetPobsFromIdWithLimit(startId+1, 1000)

		if err != nil {
			fmt.Printf("Error getting PoBs from id %d: %v\n", startId, err)
			return err
		}
		if len(pobs) == 0 {
			break
		}
		for _, characterPob := range pobs {
			startId = characterPob.Id
			pob, err := characterPob.Export.Decode()
			if err != nil {
				fmt.Printf("Error decoding PoB for character %s: %v\n", characterPob.CharacterId, err)
				continue
			}
			itemIndexes := make(map[int]bool)
			for _, item := range pob.Items {
				if item.Rarity == "UNIQUE" {
					itemId, err := c.itemService.GetOrCreateId(item.Name, repository.ItemTypeUnique)
					if err != nil {
						fmt.Printf("Error getting unique item id %s: %v\n", item.Name, err)
						continue
					}
					itemIndexes[itemId] = true
				}
			}
			for _, skillset := range pob.Skills.SkillSets {
				for _, skill := range skillset.Skills {
					for _, gem := range skill.Gems {
						name := gem.NameSpec
						if strings.Contains(gem.GemID, "Support") {
							name += " Support"
						}
						itemId, err := c.itemService.GetOrCreateId(name, repository.ItemTypeGem)
						if err != nil {
							fmt.Printf("Error getting gem item id %s: %v\n", name, err)
							continue
						}
						itemIndexes[itemId] = true
					}
				}
			}
			if len(itemIndexes) == 0 {
				continue
			}
			characterPob.Items = make([]int32, 0, len(itemIndexes))
			for itemId := range itemIndexes {
				characterPob.Items = append(characterPob.Items, int32(itemId))
			}
			characterPob.MainSkill = pob.GetMainSkill()
		}
		err = c.characterRepository.SavePoBs(pobs)
		if err != nil {
			fmt.Printf("Error saving PoBs: %v\n", err)
		}
		fmt.Printf("Updated PoBs up to ID %d\n", startId)
	}
	return nil
}

func (c *CharacterServiceImpl) GetPoBById(pobId int) (*repository.CharacterPob, error) {
	return c.characterRepository.GetPoBById(pobId)
}

func (c *CharacterServiceImpl) DeletePoB(pobId int) error {
	return c.characterRepository.DeletePoB(pobId)
}
