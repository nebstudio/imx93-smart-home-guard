import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'providers/guard_provider.dart';
import 'screens/dashboard_screen.dart';
import 'services/guard_socket_service.dart';
import 'theme/app_theme.dart';

const _deviceHost = String.fromEnvironment('DEVICE_HOST', defaultValue: 'YOUR_DEVICE_HOST');
const _devicePort = String.fromEnvironment('DEVICE_PORT', defaultValue: 'YOUR_DEVICE_PORT');

void main() {
  runApp(const GuardApp());
}

class GuardApp extends StatelessWidget {
  const GuardApp({super.key});

  @override
  Widget build(BuildContext context) {
    final socketService = GuardSocketService('ws://$_deviceHost:$_devicePort/ws/app');

    return ChangeNotifierProvider(
      create: (_) => GuardProvider(socketService),
      child: MaterialApp(
        title: '家庭安全守护',
        theme: AppTheme.light,
        debugShowCheckedModeBanner: false,
        home: const DashboardScreen(),
      ),
    );
  }
}
