package game

import (
	"encoding/json"
	"fmt"

	"os"
)

func LoadTownConfig(path string) (TownConfig, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return TownConfig{}, fmt.Errorf("read town config %v: %v", path, err)
	}

	var config TownConfig

	err = json.Unmarshal(data, &config)
	if err != nil {
		return TownConfig{}, fmt.Errorf("decode town config %v: %v", err, err)
	}

	if config.World.ID == "" {
		return TownConfig{}, fmt.Errorf("town config is required.")
	}

	if len(config.Scenes) == 0 {
		return TownConfig{}, fmt.Errorf("town config requres at least one scene.")
	}

	if len(config.Players) != 1 {
		return TownConfig{}, fmt.Errorf("town config requires exactly one player.")
	}

	if len(config.NPCs) == 0 {
		return TownConfig{}, fmt.Errorf("town config requires at least one NPC.")
	}

	return config, nil

}
