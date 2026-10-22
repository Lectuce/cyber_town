extends Node2D

enum Floor { OUTSIDE, FIRST, SECOND, THIRD }
const MAP_EDGE_LAYER := 128
const INTERACTION_DISTANCE := 11.0

@export var camera_zoom: float = 3.0
@onready var player: CharacterBody2D = $Player
@onready var camera: Camera2D = $Player/Camera2D
@onready var castle: Node2D = $Castle
@onready var ground: TileMapLayer = $Ground/GroundTiles
@onready var stairs_first: Marker2D = $Castle/Floor1/StairsUp
@onready var stairs_second: Marker2D = $Castle/Floor2/StairsDown
@onready var ladder_second: Marker2D = $Castle/Floor2/LadderUp
@onready var ladder_third: Marker2D = $Castle/Floor3/LadderDown
@onready var stairs: Array[Dictionary] = [
	{"from": Floor.FIRST, "to": Floor.SECOND, "entrance": stairs_first, "exit": stairs_second, "label": "上二楼"},
	{"from": Floor.SECOND, "to": Floor.FIRST, "entrance": stairs_second, "exit": stairs_first, "label": "下一楼"},
	{"from": Floor.SECOND, "to": Floor.THIRD, "entrance": ladder_second, "exit": ladder_third, "label": "上三楼"},
	{"from": Floor.THIRD, "to": Floor.SECOND, "entrance": ladder_third, "exit": ladder_second, "label": "下二楼"},
]

var current_floor: Floor = Floor.OUTSIDE
var map_rect: Rect2
var hint: Label
var dialogue

func _ready() -> void:
	texture_filter = CanvasItem.TEXTURE_FILTER_NEAREST
	configure_map_bounds()
	configure_camera()
	create_hint()
	dialogue = preload("res://scripts/dialogue_box.gd").new()
	dialogue.name = "DialogueBox"
	add_child(dialogue)
	set_floor(Floor.OUTSIDE)
	get_viewport().size_changed.connect(update_camera_zoom)

func configure_map_bounds() -> void:
	var cells := ground.get_used_rect()
	var tile_size := Vector2(ground.tile_set.tile_size)
	map_rect = Rect2(ground.to_global(Vector2(cells.position) * tile_size), Vector2(cells.size) * tile_size)
	var edges := StaticBody2D.new()
	edges.name = "MapBounds"
	edges.collision_layer = MAP_EDGE_LAYER
	edges.collision_mask = 0
	add_child(edges)
	# Place the inner faces at the painted ground's four edges.
	var p := map_rect.position
	var s := map_rect.size
	add_boundary(edges, "Left", Rect2(p + Vector2(-32, -32), Vector2(32, s.y + 64)))
	add_boundary(edges, "Right", Rect2(p + Vector2(s.x, -32), Vector2(32, s.y + 64)))
	add_boundary(edges, "Top", Rect2(p + Vector2(0, -32), Vector2(s.x, 32)))
	add_boundary(edges, "Bottom", Rect2(p + Vector2(0, s.y), Vector2(s.x, 32)))

func add_boundary(body: StaticBody2D, node_name: String, rect: Rect2) -> void:
	var collider := CollisionShape2D.new()
	collider.name = node_name
	var shape := RectangleShape2D.new()
	shape.size = rect.size
	collider.shape = shape
	body.add_child(collider)
	collider.global_position = rect.get_center()

func configure_camera() -> void:
	$Camera2D.enabled = false
	camera.enabled = true
	camera.position = Vector2.ZERO
	camera.offset = Vector2.ZERO
	camera.position_smoothing_enabled = false
	camera.limit_smoothed = false
	camera.limit_left = int(map_rect.position.x)
	camera.limit_top = int(map_rect.position.y)
	camera.limit_right = int(map_rect.end.x)
	camera.limit_bottom = int(map_rect.end.y)
	update_camera_zoom()
	camera.make_current()
	camera.force_update_scroll()

func update_camera_zoom() -> void:
	# A larger window must not reveal space beyond the map limits.
	var viewport_size := get_viewport_rect().size
	var fit_zoom := maxf(viewport_size.x / map_rect.size.x, viewport_size.y / map_rect.size.y)
	var value := maxf(camera_zoom, fit_zoom)
	camera.zoom = Vector2.ONE * value
	camera.force_update_scroll()

func after_player_move(previous_position: Vector2) -> void:
	if is_dialogue_open():
		if not is_instance_valid(dialogue.speaker) or Input.is_action_just_pressed("ui_cancel"):
			dialogue.close()
		elif Input.is_action_just_pressed("interact"):
			dialogue.advance()
		update_hint()
		return
	var previous := castle.to_local(previous_position)
	var current := castle.to_local(player.global_position)
	if current_floor == Floor.OUTSIDE:
		# Only crossing the front threshold inward grants access to the ladder.
		if previous.y >= 0.0 and current.y < 0.0 and absf(current.x) <= 7.1:
			set_floor(Floor.FIRST)
	elif current_floor == Floor.FIRST:
		if previous.y < 0.0 and current.y >= 0.0 and absf(current.x) <= 7.1:
			set_floor(Floor.OUTSIDE)
	if Input.is_action_just_pressed("interact"):
		var npc := dialogue_target()
		if npc != null:
			dialogue.open(npc)
			player.velocity = Vector2.ZERO
		else:
			try_use_ladder()
	update_hint()

func is_dialogue_open() -> bool:
	return is_instance_valid(dialogue) and dialogue.active

func dialogue_target() -> StaticBody2D:
	var closest := INF
	var result: StaticBody2D
	var stair := nearby_stair()
	# Compare distances so the ladder remains usable beside the upstairs guard.
	if not stair.is_empty():
		closest = player.global_position.distance_to(stair["entrance"].global_position)
	for npc in get_tree().get_nodes_in_group("talkable_npcs"):
		if not npc.is_visible_in_tree() or (npc.collision_layer & (1 << int(current_floor))) == 0:
			continue
		var distance := player.global_position.distance_to(npc.global_position)
		if distance > npc.talk_distance or distance >= closest:
			continue
		var ray := PhysicsRayQueryParameters2D.create(player.global_position, npc.global_position + Vector2(0, -1), 1 << int(current_floor), [player.get_rid()])
		var hit := get_world_2d().direct_space_state.intersect_ray(ray)
		if not hit.is_empty() and hit["collider"] != npc:
			continue
		closest = distance
		result = npc
	return result

func try_use_ladder() -> void:
	var target := nearby_stair()
	if target.is_empty():
		return
	set_floor(target["to"])
	player.global_position = target["exit"].global_position
	player.velocity = Vector2.ZERO
	update_hint()

func nearby_stair() -> Dictionary:
	var result: Dictionary = {}
	var closest := INTERACTION_DISTANCE
	for stair in stairs:
		if stair["from"] != current_floor:
			continue
		var distance := player.global_position.distance_to(stair["entrance"].global_position)
		if distance <= closest:
			closest = distance
			result = stair
	return result

func set_floor(floor_value: Floor) -> void:
	current_floor = floor_value
	var floor_layer := 1 << int(current_floor)
	player.collision_layer = floor_layer
	player.collision_mask = floor_layer | MAP_EDGE_LAYER
	player.z_index = 1 if current_floor == Floor.OUTSIDE else (2 if current_floor == Floor.THIRD else 3)
	# Reveal the occupied floor beneath the facade; the tower caps remain visible.
	var first := current_floor == Floor.FIRST
	var second := current_floor == Floor.SECOND
	$Castle/Floor1/Interior.visible = first
	$Castle/Floor2/Interior.visible = second
	# The indoor cutaway must not leave fragments of the outside guards visible.
	$Guard.visible = not first
	$Guard2.visible = not first
	# Upper-floor occupants must not show through the interior cutaway.
	# Visibility does not disable their separate third-floor collision layer.
	$Castle/Floor3.visible = not (first or second)
	$Castle/Visuals/UpperPlatform.modulate.a = 0.2 if first or second else 1.0
	$Castle/Visuals/UpperFacade.modulate.a = 0.2 if first or second else 1.0
	$Castle/Visuals/GateFacade.modulate.a = 0.2 if first else 1.0
	$Castle/Visuals/TowerMiddle.modulate.a = 0.2 if first or second else 1.0
	$Castle/Visuals/LowPlatforms.modulate.a = 0.2 if first else 1.0
	$Castle/Visuals/TowerBase.modulate.a = 0.2 if first else 1.0
	update_hint()

func create_hint() -> void:
	var hud := CanvasLayer.new()
	hud.name = "FloorHUD"
	add_child(hud)
	hint = Label.new()
	hint.name = "Hint"
	hud.add_child(hint)
	hint.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_WIDE)
	hint.offset_top = -44
	hint.offset_bottom = -12
	hint.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	hint.mouse_filter = Control.MOUSE_FILTER_IGNORE
	hint.add_theme_font_size_override("font_size", 20)
	hint.add_theme_color_override("font_outline_color", Color.BLACK)
	hint.add_theme_constant_override("outline_size", 5)

func update_hint() -> void:
	if not is_instance_valid(hint):
		return
	hint.visible = not is_dialogue_open()
	if is_dialogue_open():
		return
	var npc := dialogue_target()
	if npc != null:
		hint.text = "E：与%s对话" % npc.display_name
		return
	var target := nearby_stair()
	if not target.is_empty():
		hint.text = "%d 楼 · E：%s" % [int(current_floor), target["label"]]
		return
	match current_floor:
		Floor.OUTSIDE:
			var local := castle.to_local(player.global_position)
			hint.text = "向上走，从正门进入城堡" if absf(local.x) < 22 and local.y >= 0 and local.y < 32 else "方向键移动"
		Floor.FIRST:
			hint.text = "一楼大厅：左右两侧可通行；中央偏左楼梯上二楼，正门出城堡"
		Floor.SECOND:
			hint.text = "二楼：左右通往低平台；左侧楼梯下一楼，中央梯子上三楼"
		Floor.THIRD:
			hint.text = "三楼高台：回到梯子口，按 E 返回二楼"
