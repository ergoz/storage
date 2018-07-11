package sqlstore

import (
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
	"net/http"
)

type SqlSessionStore struct {
	SqlStore
}

func NewSqlSessionStore(sqlStore SqlStore) store.SessionStore {
	us := &SqlSessionStore{sqlStore}
	for _, db := range sqlStore.GetAllConns() {
		table := db.AddTableWithName(model.Session{}, "session").SetKeys(true, "Id")
		table.ColMap("Id").SetMaxSize(26)
		table.ColMap("Token").SetMaxSize(500)
		table.ColMap("UserId").SetMaxSize(26)
	}
	return us
}

func (self *SqlSessionStore) CreateIndexesIfNotExists() {

}

func (self *SqlSessionStore) Get(sessionIdOrToken string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var sessions []*model.Session

		if _, err := self.GetReplica().Select(&sessions, "SELECT id as id, 'my-token' as token, '100@10.10.10.144' as userid  FROM tokens LIMIT 1", map[string]interface{}{}); err != nil {
			result.Err = model.NewAppError("SqlSessionStore.Get", "store.sql_session.get.app_error", nil, "sessionIdOrToken="+sessionIdOrToken+", "+err.Error(), http.StatusInternalServerError)
		} else if len(sessions) == 0 {
			result.Err = model.NewAppError("SqlSessionStore.Get", "store.sql_session.get.app_error", nil, "sessionIdOrToken="+sessionIdOrToken, http.StatusNotFound)
		} else {
			result.Data = sessions[0]
			return
		}
	})
}
