class DeviceEvent {
  final String kind;
  final String message;
  final DateTime receivedAt;

  DeviceEvent({required this.kind, required this.message, DateTime? receivedAt})
      : receivedAt = receivedAt ?? DateTime.now();

  factory DeviceEvent.fromJson(Map<String, dynamic> json) {
    return DeviceEvent(
      kind: json['kind'] as String? ?? '',
      message: json['message'] as String? ?? '',
    );
  }
}

class ChatMessage {
  final String text;
  final bool isUser;
  final DateTime sentAt;

  ChatMessage({required this.text, required this.isUser, DateTime? sentAt})
      : sentAt = sentAt ?? DateTime.now();
}
