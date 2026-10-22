extends CanvasLayer

const MAX_MESSAGES := 60
var active: bool = false
var speaker: Node2D
var lines: Array[String] = []
var line_index: int = 0
var panel: PanelContainer
var title_label: Label
var body_label: RichTextLabel
var footer_label: Label
var input_field: LineEdit
var next_button: Button
var send_button: Button
var close_button: Button
var messages: Array[String] = []

func _ready() -> void:
	layer = 20
	panel = PanelContainer.new()
	add_child(panel)
	panel.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_WIDE)
	panel.anchor_left = 0.08
	panel.anchor_right = 0.92
	panel.offset_top = -330
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
	var header := HBoxContainer.new()
	column.add_child(header)
	title_label = Label.new()
	title_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	title_label.add_theme_font_size_override("font_size", 22)
	title_label.add_theme_color_override("font_color", Color("edcf89"))
	header.add_child(title_label)
	next_button = Button.new()
	next_button.text = "听下一句"
	next_button.pressed.connect(advance)
	header.add_child(next_button)
	close_button = Button.new()
	close_button.text = "结束对话"
	close_button.pressed.connect(close)
	header.add_child(close_button)
	body_label = RichTextLabel.new()
	body_label.bbcode_enabled = false
	body_label.add_theme_font_size_override("normal_font_size", 20)
	body_label.custom_minimum_size.y = 140
	body_label.size_flags_vertical = Control.SIZE_EXPAND_FILL
	body_label.scroll_following = true
	body_label.selection_enabled = true
	column.add_child(body_label)
	var row := HBoxContainer.new()
	column.add_child(row)
	input_field = LineEdit.new()
	input_field.placeholder_text = "输入你想说的话，按 Enter 发送…"
	input_field.max_length = 256
	input_field.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	input_field.custom_minimum_size.y = 40
	input_field.add_theme_font_size_override("font_size", 20)
	input_field.text_submitted.connect(send_message)
	row.add_child(input_field)
	send_button = Button.new()
	send_button.text = "发送"
	send_button.custom_minimum_size.x = 80
	send_button.pressed.connect(func(): send_message(input_field.text))
	row.add_child(send_button)
	footer_label = Label.new()
	footer_label.text = "Enter：发送    Esc：结束对话    输入时 E 和方向键用于编辑文字"
	footer_label.add_theme_font_size_override("font_size", 15)
	footer_label.add_theme_color_override("font_color", Color("b2c0cd"))
	column.add_child(footer_label)
	panel.hide()

func _input(event: InputEvent) -> void:
	# Let Escape dismiss an IME candidate first, rather than closing the dialogue.
	if active and event.is_action_pressed("ui_cancel") and not is_ime_composing():
		close()
		get_viewport().set_input_as_handled()

func is_ime_composing() -> bool:
	return DisplayServer.has_feature(DisplayServer.FEATURE_IME) and not DisplayServer.ime_get_text().is_empty()

func open(npc: Node2D) -> void:
	speaker = npc
	lines.assign(npc.dialogue_lines)
	if lines.is_empty():
		lines.append("你好。")
	line_index = 0
	messages.clear()
	active = true
	title_label.text = npc.display_name
	input_field.clear()
	append_message(speaker.display_name, lines[0])
	next_button.disabled = lines.size() <= 1
	panel.show()
	input_field.grab_focus.call_deferred()

func append_message(who: String, message: String) -> void:
	messages.append(who + "：" + message)
	if messages.size() > MAX_MESSAGES:
		messages.pop_front()
	body_label.clear()
	# add_text treats all player input as plain text, including BBCode-like text.
	body_label.add_text("\n\n".join(messages))

func send_message(text: String) -> void:
	if not active or not is_instance_valid(speaker):
		return
	var message := text.strip_edges().left(input_field.max_length)
	if message.is_empty() or is_ime_composing():
		return
	append_message("你", message)
	input_field.clear()
	append_message(speaker.display_name, speaker.reply_to(message))
	input_field.grab_focus()

func advance() -> void:
	if not active or line_index + 1 >= lines.size():
		return
	line_index += 1
	append_message(speaker.display_name, lines[line_index])
	next_button.disabled = line_index + 1 >= lines.size()
	input_field.grab_focus()

func close() -> void:
	active = false
	speaker = null
	input_field.release_focus()
	panel.hide()
