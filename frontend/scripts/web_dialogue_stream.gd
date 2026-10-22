extends Node

signal event_received(request_id: String, event: String, data: Dictionary)

const BRIDGE_PATH := "res://scripts/web_dialogue_stream.js"
var _callback: JavaScriptObject
var _client: JavaScriptObject

func _ready() -> void:
	if not OS.has_feature("web"):
		return
	var source := FileAccess.get_file_as_string(BRIDGE_PATH)
	if source.is_empty():
		push_error("Missing browser dialogue bridge: " + BRIDGE_PATH)
		return
	JavaScriptBridge.eval(source, true)
	var factory := JavaScriptBridge.get_interface("TownDialogueStream")
	# Retain this reference for the entire client lifetime.
	_callback = JavaScriptBridge.create_callback(_on_browser_event)
	_client = factory.create(_callback)

func start(request_id: String, npc_id: String, message: String) -> void:
	if _client == null:
		event_received.emit(request_id, "error", {
			"message": "请导出 Web 版本，在浏览器中测试在线对话。" if not OS.has_feature("web") else "浏览器对话组件未加载。"
		})
		return
	var template := str(ProjectSettings.get_setting(
		"dialogue/stream_url", "/api/npcs/{npc_id}/chat/stream"))
	if not "{npc_id}" in template:
		event_received.emit(request_id, "error", {"message": "对话接口地址缺少 {npc_id}。"})
		return
	var timeout_seconds := float(ProjectSettings.get_setting("dialogue/timeout_seconds", 90.0))
	_client.start(template.replace("{npc_id}", npc_id.uri_encode()), JSON.stringify({
		"request_id": request_id,
		"npc_id": npc_id,
		"message": message,
	}), int(clampf(timeout_seconds, 1.0, 600.0) * 1000))

func cancel() -> void:
	if _client != null:
		_client.cancel()

func _on_browser_event(arguments: Array) -> void:
	if arguments.size() == 1:
		# Re-enter Godot UI on its next frame, not inside a browser callback.
		_dispatch.call_deferred(str(arguments[0]))

func _dispatch(payload: String) -> void:
	var event = JSON.parse_string(payload)
	if event is Dictionary and event.get("data") is Dictionary:
		event_received.emit(str(event.get("request_id", "")),
			str(event.get("event", "")), event["data"])

func _exit_tree() -> void:
	cancel()
