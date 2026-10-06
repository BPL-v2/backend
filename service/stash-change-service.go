package service

import (
	"bpl/repository"
	"bpl/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ninjaStatsURL = "https://poe.ninja/poe1/api/data/stats"

type NinjaResponse struct {
	NextChangeId string `json:"river_next_change_id"`
}

type StashChangeService interface {
	GetLatestTimestamp(eventId int) (time.Time, error)
	SaveStashChangesConditionally(message repository.StashChangeMessage, eventId int, sendFunc func([]byte) error) error
	GetNextChangeIdForEvent(event *repository.Event) (string, error)
	GetCurrentChangeIdForEvent(event *repository.Event) (*repository.ChangeId, error)
	GetInitialChangeId(event *repository.Event) (string, error)
	GetNinjaDifference(changeId string) int
}

type StashChangeServiceImpl struct {
	stashChangeRepository repository.StashChangeRepository
}

func NewStashChangeService() StashChangeService {
	return &StashChangeServiceImpl{
		stashChangeRepository: repository.NewStashChangeRepository(),
	}
}

func (s *StashChangeServiceImpl) GetLatestTimestamp(eventId int) (time.Time, error) {
	return s.stashChangeRepository.GetLatestTimestamp(eventId)
}

func (s *StashChangeServiceImpl) SaveStashChangesConditionally(message repository.StashChangeMessage, eventId int, sendFunc func([]byte) error) error {
	return s.stashChangeRepository.SaveStashChangesConditionally(message, eventId, sendFunc)
}

func (s *StashChangeServiceImpl) GetNextChangeIdForEvent(event *repository.Event) (string, error) {
	changeId, err := s.stashChangeRepository.GetChangeIdForEvent(event)
	if err != nil {
		return "", fmt.Errorf("failed to get next change id for event %d: %s", event.Id, err)
	}
	return changeId.NextChangeId, nil
}
func (s *StashChangeServiceImpl) GetCurrentChangeIdForEvent(event *repository.Event) (*repository.ChangeId, error) {
	changeId, err := s.stashChangeRepository.GetChangeIdForEvent(event)
	if err != nil {
		return nil, fmt.Errorf("failed to get current change id for event %d: %s", event.Id, err)
	}
	return &changeId, nil
}

func GetNinjaChangeId() (string, error) {
	response, err := http.Get(ninjaStatsURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch ninja change id: %s", err)
	}
	defer utils.Closer(response.Body)()
	var ninjaResponse NinjaResponse
	err = json.NewDecoder(response.Body).Decode(&ninjaResponse)
	if err != nil {
		return "", fmt.Errorf("failed to decode ninja change id response: %s", err)
	}
	return ninjaResponse.NextChangeId, nil
}

func (s *StashChangeServiceImpl) GetInitialChangeId(event *repository.Event) (string, error) {
	stashChange, err := s.GetNextChangeIdForEvent(event)
	if err == nil {
		return stashChange, nil
	}
	log.Print("Initial change id not found, fetching from poe.ninja")
	return GetNinjaChangeId()
}

func ChangeIdToInt(changeId string) int {
	sum := 0
	for str := range strings.SplitSeq(changeId, "-") {
		i, err := strconv.Atoi(str)
		if err != nil {
			return 0
		}
		sum += i
	}
	return sum
}

func (s *StashChangeServiceImpl) GetNinjaDifference(changeId string) int {
	ninjaId, err := GetNinjaChangeId()
	if err != nil {
		return 0
	}
	return ChangeIdToInt(ninjaId) - ChangeIdToInt(changeId)
}
