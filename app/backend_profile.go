package app

import (
	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
)

func (app *App) FileBackendProfileCheckAccess(domainId, id int64, groups []int, access auth_manager.PermissionAccess) (bool, *model.AppError) {
	return app.Store.FileBackendProfile().CheckAccess(domainId, id, groups, access)
}

func (app *App) CreateFileBackendProfile(profile *model.FileBackendProfile) (*model.FileBackendProfile, *model.AppError) {
	return app.Store.FileBackendProfile().Create(profile)
}

func (app *App) GetFileBackendProfilePage(domainId int64, page, perPage int) ([]*model.FileBackendProfile, *model.AppError) {
	return app.Store.FileBackendProfile().GetAllPage(domainId, page*perPage, perPage)
}

func (app *App) GetFileBackendProfilePageByGroups(domainId int64, groups []int, page, perPage int) ([]*model.FileBackendProfile, *model.AppError) {
	return app.Store.FileBackendProfile().GetAllPageByGroups(domainId, groups, page*perPage, perPage)
}

func (app *App) GetFileBackendProfile(id, domain int64) (*model.FileBackendProfile, *model.AppError) {
	return app.Store.FileBackendProfile().Get(id, domain)
}

func (app *App) UpdateFileBackendProfile(profile *model.FileBackendProfile) (*model.FileBackendProfile, *model.AppError) {
	oldProfile, err := app.GetFileBackendProfile(profile.Id, profile.DomainId)
	if err != nil {
		return nil, err
	}

	oldProfile.UpdatedBy = profile.UpdatedBy
	oldProfile.UpdatedAt = profile.UpdatedAt

	oldProfile.Name = profile.Name
	oldProfile.ExpireDay = profile.ExpireDay
	oldProfile.Priority = profile.Priority
	oldProfile.Disabled = profile.Disabled
	oldProfile.MaxSizeMb = profile.MaxSizeMb
	oldProfile.Properties = profile.Properties
	oldProfile.Description = profile.Description

	return app.Store.FileBackendProfile().Update(oldProfile)

}

func (app *App) PatchFileBackendProfile(domainId, id int64, patch *model.FileBackendProfilePath) (*model.FileBackendProfile, *model.AppError) {
	oldProfile, err := app.GetFileBackendProfile(id, domainId)
	if err != nil {
		return nil, err
	}

	oldProfile.Path(patch)

	if err = oldProfile.IsValid(); err != nil {
		return nil, err
	}

	return app.Store.FileBackendProfile().Update(oldProfile)
}

func (app *App) DeleteFileBackendProfiles(domainId, id int64) (*model.FileBackendProfile, *model.AppError) {
	profile, err := app.GetFileBackendProfile(id, domainId)
	if err != nil {
		return nil, err
	}
	err = app.Store.FileBackendProfile().Delete(domainId, id)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

//FIXME remove

func (app *App) GetFileBackendProfileById(id int) (*model.FileBackendProfile, *model.AppError) {
	if result := <-app.Store.FileBackendProfile().GetById(id); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.FileBackendProfile), nil
	}
}

func (app *App) ListFileBackendProfiles(domain string, page, perPage int) ([]*model.FileBackendProfile, *model.AppError) {
	if result := <-app.Store.FileBackendProfile().GetAllPageByDomain(domain, page*perPage, perPage); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.([]*model.FileBackendProfile), nil
	}
}

func (app *App) PathFileBackendProfile(profile *model.FileBackendProfile, path *model.FileBackendProfilePath) (*model.FileBackendProfile, *model.AppError) {
	profile.Path(path)
	profile, err := app.UpdateFileBackendProfile(profile)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (app *App) GetFileBackendStore(id int, syncTime int64) (store utils.FileBackend, appError *model.AppError) {
	var ok bool
	var cache interface{}
	cache, ok = app.fileBackendCache.Get(id)
	if ok {
		store = cache.(utils.FileBackend)
		if store.GetSyncTime() != syncTime {
			store = nil
		} else {
			return
		}
	}

	if store == nil {
		var profile *model.FileBackendProfile
		profile, appError = app.GetFileBackendProfileById(id)
		if appError != nil {
			return
		}
		store, appError = utils.NewBackendStore(profile)
	}

	if appError != nil {
		return
	}

	app.fileBackendCache.Add(id, store)
	wlog.Info("Added to cache", wlog.String("name", store.Name()))
	return store, nil
}
