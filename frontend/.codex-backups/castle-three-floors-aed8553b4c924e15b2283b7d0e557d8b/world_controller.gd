extends Node2D

enum Floor { OUTSIDE, INSIDE, UPSTAIRS }
const MAP_EDGE_LAYER := 128
const INTERACTION_DISTANCE := 11.0

@export var camera_zoom: float = 3.0
@onready var player: CharacterBody2D = $Player
@onready var camera: Camera2D = $Player/Camera2D
@onready var castle: Node2D = $Castle
@onready var ground: TileMapLayer = $Ground/GroundTiles
@onready var ladder_bottom: Marker2D = $Castle/LadderBottom
@onready var ladder_top: Marker2D = $Castle/LadderTop

var current_floor: Floor = Floor.OUTSIDE
var map_rect: Rect2
var hint: Label

func _ready() -> void:
	texture_filter = CanvasItem.TEXTURE_FILTER_NEAREST
	configure_map_bounds()
	configure_camera()
	create_hint()
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
	var previous := castle.to_local(previous_position)
	var current := castle.to_local(player.global_position)
	if current_floor == Floor.OUTSIDE:
		# Only crossing the front threshold inward grants access to the ladder.
		if previous.y >= 0.0 and current.y < 0.0 and absf(current.x) <= 7.1:
			set_floor(Floor.INSIDE)
	elif current_floor == Floor.INSIDE:
		if previous.y < 0.0 and current.y >= 0.0 and absf(current.x) <= 7.1:
			set_floor(Floor.OUTSIDE)
	if Input.is_action_just_pressed("interact"):
		try_use_ladder()
	update_hint()

func try_use_ladder() -> void:
	if current_floor == Floor.INSIDE:
		if player.global_position.distance_to(ladder_bottom.global_position) <= INTERACTION_DISTANCE:
			set_floor(Floor.UPSTAIRS)
			player.global_position = ladder_top.global_position
			player.velocity = Vector2.ZERO
	elif current_floor == Floor.UPSTAIRS:
		if player.global_position.distance_to(ladder_top.global_position) <= INTERACTION_DISTANCE:
			set_floor(Floor.INSIDE)
			player.global_position = ladder_bottom.global_position
			player.velocity = Vector2.ZERO

func set_floor(floor_value: Floor) -> void:
	current_floor = floor_value
	var floor_layer := 1 << int(current_floor)
	player.collision_layer = floor_layer
	player.collision_mask = floor_layer | MAP_EDGE_LAYER
	player.z_index = 1 if current_floor == Floor.OUTSIDE else (3 if current_floor == Floor.INSIDE else 2)
	# Reveal the room beneath the castle facade only after entering its door.
	$Castle/Visuals.modulate.a = 0.3 if current_floor == Floor.INSIDE else 1.0
	$Castle/Interior.visible = current_floor == Floor.INSIDE
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
	match current_floor:
		Floor.OUTSIDE:
			var local := castle.to_local(player.global_position)
			hint.text = "向上走，从正门进入城堡" if absf(local.x) < 22 and local.y >= 0 and local.y < 32 else "方向键移动"
		Floor.INSIDE:
			hint.text = "E：上二楼" if player.global_position.distance_to(ladder_bottom.global_position) <= INTERACTION_DISTANCE else "城堡内：向梯子走，按 E 上楼；向下走出正门"
		Floor.UPSTAIRS:
			hint.text = "E：下楼" if player.global_position.distance_to(ladder_top.global_position) <= INTERACTION_DISTANCE else "城堡二楼：回到梯子口，按 E 下楼"
