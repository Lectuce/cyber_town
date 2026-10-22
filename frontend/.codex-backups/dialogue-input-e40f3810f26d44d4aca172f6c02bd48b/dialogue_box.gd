extends CanvasLayer

var active: bool = false
var speaker: Node2D
var lines: Array[String] = []
var line_index: int = 0
var panel: PanelContainer
var title_label: Label
var body_label: Label
var footer_label: Label

func _ready() -> void:
	layer = 20
	panel = PanelContainer.new()
	add_child(panel)
	panel.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_WIDE)
	panel.anchor_left = 0.08
	panel.anchor_right = 0.92
	panel.offset_top = -206
	panel.offset_bottom = -24
	var style := StyleBoxFlat.new()
	style.bg_color = Color("202635")
	style.border_color = Color("b2c0cd")
	style.set_border_width_all(2)
	style.content_margin_left = 24
	style.content_margin_right = 24
	style.content_margin_top = 16
	style.content_margin_bottom = 16
	panel.add_theme_stylebox_override("panel", style)
	var column := VBoxContainer.new()
	column.add_theme_constant_override("separation", 10)
	panel.add_child(column)
	title_label = Label.new()
	title_label.add_theme_font_size_override("font_size", 22)
	title_label.add_theme_color_override("font_color", Color("edcf89"))
	column.add_child(title_label)
	body_label = Label.new()
	body_label.add_theme_font_size_override("font_size", 22)
	body_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	body_label.custom_minimum_size.y = 60
	column.add_child(body_label)
	footer_label = Label.new()
	footer_label.add_theme_font_size_override("font_size", 16)
	footer_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_RIGHT
	footer_label.add_theme_color_override("font_color", Color("b2c0cd"))
	column.add_child(footer_label)
	panel.hide()

func open(npc: Node2D) -> void:
	speaker = npc
	lines.assign(npc.dialogue_lines)
	if lines.is_empty():
		lines.append("你好。")
	line_index = 0
	active = true
	title_label.text = npc.display_name
	show_line()
	panel.show()

func advance() -> void:
	if not active:
		return
	line_index += 1
	if line_index >= lines.size():
		close()
	else:
		show_line()

func close() -> void:
	active = false
	speaker = null
	panel.hide()

func show_line() -> void:
	body_label.text = lines[line_index]
	footer_label.text = "%d / %d    E：%s    Esc：结束对话" % [line_index + 1, lines.size(), "结束" if line_index == lines.size() - 1 else "下一句"]
