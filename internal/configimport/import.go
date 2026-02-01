package configimport

import (
	"encoding/json"
	"fmt"
	"log"

	"mteam-rss/internal/config"
	"mteam-rss/internal/database"
)

// ImportConfigSources 从配置文件导入RSS源到数据库
func ImportConfigSources(cfg *config.Config, db *database.Database) error {
	sources := cfg.RSSSources
	if len(sources) == 0 {
		log.Println("配置文件中没有RSS源,跳过导入")
		return nil
	}

	// 检查数据库中是否有RSS源
	existingSources, err := db.GetRSSSources()
	if err != nil {
		return fmt.Errorf("failed to query existing sources: %w", err)
	}

	// 如果数据库中已有RSS源,不重复导入
	if len(existingSources) > 0 {
		log.Printf("数据库中已有 %d 个RSS源,跳过配置文件导入", len(existingSources))
		return nil
	}

	imported := 0
	for _, source := range sources {
		// 序列化过滤规则
		filtersJSON := ""
		if source.Filters != nil {
			if data, err := json.Marshal(source.Filters); err == nil {
				filtersJSON = string(data)
			}
		}

		dbSource := database.RSSSource{
			Name:         source.Name,
			SiteType:     source.SiteType,
			RSSURL:       source.RSSURL,
			Enabled:      source.Enabled,
			PollInterval: source.PollInterval,
			MaxItems:     source.MaxItems,
			Filters:      filtersJSON,
		}

		id, err := db.CreateRSSSource(dbSource)
		if err != nil {
			log.Printf("导入RSS源失败 %s: %v", source.Name, err)
			continue
		}

		log.Printf("导入RSS源成功 %s (ID: %d)", source.Name, id)
		imported++
	}

	log.Printf("从配置文件导入了 %d 个RSS源", imported)
	return nil
}
