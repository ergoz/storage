package utils

import (
	"github.com/webitel/storage/model"
	"os"
	"path/filepath"
)

var (
	commonBaseSearchPaths = []string{
		".",
		"..",
		"../..",
		"../../..",
	}
)

func LoadConfig(fileName string) (*model.Config, string, map[string]interface{}, *model.AppError) {
	var envConfig = make(map[string]interface{})
	dbDatasource := "postgres://webitel:webitel@10.10.10.200:5432/webitel?sslmode=disable&connect_timeout=10"
	dbDriverName := "postgres"
	maxIdleConns := 100
	maxOpenConns := 100
	connMaxLifetimeMilliseconds := 3600000
	sessionCacheInMinutes := 1

	return &model.Config{
		LocalizationSettings: model.LocalizationSettings{
			DefaultClientLocale: model.NewString(model.DEFAULT_LOCALE),
			DefaultServerLocale: model.NewString(model.DEFAULT_LOCALE),
			AvailableLocales:    model.NewString(model.DEFAULT_LOCALE),
		},
		ServiceSettings: model.ServiceSettings{
			ListenAddress:         model.NewString(":10023"),
			ListenInternalAddress: model.NewString(":10021"),
			SessionCacheInMinutes: &sessionCacheInMinutes,
		},
		SqlSettings: model.SqlSettings{
			DriverName:                  &dbDriverName,
			DataSource:                  &dbDatasource,
			MaxIdleConns:                &maxIdleConns,
			MaxOpenConns:                &maxOpenConns,
			ConnMaxLifetimeMilliseconds: &connMaxLifetimeMilliseconds,
			Trace: true,
		},
		BrokerSettings: model.BrokerSettings{
			ConnectionString: model.NewString("amqp://webitel:secret@10.10.10.200:5672?heartbeat=0"),
		},
	}, "", envConfig, nil
}

func FindPath(path string, baseSearchPaths []string, filter func(os.FileInfo) bool) string {
	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err == nil {
			return path
		}

		return ""
	}

	searchPaths := []string{}
	for _, baseSearchPath := range baseSearchPaths {
		searchPaths = append(searchPaths, baseSearchPath)
	}

	// Additionally attempt to search relative to the location of the running binary.
	var binaryDir string
	if exe, err := os.Executable(); err == nil {
		if exe, err = filepath.EvalSymlinks(exe); err == nil {
			if exe, err = filepath.Abs(exe); err == nil {
				binaryDir = filepath.Dir(exe)
			}
		}
	}
	if binaryDir != "" {
		for _, baseSearchPath := range baseSearchPaths {
			searchPaths = append(
				searchPaths,
				filepath.Join(binaryDir, baseSearchPath),
			)
		}
	}

	for _, parent := range searchPaths {
		found, err := filepath.Abs(filepath.Join(parent, path))
		if err != nil {
			continue
		} else if fileInfo, err := os.Stat(found); err == nil {
			if filter != nil {
				if filter(fileInfo) {
					return found
				}
			} else {
				return found
			}
		}
	}

	return ""
}

// FindDir looks for the given directory in nearby ancestors relative to the current working
// directory as well as the directory of the executable, falling back to `./` if not found.
func FindDir(dir string) (string, bool) {
	found := FindPath(dir, commonBaseSearchPaths, func(fileInfo os.FileInfo) bool {
		return fileInfo.IsDir()
	})
	if found == "" {
		return "./", false
	}

	return found, true
}
