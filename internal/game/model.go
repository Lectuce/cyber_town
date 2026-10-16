package game

import "time"

// 玩家在哪，NPC在哪，哪些场景相邻

type World struct {
	ID       string
	GameTime time.Time

	// 世界状态版本号，世界状态发生重要变化时递增，用于判断 Agent 是否基于过期状态进行了规划
	Revision int64
}

type Scene struct {
	WorldID string
	ID      string
	Name    string

	// 可以直接移动到的相邻场景
	AdjacentSceneIDs []string
}

type Player struct {
	WorldID  string
	ID       string
	SceneID  string
	Status   string
	Revision int64
}

type NPC struct {
	WorldID string
	ID      string
	Name    string

	// 职业
	Role string

	// 性格和说法方式
	Persona    string
	Background string

	// NPC当前所在场景
	SceneID         string
	CurrentActivity string

	// NPC状态版本
	Revision int64
}

type TownConfig struct {
	World   World    `json:"world"`
	Scenes  []Scene  `json:"scenes"`
	Players []Player `json:"players"`
	NPCs    []NPC    `json:"npcs"`
}
