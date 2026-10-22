extends CanvasLayer

const MAX_MESSAGES := 60
const QUEST_ID := "missing_iron_cart"
const INPUT_HINT := "Enter：发送    Esc：结束对话    输入时 E 和方向键用于编辑文字"
var stream := preload("res://scripts/web_dialogue_stream.gd").new()
var waiting: bool = false
var request_id: String = ""
var reply_index: int = -1
var reply_text: String = ""
var reply_name: String = ""
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
var quest_panel: HBoxContainer
var quest_label: Label
var quest_button: Button
var quest_request: HTTPRequest
var quest_generation: int = 0
var quest_status: String = ""

func _ready() -> void:
	add_child(stream)
	stream.event_received.connect(_on_stream_event)
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
	quest_panel = HBoxContainer.new()
	column.add_child(quest_panel)
	quest_label = Label.new()
	quest_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	quest_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	quest_label.add_theme_font_size_override("font_size", 16)
	quest_panel.add_child(quest_label)
	quest_button = Button.new()
	quest_button.text = "接受委托"
	quest_button.pressed.connect(_on_quest_pressed)
	quest_panel.add_child(quest_button)
	quest_panel.hide()
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
	footer_label.text = INPUT_HINT
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
	_cancel_request()
	_cancel_quest_request()
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
	footer_label.text = INPUT_HINT
	panel.show()
	quest_panel.visible = str(npc.npc_id) == "merchant"
	panel.offset_top = -420 if quest_panel.visible else -330
	quest_status = ""
	if quest_panel.visible:
		_request_quest(false)
	input_field.grab_focus.call_deferred()

func _on_quest_pressed() -> void:
	# After a failed acceptance, query again: the server may have saved it already.
	_request_quest(quest_status == "not_started")

func _request_quest(accept: bool) -> void:
	if not active or not is_instance_valid(speaker) or str(speaker.npc_id) != "merchant":
		return
	_cancel_quest_request()
	quest_button.disabled = true
	quest_button.text = "提交中…" if accept else "读取中…"
	quest_label.text = "正在接受委托…" if accept else "正在读取委托…"

	# HTTPRequest needs an absolute URL. Web builds use the current website.
	var base_url := str(ProjectSettings.get_setting("quest/base_url", "")).trim_suffix("/")
	if base_url.is_empty():
		if OS.has_feature("web"):
			base_url = str(JavaScriptBridge.eval("window.location.origin"))
		else:
			base_url = "http://127.0.0.1:8080"
	var url := base_url + "/api/quests/" + QUEST_ID
	if accept:
		url += "/accept"

	quest_request = HTTPRequest.new()
	quest_request.timeout = 15.0
	quest_request.body_size_limit = 65536
	add_child(quest_request)
	quest_request.request_completed.connect(
		_on_quest_completed.bind(quest_generation, accept))
	var method := HTTPClient.METHOD_POST if accept else HTTPClient.METHOD_GET
	var err := quest_request.request(url, PackedStringArray(), method)
	if err != OK:
		_cancel_quest_request()
		_quest_failed("无法发起请求，请确认后端地址和网络。")

func _on_quest_completed(
	result: int, response_code: int, _headers: PackedStringArray,
	body: PackedByteArray, generation: int, accept: bool
) -> void:
	if generation != quest_generation:
		return
	_cancel_quest_request()
	if not active or not is_instance_valid(speaker) or str(speaker.npc_id) != "merchant":
		return
	if result != HTTPRequest.RESULT_SUCCESS:
		_quest_failed("委托请求未完成，请重试查询。")
		return
	if response_code != 200:
		_quest_failed("委托请求失败（HTTP %d），请重试查询。" % response_code)
		return
	var data = JSON.parse_string(body.get_string_from_utf8())
	if not data is Dictionary:
		_quest_failed("委托响应格式错误。")
		return
	if data.get("quest_id") != QUEST_ID:
		_quest_failed("返回的委托不匹配。")
		return
	var status := str(data.get("status", ""))
	if not status in ["not_started", "active", "completed"] or (accept and status == "not_started"):
		_quest_failed("返回的委托状态无效。")
		return
	quest_status = status
	quest_button.disabled = status != "not_started"
	match status:
		"not_started":
			quest_label.text = str(data.get("title", "寻找铁矿货车")) + "\n" + str(data.get("description", ""))
			quest_button.text = "接受委托"
		"active":
			quest_label.text = "寻找铁矿货车：调查进行中。"
			quest_button.text = "已接受"
		"completed":
			quest_label.text = "寻找铁矿货车：委托已完成。"
			quest_button.text = "已完成"

func _quest_failed(message: String) -> void:
	quest_status = ""
	quest_label.text = message
	quest_button.text = "重试查询"
	quest_button.disabled = false

func _cancel_quest_request() -> void:
	# Invalidate callbacks even if the player reopens the same merchant.
	quest_generation += 1
	if is_instance_valid(quest_request):
		quest_request.cancel_request()
		quest_request.queue_free()
		quest_request = null

func append_message(who: String, message: String) -> void:
	messages.append(who + "：" + message)
	if messages.size() > MAX_MESSAGES:
		messages.pop_front()
	_render_messages()

func _render_messages() -> void:
	body_label.clear()
	# add_text treats all player input as plain text, including BBCode-like text.
	body_label.add_text("\n\n".join(messages))

func send_message(text: String) -> void:
	if not active or waiting or not is_instance_valid(speaker):
		return
	var message := text.strip_edges().left(input_field.max_length)
	if message.is_empty() or is_ime_composing():
		return
	var npc_id := str(speaker.npc_id).strip_edges()
	if npc_id.is_empty():
		footer_label.text = "这位 NPC 尚未设置 Npc Id，暂时无法发送。"
		return
	append_message("你", message)
	input_field.clear()
	reply_name = speaker.display_name
	reply_text = ""
	append_message(reply_name, "正在思考…")
	reply_index = messages.size() - 1
	request_id = Crypto.new().generate_random_bytes(16).hex_encode()
	_set_waiting(true)
	footer_label.text = "正在等待回复…    Esc：取消并结束对话"
	stream.start(request_id, npc_id, message)

func _on_stream_event(id: String, event: String, data: Dictionary) -> void:
	if not active or not waiting or id != request_id:
		return
	if not is_instance_valid(speaker):
		close()
		return
	match event:
		"delta":
			if not data.get("text") is String:
				_fail_request("回复片段格式错误。")
				return
			reply_text += data["text"]
			_update_reply(reply_text)
			footer_label.text = "正在回复…    Esc：取消并结束对话"
		"done":
			if not data.get("reply") is String:
				_fail_request("完整回复格式错误。")
				return
			reply_text = data["reply"]
			_update_reply(reply_text)
			_finish_request()
			footer_label.text = INPUT_HINT
		"error":
			_fail_request(str(data.get("message", data.get("error", "网络请求失败。"))))

func _update_reply(text: String) -> void:
	if reply_index >= 0 and reply_index < messages.size():
		messages[reply_index] = reply_name + "：" + text
		_render_messages()

func _fail_request(reason: String) -> void:
	_update_reply(reply_text + ("\n" if not reply_text.is_empty() else "") +
		"【本轮未完成】" + reason)
	_finish_request()
	footer_label.text = "本轮未完成，可重新发送或按 Esc 结束对话。"

func _finish_request() -> void:
	# Invalidate before aborting; queued callbacks must not touch the next turn.
	request_id = ""
	stream.cancel()
	_set_waiting(false)
	input_field.grab_focus()

func _set_waiting(value: bool) -> void:
	waiting = value
	send_button.disabled = value
	send_button.text = "回复中…" if value else "发送"
	input_field.editable = not value
	next_button.disabled = value or line_index + 1 >= lines.size()

func _cancel_request() -> void:
	request_id = ""
	stream.cancel()
	reply_index = -1
	reply_text = ""
	_set_waiting(false)

func advance() -> void:
	if not active or waiting or line_index + 1 >= lines.size():
		return
	line_index += 1
	append_message(speaker.display_name, lines[line_index])
	next_button.disabled = line_index + 1 >= lines.size()
	input_field.grab_focus()

func close() -> void:
	_cancel_request()
	_cancel_quest_request()
	active = false
	speaker = null
	input_field.release_focus()
	panel.hide()

func _exit_tree() -> void:
	_cancel_quest_request()
	request_id = ""
	stream.cancel()
