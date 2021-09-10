package sqlstore

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
	"net/http"
	"strings"
)

type SqlMediaFileStore struct {
	SqlStore
}

func NewSqlMediaFileStore(sqlStore SqlStore) store.MediaFileStore {
	us := &SqlMediaFileStore{sqlStore}
	return us
}

func (self SqlMediaFileStore) CreateIndexesIfNotExists() {

}

func (s *SqlMediaFileStore) Create(file *model.MediaFile) (*model.MediaFile, *model.AppError) {
	err := s.GetMaster().SelectOne(&file, `with f as (
    insert into storage.media_files (name,
                                     size,
                                     mime_type,
                                     properties,
                                     instance,
                                     created_by,
                                     created_at, updated_by, updated_at, domain_id)
    values (:Name, :Size, :Mime, :Properties, :Instance, :CreatedBy, :CreatedAt, :UpdatedBy, :UpdatedAt, :DomainId)
    returning *
)
select f.id, f.name, f.created_at, call_center.cc_get_lookup(c.id, c.name) created_by,
       f.updated_at, call_center.cc_get_lookup(u.id, u.name) updated_by, f.mime_type, f.size, properties, d.name as domain_name
from f
    left join directory.wbt_user c on f.created_by = c.id
    left join directory.wbt_user u on f.updated_by = u.id
    inner join directory.wbt_domain d on d.dc = f.domain_id`, map[string]interface{}{
		"Name":       file.Name,
		"Size":       file.Size,
		"Mime":       file.MimeType,
		"Properties": model.StringInterfaceToJson(file.Properties),
		"Instance":   file.Instance,
		"CreatedBy":  file.CreatedBy.Id,
		"CreatedAt":  file.CreatedAt,
		"UpdatedBy":  file.UpdatedBy.Id,
		"UpdatedAt":  file.UpdatedAt,
		"DomainId":   file.DomainId,
	})

	if err != nil {
		if strings.Index(err.Error(), "duplicate") > -1 {
			return nil, model.NewAppError("SqlMediaFileStore.Save", "store.sql_media_file.save.saving.duplicate", nil,
				fmt.Sprintf("name=%s, %s", file.Name, err.Error()), http.StatusInternalServerError)
		} else {
			return nil, model.NewAppError("SqlMediaFileStore.Save", "store.sql_media_file.save.saving.app_error", nil,
				fmt.Sprintf("name=%s, %s", file.Name, err.Error()), http.StatusInternalServerError)
		}
	}

	return file, nil
}

func (s *SqlMediaFileStore) GetAllPage(domainId int64, search *model.SearchMediaFile) ([]*model.MediaFile, *model.AppError) {
	var files []*model.MediaFile

	f := map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(search.Ids),
		"Q":        search.GetQ(),
	}

	err := s.ListQuery(&files, search.ListRequest,
		`domain_id = :DomainId
				and (:Ids::int[] isnull or id = any(:Ids))
				and (:Q::varchar isnull or (name ilike :Q::varchar ))`,
		model.MediaFile{}, f)

	if err != nil {
		return nil, model.NewAppError("SqlMediaFileStore.GetAllPage", "store.sql_media_file.get_all.finding.app_error",
			nil, err.Error(), extractCodeFromErr(err))
	}

	return files, nil
}

func (s *SqlMediaFileStore) Get(domainId int64, id int) (*model.MediaFile, *model.AppError) {
	var file *model.MediaFile

	err := s.GetMaster().SelectOne(&file, `select f.id, f.name, f.created_at, call_center.cc_get_lookup(c.id, c.name) created_by,
       f.updated_at, call_center.cc_get_lookup(u.id, u.name) updated_by, f.mime_type, f.size, properties, d.name as domain_name
	from  storage.media_files f
		left join directory.wbt_user c on f.created_by = c.id
		left join directory.wbt_user u on f.updated_by = u.id
		inner join directory.wbt_domain d on d.dc = f.domain_id
    where f.domain_id = :DomainId and f.id = :Id`, map[string]interface{}{
		"DomainId": domainId,
		"Id":       id,
	})
	if err != nil {
		return nil, model.NewAppError("SqlMediaFileStore.Get", "store.sql_media_file.get.finding.app_error",
			nil, err.Error(), extractCodeFromErr(err))
	}

	return file, nil
}

func (s SqlMediaFileStore) Delete(domainId, id int64) *model.AppError {
	if _, err := s.GetMaster().Exec(`delete from storage.media_files p where id = :Id and domain_id = :DomainId`,
		map[string]interface{}{"Id": id, "DomainId": domainId}); err != nil {
		return model.NewAppError("SqlMediaFileStore.Delete", "store.sql_media_file.delete.app_error", nil,
			fmt.Sprintf("Id=%v, %s", id, err.Error()), extractCodeFromErr(err))
	}
	return nil
}

func (self *SqlMediaFileStore) Save(file *model.MediaFile) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		file.PreSave()
		if err := self.GetMaster().Insert(file); err != nil {
			//TODO
			if strings.Index(err.Error(), "duplicate") > -1 {
				result.Err = model.NewAppError("SqlMediaFileStore.Save", "store.sql_media_file.save.saving.duplicate", nil,
					fmt.Sprintf("name=%s, %s", file.Name, err.Error()), http.StatusInternalServerError)
			} else {
				result.Err = model.NewAppError("SqlMediaFileStore.Save", "store.sql_media_file.save.saving.app_error", nil,
					fmt.Sprintf("name=%s, %s", file.Name, err.Error()), http.StatusInternalServerError)
			}
		} else {
			result.Data = file
		}
	})
}

func (self *SqlMediaFileStore) GetAllByDomain(domain string, offset, limit int) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var files []*model.MediaFile

		query := `SELECT * FROM storage.media_files 
			WHERE domain = :Domain  
			LIMIT :Limit OFFSET :Offset`

		if _, err := self.GetReplica().Select(&files, query, map[string]interface{}{"Domain": domain, "Offset": offset, "Limit": limit}); err != nil {
			result.Err = model.NewAppError("SqlMediaFileStore.List", "store.sql_media_file.get_all.finding.app_error", nil, err.Error(), http.StatusInternalServerError)
		} else {
			result.Data = files
		}
	})
}

func (self *SqlMediaFileStore) GetCountByDomain(domain string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		query := `SELECT count(*) FROM storage.media_files 
			WHERE domain = :Domain`

		if count, err := self.GetReplica().SelectInt(query, map[string]interface{}{"Domain": domain}); err != nil {
			result.Err = model.NewAppError("SqlMediaFileStore.LiGetCountByDomainst", "store.sql_media_file.get_all.finding.app_error", nil, err.Error(), http.StatusInternalServerError)
		} else {
			result.Data = count
		}
	})
}

func (self *SqlMediaFileStore) Get1(id int64, domain string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		query := `SELECT * FROM storage.media_files WHERE id = :Id AND domain = :Domain`

		file := &model.MediaFile{}

		if err := self.GetReplica().SelectOne(file, query, map[string]interface{}{"Id": id, "Domain": domain}); err != nil {
			result.Err = model.NewAppError("SqlMediaFileStore.Get", "store.sql_media_file.get.finding.app_error", nil,
				fmt.Sprintf("id=%d, domain=%s, %s", id, domain, err.Error()), http.StatusInternalServerError)

			if err == sql.ErrNoRows {
				result.Err.StatusCode = http.StatusNotFound
			}
		} else {
			result.Data = file
		}
	})
}

func (self *SqlMediaFileStore) GetByName(name, domain string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		query := `SELECT * FROM storage.media_files WHERE name = :Name AND domain = :Domain`

		file := &model.MediaFile{}

		if err := self.GetReplica().SelectOne(file, query, map[string]interface{}{"Name": name, "Domain": domain}); err != nil {
			result.Err = model.NewAppError("SqlMediaFileStore.GetByName", "store.sql_media_file.get_by_name.finding.app_error", nil,
				fmt.Sprintf("name=%s, domain=%s, %s", name, domain, err.Error()), http.StatusInternalServerError)

			if err == sql.ErrNoRows {
				result.Err.StatusCode = http.StatusNotFound
			}
		} else {
			result.Data = file
		}
	})
}

func (self *SqlMediaFileStore) DeleteByName(name, domain string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		res, err := self.GetMaster().Exec("DELETE FROM storage.media_files WHERE name = :Name AND domain = :Domain", map[string]interface{}{"Name": name, "Domain": domain})
		if err != nil {
			result.Err = model.NewAppError("SqlMediaFileStore.DeleteByName", "store.sql_media_file.delete.app_error", nil,
				fmt.Sprintf("name=%s, err: %s", name, err.Error()), http.StatusInternalServerError)
			return
		}
		count, _ := res.RowsAffected()
		if count == 0 {
			result.Err = model.NewAppError("SqlMediaFileStore.DeleteByName", "store.sql_media_file.delete.not_found.app_error", map[string]interface{}{"Name": name},
				"", http.StatusNotFound)
		}
	})
}

func (self *SqlMediaFileStore) DeleteById(id int64) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		res, err := self.GetMaster().Exec("DELETE FROM storage.media_files WHERE id = :Id", map[string]interface{}{"Id": id})
		if err != nil {
			result.Err = model.NewAppError("SqlMediaFileStore.DeleteById", "store.sql_media_file.delete.app_error", nil,
				fmt.Sprintf("id=%d, err: %s", id, err.Error()), http.StatusInternalServerError)
			return
		}
		count, _ := res.RowsAffected()
		if count == 0 {
			result.Err = model.NewAppError("SqlMediaFileStore.DeleteById", "store.sql_media_file.delete.not_found.app_error", map[string]interface{}{"Id": id},
				"", http.StatusNotFound)
		}
	})
}
