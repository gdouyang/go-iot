package agent

import (
	"go-iot/pkg/es/orm"
	"go-iot/pkg/models"
)

// ESStore persists Agent entities in Elasticsearch via ORM.
type ESStore struct{}

func NewESStore() *ESStore { return &ESStore{} }

var _ Store = (*ESStore)(nil)

func (s *ESStore) SaveConversation(c *models.AgentConversation) error {
	c.UpdateTime = models.NewDateTime()
	o := orm.NewOrm()
	exist := models.AgentConversation{Id: c.Id}
	err := o.Read(&exist, "id")
	if err == orm.ErrNoRows {
		_, err = o.Insert(c)
		return err
	}
	if err != nil {
		return err
	}
	_, err = o.Update(c)
	return err
}

func (s *ESStore) GetConversation(id string) (*models.AgentConversation, error) {
	o := orm.NewOrm()
	c := models.AgentConversation{Id: id}
	err := o.Read(&c, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *ESStore) ListConversations(userId int64) ([]models.AgentConversation, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(models.AgentConversation{}).Filter("createId", userId)
	var result []models.AgentConversation
	_, err := qs.OrderBy("-UpdateTime", "-id").All(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ESStore) DeleteConversation(id string) error {
	o := orm.NewOrm()
	if _, err := o.Delete(&models.AgentConversation{Id: id}); err != nil {
		return err
	}
	msgs, err := s.ListMessages(id)
	if err != nil {
		return err
	}
	for i := range msgs {
		if _, err := o.Delete(&msgs[i]); err != nil {
			return err
		}
	}
	drafts, err := s.ListDrafts(id)
	if err != nil {
		return err
	}
	for i := range drafts {
		if _, err := o.Delete(&drafts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *ESStore) SaveMessage(m *models.AgentMessage) error {
	StampMessage(m)
	o := orm.NewOrm()
	_, err := o.Insert(m)
	return err
}

func (s *ESStore) ListMessages(convId string) ([]models.AgentMessage, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(models.AgentMessage{}).Filter("conversationId", convId)
	var result []models.AgentMessage
	_, err := qs.All(&result)
	if err != nil {
		return nil, err
	}
	SortMessages(result)
	return result, nil
}

func (s *ESStore) SaveDraft(d *models.AgentDraft) error {
	o := orm.NewOrm()
	exist := models.AgentDraft{Id: d.Id}
	err := o.Read(&exist, "id")
	if err == orm.ErrNoRows {
		_, err = o.Insert(d)
		return err
	}
	if err != nil {
		return err
	}
	_, err = o.Update(d)
	return err
}

func (s *ESStore) GetDraft(id string) (*models.AgentDraft, error) {
	o := orm.NewOrm()
	d := models.AgentDraft{Id: id}
	err := o.Read(&d, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *ESStore) ListDrafts(convId string) ([]models.AgentDraft, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(models.AgentDraft{}).Filter("conversationId", convId)
	var result []models.AgentDraft
	_, err := qs.All(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ESStore) SaveAudit(a *models.AgentAudit) error {
	o := orm.NewOrm()
	_, err := o.Insert(a)
	return err
}

func (s *ESStore) GetSettings(userId int64) (*models.AgentUserSettings, error) {
	o := orm.NewOrm()
	st := models.AgentUserSettings{Id: settingsID(userId)}
	err := o.Read(&st, "id")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *ESStore) SaveSettings(st *models.AgentUserSettings) error {
	o := orm.NewOrm()
	exist := models.AgentUserSettings{Id: st.Id}
	err := o.Read(&exist, "id")
	if err == orm.ErrNoRows {
		_, err = o.Insert(st)
		return err
	}
	if err != nil {
		return err
	}
	_, err = o.Update(st)
	return err
}
