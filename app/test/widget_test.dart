import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';

import 'package:guard_app/providers/guard_provider.dart';
import 'package:guard_app/screens/dashboard_screen.dart';
import 'package:guard_app/services/guard_socket_service.dart';
import 'package:guard_app/theme/app_theme.dart';

void main() {
  testWidgets('DashboardScreen renders without throwing', (WidgetTester tester) async {

    final socketService = GuardSocketService('ws://127.0.0.1:1/ws/app');

    await tester.pumpWidget(
      ChangeNotifierProvider(
        create: (_) => GuardProvider(socketService),
        child: MaterialApp(
          theme: AppTheme.light,
          home: const DashboardScreen(),
        ),
      ),
    );

    expect(find.text('家庭安全守护'), findsOneWidget);
    expect(find.text('设备离线'), findsOneWidget);
  });
}
