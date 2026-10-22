extends StaticBody2D

@export var display_name: String = "居民"
@export var dialogue_lines: Array[String] = ["你好，欢迎来到小镇。"]
@export_range(8.0, 48.0) var talk_distance: float = 24.0

func _ready() -> void:
	add_to_group("talkable_npcs")
