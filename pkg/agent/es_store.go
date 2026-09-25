package agent

import (
	"context"
	"time"

	"go-iot/pkg/es/orm"
	"go-iot/pkg/models"
	"go-iot/pkg/redis"
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
	_, err = o.Update(c, convSnapshotCols()...)
	return err
}

// convSnapshotCols 返回整份回写要写的列：除 title 外的全部字段。
// title 由 UpdateTitle 单独维护，任何整份回写都不应该碰它。
func convSnapshotCols() []string {
	all := orm.FieldNames(&models.AgentConversation{})
	out := make([]string, 0, len(all))
	for _, col := range all {
		if col == "Title" {
			continue
		}
		out = append(out, col)
	}
	return out
}

// UpdateTitle 只更新 title（及 updateTime）字段。标题由异步生成/用户改名单独维护，
// 不走整份回写，避免运行中的旧快照把标题盖回去。
func (s *ESStore) UpdateTitle(id, title string) error {
	o := orm.NewOrm()
	_, err := o.Update(&models.AgentConversation{Id: id, Title: title, UpdateTime: models.NewDateTime()}, "Title", "UpdateTime")
	return err
}

// TouchConversation 只更新 updateTime 字段，避免运行中心跳整份回写覆盖 run 刚写入的状态/计数。
func (s *ESStore) TouchConversation(id string) error {
	o := orm.NewOrm()
	_, err := o.Update(&models.AgentConversation{Id: id, UpdateTime: models.NewDateTime()}, "UpdateTime")
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
	if _, err := o.Delete(&models.AgentMessage{ConversationId: id}, "ConversationId"); err != nil {
		return err
	}
	if _, err := o.Delete(&models.AgentDraft{ConversationId: id}, "ConversationId"); err != nil {
		return err
	}
	if _, err := o.Delete(&models.AgentAudit{ConversationId: id}, "ConversationId"); err != nil {
		return err
	}
	if client := redis.GetRedisClient(); client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = client.Del(ctx, ConvRedisKey(id, "seq"), ConvRedisKey(id, "lock")).Err()
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

func (s *ESStore) GetMaxSeq(convId string) (int64, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(models.AgentMessage{}).Filter("conversationId", convId)
	var result []models.AgentMessage
	_, err := qs.Limit(1, 0).OrderBy("-seqNo").All(&result, "seqNo")
	if err != nil {
		return 0, err
	}
	if len(result) == 0 {
		return 0, nil
	}
	return result[0].SeqNo, nil
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

func (s *ESStore) DeleteDraft(id string) error {
	o := orm.NewOrm()
	_, err := o.Delete(&models.AgentDraft{Id: id})
	return err
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
