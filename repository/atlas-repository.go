package repository

import (
	"bpl/config"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type AtlasRepository interface {
	CreateAtlasTree(userId int, eventId int, index int, nodes []int) error
	GetLatestAtlasesForEventAndTeam(eventId int, teamId int) (atlas []*AtlasTree, err error)
	GetLatestTreesForEvent(eventId int) (atlas []*AtlasTree, err error)
	GetAtlasesForEventAndUser(eventId int, userId int) (atlas []*AtlasTree, err error)
}

type AtlasRepositoryImpl struct {
	DB *gorm.DB
}

func NewAtlasRepository() AtlasRepository {
	return &AtlasRepositoryImpl{DB: config.DatabaseConnection()}
}

type AtlasTree struct {
	UserID    int          `gorm:"not null"`
	EventID   int          `gorm:"not null"`
	Index     int          `gorm:"not null"`
	Nodes     PassiveNodes `gorm:"not null"`
	Timestamp time.Time    `gorm:"not null;default:current_timestamp"`

	User  *User  `gorm:"foreignKey:UserID"`
	Event *Event `gorm:"foreignKey:EventID"`
}

type PassiveNodes []int

func (a PassiveNodes) GetHash() [32]byte {
	buf := make([]byte, len(a)*4)
	for i, hash := range a {
		binary.LittleEndian.PutUint32(buf[i*4:], uint32(hash))
	}
	return sha256.Sum256(buf)
}
func (t *PassiveNodes) UnmarshalJSON(data []byte) error {
	var raw []int
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*t = PassiveNodes(raw)
	return nil
}

func (t PassiveNodes) MarshalJSON() ([]byte, error) {
	return json.Marshal([]int(t))
}

func (t *PassiveNodes) Scan(value any) error {
	var array pq.Int32Array
	if err := array.Scan(value); err != nil {
		return err
	}
	tree := make([]int, len(array))
	for i, v := range array {
		tree[i] = int(int32(v) + (1 << 15))
	}
	*t = PassiveNodes(tree)
	return nil
}

func (t PassiveNodes) Value() (driver.Value, error) {
	int32Array := make(pq.Int32Array, len(t))
	for i, v := range t {
		int32Array[i] = int32(v - (1 << 15))
	}
	return int32Array.Value()
}

func (r *AtlasRepositoryImpl) CreateAtlasTree(userId int, eventId int, index int, nodes []int) error {
	tree := &AtlasTree{
		UserID:    userId,
		EventID:   eventId,
		Nodes:     PassiveNodes(nodes),
		Index:     index,
		Timestamp: time.Now(),
	}
	return r.DB.Create(tree).Error
}

func (r *AtlasRepositoryImpl) GetLatestAtlasesForEventAndTeam(eventId int, teamId int) (atlas []*AtlasTree, err error) {
	query := `
        SELECT DISTINCT ON (a.user_id, a.index) a.* 
        FROM atlas_trees a
        JOIN team_users tu ON a.user_id = tu.user_id
        WHERE tu.team_id = ? AND a.event_id = ?
        ORDER BY a.user_id, a.index, a.timestamp DESC
    `
	err = r.DB.Raw(query, teamId, eventId).Scan(&atlas).Error
	if err != nil {
		return nil, err
	}
	return atlas, nil
}

func (r *AtlasRepositoryImpl) GetLatestTreesForEvent(eventId int) (atlas []*AtlasTree, err error) {
	query := `
		SELECT DISTINCT ON (a.user_id, a.index) a.*
		FROM atlas_trees a
		WHERE a.event_id = ?
		ORDER BY a.user_id, a.index, a.timestamp DESC
	`
	err = r.DB.Raw(query, eventId).Scan(&atlas).Error
	if err != nil {
		return nil, err
	}
	return atlas, nil
}

func (r *AtlasRepositoryImpl) GetAtlasesForEventAndUser(eventId int, userId int) (atlas []*AtlasTree, err error) {
	err = r.DB.Where("event_id = ? AND user_id = ?", eventId, userId).Order("timestamp DESC").Find(&atlas).Error
	if err != nil {
		return nil, err
	}
	return atlas, nil
}
