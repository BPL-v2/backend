package repository

import (
	"bpl/config"
	"bpl/utils"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ApprovalStatus = string

const (
	APPROVED ApprovalStatus = "APPROVED"
	REJECTED ApprovalStatus = "REJECTED"
	PENDING  ApprovalStatus = "PENDING"
)

type Submission struct {
	Id             int             `gorm:"primaryKey"`
	ObjectiveId    int             `gorm:"not null;references:objectives(id)"`
	Timestamp      time.Time       `gorm:"not null"`
	Number         int             `gorm:"not null"`
	UserId         int             `gorm:"not null;references:users(id)"`
	Proof          string          `gorm:"not null"`
	Comment        string          `gorm:"not null"`
	ApprovalStatus ApprovalStatus  `gorm:"not null"`
	ReviewComment  *string         `gorm:"null"`
	ReviewerId     *int            `gorm:"null;references:users(id)"`
	TeamId         int             `gorm:"not null;references:teams(id)"`
	Extra          SubmissionExtra `gorm:"type:jsonb;not null;default:'{}'"`

	Objective *Objective `gorm:"foreignKey:ObjectiveId;constraint:OnDelete:CASCADE;"`
	User      *User      `gorm:"foreignKey:UserId;constraint:OnDelete:CASCADE;"`
	Reviewer  *User      `gorm:"foreignKey:ReviewerId;constraint:OnDelete:CASCADE;"`
}

// SubmissionExtra holds optional, structured details about how a submission
// was fulfilled (e.g. gems or ascendancy classes used). It mirrors
// ObjectiveDetails on the objective side and, like it, is stored as a
// single jsonb column so new fields can be added without a migration.
type SubmissionExtra struct {
	GemsUsed              []string `json:"gems_used,omitempty"`
	AscendancyClassesUsed []string `json:"ascendancy_classes_used,omitempty"`
}

func (e *SubmissionExtra) Scan(value any) error {
	if value == nil {
		*e = SubmissionExtra{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: not a byte slice")
	}
	return json.Unmarshal(bytes, e)
}

func (e SubmissionExtra) Value() (driver.Value, error) {
	return json.Marshal(e)
}

func (s *Submission) ToObjectiveMatch() *ObjectiveMatch {
	return &ObjectiveMatch{
		ObjectiveId: s.ObjectiveId,
		Timestamp:   s.Timestamp,
		Number:      s.Number,
		UserId:      &s.UserId,
		TeamId:      s.TeamId,
	}
}

type SubmissionRepository interface {
	GetSubmissionsForEvent(event *Event) ([]*Submission, error)
	GetSubmissionsForObjectives(objectives []*Objective) ([]*Submission, error)
	GetSubmissionById(id int) (*Submission, error)
	SaveSubmission(submission *Submission) (*Submission, error)
	AddMatchToSubmission(submission *Submission) error
	RemoveMatchFromSubmission(submission *Submission) error
	DeleteSubmission(submissionId int) error
	GetApprovedSubmissionUserIds(eventId *int) ([]int, error)
}

type SubmissionRepositoryImpl struct {
	DB *gorm.DB
}

func NewSubmissionRepository() SubmissionRepository {
	return &SubmissionRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *SubmissionRepositoryImpl) GetSubmissionsForEvent(event *Event) ([]*Submission, error) {
	var submissions []*Submission
	result := r.DB.Find(&submissions, "team_id in ?", event.TeamIds())
	if result.Error != nil {
		return nil, result.Error
	}
	return submissions, nil
}

func (r *SubmissionRepositoryImpl) GetSubmissionsForObjectives(objectives []*Objective) ([]*Submission, error) {
	var submissions []*Submission
	result := r.DB.Preload("Objective").Preload("User").Find(&submissions, "objective_id IN ?", utils.Map(objectives, func(o *Objective) int { return o.Id }))
	if result.Error != nil {
		return nil, result.Error
	}
	return submissions, nil
}

func (r *SubmissionRepositoryImpl) GetSubmissionById(id int) (*Submission, error) {
	var submission Submission
	result := r.DB.Preload("Objective").First(&submission, Submission{Id: id})
	if result.Error != nil {
		return nil, result.Error
	}
	return &submission, nil
}

func (r *SubmissionRepositoryImpl) SaveSubmission(submission *Submission) (*Submission, error) {
	result := r.DB.Save(submission)
	if result.Error != nil {
		return nil, result.Error
	}
	return submission, nil
}

func (r *SubmissionRepositoryImpl) AddMatchToSubmission(submission *Submission) error {
	return r.DB.Create(submission.ToObjectiveMatch()).Error
}
func (r *SubmissionRepositoryImpl) RemoveMatchFromSubmission(submission *Submission) error {
	return r.DB.Delete(ObjectiveMatch{},
		ObjectiveMatch{
			ObjectiveId: submission.ObjectiveId,
			UserId:      &submission.UserId,
			TeamId:      submission.TeamId,
			Number:      submission.Number,
		}).Error
}

func (r *SubmissionRepositoryImpl) DeleteSubmission(submissionId int) error {
	result := r.DB.Delete(&Submission{Id: submissionId})
	return result.Error
}

func (r *SubmissionRepositoryImpl) GetApprovedSubmissionUserIds(eventId *int) ([]int, error) {
	var userIds []int
	query := r.DB.Model(&Submission{}).Where("submissions.approval_status = ?", APPROVED)
	if eventId != nil {
		query = query.Joins("JOIN teams ON teams.id = submissions.team_id").
			Where("teams.event_id = ?", *eventId)
	}
	result := query.Distinct("submissions.user_id").Pluck("submissions.user_id", &userIds)
	if result.Error != nil {
		return nil, result.Error
	}
	return userIds, nil
}
