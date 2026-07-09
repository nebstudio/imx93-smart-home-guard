class DeviceStatus {
  final String behavior;
  final String env;

  final int distanceCm;
  final bool distanceValid;
  final int smokeAdc;
  final int flameAdc;

  final bool poseAvailable;
  final bool posePerson;
  final String posePosture;

  final String lightColor;
  final bool fanOn;

  final bool windowOpen;
  final bool doorOpen;
  final bool garageOpen;

  final String manualScenario;

  final bool systemEnabled;
  final bool voiceEnabled;

  const DeviceStatus({
    required this.behavior,
    required this.env,
    required this.distanceCm,
    required this.distanceValid,
    required this.smokeAdc,
    required this.flameAdc,
    required this.poseAvailable,
    required this.posePerson,
    required this.posePosture,
    required this.lightColor,
    required this.fanOn,
    required this.windowOpen,
    required this.doorOpen,
    required this.garageOpen,
    required this.manualScenario,
    required this.systemEnabled,
    required this.voiceEnabled,
  });

  factory DeviceStatus.initial() => const DeviceStatus(
        behavior: 'NORMAL',
        env: '',
        distanceCm: 0,
        distanceValid: false,
        smokeAdc: 0,
        flameAdc: 0,
        poseAvailable: false,
        posePerson: false,
        posePosture: '',
        lightColor: 'off',
        fanOn: false,
        windowOpen: false,
        doorOpen: false,
        garageOpen: false,
        manualScenario: '',
        systemEnabled: true,
        voiceEnabled: false,
      );

  factory DeviceStatus.fromJson(Map<String, dynamic> json) {
    return DeviceStatus(
      behavior: json['behavior'] as String? ?? 'NORMAL',
      env: json['env'] as String? ?? '',
      distanceCm: json['distance_cm'] as int? ?? 0,
      distanceValid: json['distance_valid'] as bool? ?? false,
      smokeAdc: json['smoke_adc'] as int? ?? 0,
      flameAdc: json['flame_adc'] as int? ?? 0,
      poseAvailable: json['pose_available'] as bool? ?? false,
      posePerson: json['pose_person'] as bool? ?? false,
      posePosture: json['pose_posture'] as String? ?? '',
      lightColor: json['light_color'] as String? ?? 'off',
      fanOn: json['fan_on'] as bool? ?? false,

      windowOpen: json['window_open'] as bool? ?? false,
      doorOpen: json['door_open'] as bool? ?? false,
      garageOpen: json['garage_open'] as bool? ?? false,
      manualScenario: json['manual_scenario'] as String? ?? '',

      systemEnabled: json['system_enabled'] as bool? ?? true,
      voiceEnabled: json['voice_enabled'] as bool? ?? false,
    );
  }

  bool get hasAlert =>
      systemEnabled &&
      (behavior == 'FALL_ALERT' ||
          behavior == 'STATIC_ALERT' ||
          env == 'FIRE_ALERT' ||
          env == 'SMOKE_ALERT' ||
          env == 'EMERGENCY');

  String get behaviorLabel {
    if (!systemEnabled) return '已暂停';
    switch (behavior) {
      case 'NORMAL':
        return '正常';
      case 'MONITORING':
        return '监测中';
      case 'FALL_ALERT':
        return '疑似跌倒';
      case 'STATIC_ALERT':
        return '长时间静止';
      default:
        return behavior;
    }
  }

  String get envLabel {
    switch (env) {
      case 'FIRE_ALERT':
        return '火焰告警';
      case 'SMOKE_ALERT':
        return '烟雾告警';
      case 'EMERGENCY':
        return '紧急情况';
      default:
        return '';
    }
  }
}
