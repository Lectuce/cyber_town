extends CharacterBody2D

@export var speed: float = 80.0

func _ready() -> void:
	motion_mode = CharacterBody2D.MOTION_MODE_FLOATING

func _physics_process(_delta: float) -> void:
	var previous_position := global_position
	var direction := Input.get_vector("ui_left", "ui_right", "ui_up", "ui_down")
	velocity = direction * speed
	move_and_slide()
	if get_parent().has_method("after_player_move"):
		get_parent().after_player_move(previous_position)
