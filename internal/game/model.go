package game

import "time"

// 玩家在哪，NPC在哪，哪些场景相邻

type World struct {
	ID       string    `json:"id"`
	GameTime time.Time `json:"game_time"`

	// 世界状态版本号，世界状态发生重要变化时递增，用于判断 Agent 是否基于过期状态进行了规划
	Revision int64 `json:"revision"`
}

type Scene struct {
	WorldID string `json:"world_id"`
	ID      string `json:"id"`
	Name    string `json:"name"`
}

type Player struct {
	WorldID  string `json:"world_id"`
	ID       string `json:"id"`
	SceneID  string `json:"scene_id"`
	Status   string `json:"status"`
	Revision int64  `json:"revision"`
}

type NPC struct {
	WorldID string `json:"world_id"`
	ID      string `json:"id"`
	Name    string `json:"name"`

	// 职业
	Role string `json:"role"`

	// 性格和说法方式
	Persona    string `json:"persona"`
	Background string `json:"background"`

	// NPC当前所在场景
	SceneID         string `json:"scene_id"`
	CurrentActivity string `json:"current_activity"`

	// NPC状态版本
	Revision int64 `json:"revision"`
}

type TownConfig struct {
	World   World    `json:"world"`
	Scenes  []Scene  `json:"scenes"`
	Players []Player `json:"players"`
	NPCs    []NPC    `json:"npcs"`
}
